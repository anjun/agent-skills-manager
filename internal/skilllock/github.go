package skilllock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	githubAPITimeout = 12 * time.Second
	userAgent        = "agent-skills-manager"
)

// HTTPClient is overridable in tests.
var HTTPClient = &http.Client{Timeout: githubAPITimeout}

// TreeCache caches recursive GitHub trees per owner/repo@ref.
type TreeCache struct {
	mu    sync.Mutex
	trees map[string]*RepoTree // key: owner/repo\0ref
}

// NewTreeCache creates an empty cache.
func NewTreeCache() *TreeCache {
	return &TreeCache{trees: map[string]*RepoTree{}}
}

// RepoTree is a recursive GitHub git tree.
type RepoTree struct {
	SHA    string
	Branch string
	// path → sha for tree entries only
	Folders map[string]string
	// path → sha for blob entries
	Blobs map[string]string
}

type ghTreeResponse struct {
	SHA  string `json:"sha"`
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
	} `json:"tree"`
}

// FetchSkillFolderHash returns the GitHub tree SHA for the skill folder.
// Uses cache so many skills from the same repo share one API call.
func (c *TreeCache) FetchSkillFolderHash(ctx context.Context, e Entry) (remoteSHA string, branch string, err error) {
	or := ownerRepo(e)
	if or == "" {
		return "", "", fmt.Errorf("no owner/repo for %s", e.Name)
	}
	tree, err := c.fetchRepoTree(ctx, or, e.Ref)
	if err != nil {
		return "", "", err
	}
	hash := skillFolderHashFromTree(tree, e.SkillPath)
	if hash == "" {
		return "", tree.Branch, fmt.Errorf("skill path not found in tree: %s", e.SkillPath)
	}
	return hash, tree.Branch, nil
}

func (c *TreeCache) fetchRepoTree(ctx context.Context, ownerRepo, ref string) (*RepoTree, error) {
	key := ownerRepo + "\x00" + ref
	c.mu.Lock()
	if t, ok := c.trees[key]; ok {
		c.mu.Unlock()
		return t, nil
	}
	c.mu.Unlock()

	tree, err := fetchRepoTreeOnce(ctx, ownerRepo, ref, false)
	if err != nil {
		// rate limit / private → retry with token
		if isAuthWorthRetry(err) {
			if tree2, err2 := fetchRepoTreeOnce(ctx, ownerRepo, ref, true); err2 == nil {
				tree, err = tree2, nil
			} else {
				err = err2
			}
		}
	}
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.trees[key] = tree
	c.mu.Unlock()
	return tree, nil
}

type apiError struct {
	status      int
	rateLimited bool
	body        string
}

func (e *apiError) Error() string {
	if e.rateLimited {
		return fmt.Sprintf("GitHub API rate limited (HTTP %d)", e.status)
	}
	return fmt.Sprintf("GitHub API HTTP %d: %s", e.status, e.body)
}

func isAuthWorthRetry(err error) bool {
	ae, ok := err.(*apiError)
	if !ok {
		return false
	}
	return ae.rateLimited || ae.status == 401 || ae.status == 404
}

func fetchRepoTreeOnce(ctx context.Context, ownerRepo, ref string, withToken bool) (*RepoTree, error) {
	branches := []string{}
	if strings.TrimSpace(ref) != "" {
		branches = append(branches, ref)
	} else {
		branches = []string{"HEAD", "main", "master"}
	}

	var token string
	if withToken {
		token = githubToken()
		if token == "" {
			return nil, fmt.Errorf("GitHub token required (set GITHUB_TOKEN/GH_TOKEN or gh auth login)")
		}
	}

	var lastErr error
	for _, br := range branches {
		tree, err := fetchTreeBranch(ctx, ownerRepo, br, token)
		if err == nil {
			return tree, nil
		}
		lastErr = err
		if ae, ok := err.(*apiError); ok && ae.rateLimited {
			break // same bucket for all branches
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("empty tree for %s", ownerRepo)
	}
	return nil, lastErr
}

func fetchTreeBranch(ctx context.Context, ownerRepo, branch, token string) (*RepoTree, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", ownerRepo, branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", userAgent)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	if resp.StatusCode != http.StatusOK {
		rl := resp.StatusCode == 403 && resp.Header.Get("X-RateLimit-Remaining") == "0"
		msg := strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, &apiError{status: resp.StatusCode, rateLimited: rl, body: msg}
	}

	var raw ghTreeResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	tree := &RepoTree{
		SHA:     raw.SHA,
		Branch:  branch,
		Folders: map[string]string{},
		Blobs:   map[string]string{},
	}
	for _, e := range raw.Tree {
		if e.Path == "" {
			continue
		}
		switch e.Type {
		case "tree":
			tree.Folders[e.Path] = e.SHA
		case "blob":
			tree.Blobs[e.Path] = e.SHA
		}
	}
	return tree, nil
}

// FetchRepoTree returns (and caches) the recursive tree for owner/repo.
func (c *TreeCache) FetchRepoTree(ctx context.Context, ownerRepo, ref string) (*RepoTree, error) {
	return c.fetchRepoTree(ctx, ownerRepo, ref)
}

// BlobsUnder returns blob path→sha under folder (inclusive prefix).
// folder is e.g. "skills/mono"; paths are full repo paths.
func (t *RepoTree) BlobsUnder(folder string) map[string]string {
	if t == nil {
		return nil
	}
	folder = strings.Trim(strings.ReplaceAll(folder, "\\", "/"), "/")
	out := map[string]string{}
	prefix := folder + "/"
	for p, sha := range t.Blobs {
		if p == folder || strings.HasPrefix(p, prefix) {
			out[p] = sha
		}
	}
	return out
}

// skillFolderHashFromTree mirrors vercel-labs/skills getSkillFolderHashFromTree.
func skillFolderHashFromTree(tree *RepoTree, skillPath string) string {
	if tree == nil {
		return ""
	}
	folderPath := strings.ReplaceAll(skillPath, "\\", "/")
	lower := strings.ToLower(folderPath)
	if strings.HasSuffix(lower, "/skill.md") {
		folderPath = folderPath[:len(folderPath)-len("/skill.md")]
	} else if strings.HasSuffix(lower, "skill.md") {
		folderPath = folderPath[:len(folderPath)-len("skill.md")]
	}
	folderPath = strings.TrimSuffix(folderPath, "/")
	if folderPath == "" {
		return tree.SHA
	}
	if sha, ok := tree.Folders[folderPath]; ok {
		return sha
	}
	return ""
}

func githubToken() string {
	if t := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); t != "" {
		return t
	}
	if t := strings.TrimSpace(os.Getenv("GH_TOKEN")); t != "" {
		return t
	}
	// best-effort; may fail in restricted environments
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
