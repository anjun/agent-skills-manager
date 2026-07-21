// Package outdated checks whether skills with known provenance have upstream updates.
// Optional ApplyUpdates (--update) can refresh skill-lock / dws / git skills.
package outdated

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/knownsource"
	"github.com/anj/agent-skills-manager/internal/scan"
	"github.com/anj/agent-skills-manager/internal/skill"
	"github.com/anj/agent-skills-manager/internal/skilllock"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

// Status of one skill's update check.
type Status string

const (
	StatusOutdated Status = "outdated"
	StatusCurrent  Status = "current"
	StatusUnknown  Status = "unknown"
	StatusError    Status = "error"
	StatusSkipped  Status = "skipped" // vendor / no content
)

// Kind of provenance used for the check.
type Provenance string

const (
	ProvNone        Provenance = ""
	ProvGit         Provenance = "git"
	ProvSkillLock   Provenance = "skill-lock" // ~/.agents/.skill-lock.json (npx skills)
	ProvKnown       Provenance = "known"      // built-in fingerprint (e.g. dws CLI skill)
	ProvFrontmatter Provenance = "frontmatter"
	ProvSkillsLock  Provenance = "skills-lock" // project skills-lock.json nearby
)

// Entry is one skill's outdated report line.
type Entry struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Class      scan.Class `json:"class"`
	Tool       string     `json:"tool"`
	Status     Status     `json:"status"`
	Provenance Provenance `json:"provenance,omitempty"`
	LocalRef   string     `json:"local_ref,omitempty"`
	RemoteRef  string     `json:"remote_ref,omitempty"`
	Remote     string     `json:"remote,omitempty"`
	Message    string     `json:"message,omitempty"`
}

// Result is a full outdated report.
type Result struct {
	Entries []Entry `json:"entries"`
	Summary Summary `json:"summary"`
}

// Summary counts.
type Summary struct {
	Outdated int `json:"outdated"`
	Current  int `json:"current"`
	Unknown  int `json:"unknown"`
	Error    int `json:"error"`
	Skipped  int `json:"skipped"`
	Total    int `json:"total"`
}

const gitTimeout = 8 * time.Second

// Run scans skills and checks those with provenance (check-only).
func Run(cfg *config.Config, vl *vendorlist.List) (*Result, error) {
	res, err := scan.Run(cfg, vl)
	if err != nil {
		return nil, err
	}
	out := &Result{}
	seenPath := map[string]bool{} // content path de-dupe
	// name → best entry (prefer canonical / agents path). Multiple real copies
	// of the same skill (e.g. dws skill setup writes dirs into every agent home)
	// must not produce N outdated rows or N update runs.
	byName := map[string]Entry{}
	nameRank := map[string]int{}

	lockIdx := skilllock.LoadIndex()
	treeCache := skilllock.NewTreeCache()
	agentsAbs := fsutil.Abs(cfg.AgentsSkills)

	for _, e := range res.Entries {
		switch e.Class {
		case scan.ClassVendor:
			continue // never check vendor
		case scan.ClassCanonical, scan.ClassUser, scan.ClassProject:
			// ok
		default:
			continue
		}

		contentPath := e.Path
		if e.Symlink && e.Target != "" {
			contentPath = e.Target
		}
		contentPath = fsutil.Abs(contentPath)
		if seenPath[contentPath] {
			continue
		}
		seenPath[contentPath] = true

		ent := checkOne(e, contentPath, lockIdx, treeCache)
		key := strings.ToLower(ent.Name)
		rank := entryPreferRank(ent, agentsAbs)
		if prev, ok := byName[key]; ok {
			if rank >= nameRank[key] {
				continue // keep existing better/equal
			}
			// replace with better path; status re-check already done on this ent
			_ = prev
		}
		byName[key] = ent
		nameRank[key] = rank
	}

	// stable order by name
	names := make([]string, 0, len(byName))
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		out.Entries = append(out.Entries, byName[n])
	}

	for _, ent := range out.Entries {
		out.Summary.Total++
		switch ent.Status {
		case StatusOutdated:
			out.Summary.Outdated++
		case StatusCurrent:
			out.Summary.Current++
		case StatusUnknown:
			out.Summary.Unknown++
		case StatusError:
			out.Summary.Error++
		case StatusSkipped:
			out.Summary.Skipped++
		}
	}
	return out, nil
}

// entryPreferRank: lower is better.
func entryPreferRank(e Entry, agentsAbs string) int {
	p := fsutil.Abs(e.Path)
	if e.Class == scan.ClassCanonical || strings.HasPrefix(p, agentsAbs+string(filepath.Separator)) || p == agentsAbs {
		return 0
	}
	if e.Class == scan.ClassUser {
		return 1
	}
	return 2
}

func checkOne(e scan.Entry, contentPath string, lockIdx *skilllock.Index, trees *skilllock.TreeCache) Entry {
	ent := Entry{
		Name:  e.Name,
		Path:  contentPath,
		Class: e.Class,
		Tool:  e.Tool,
	}

	// 1) skill directory is itself a git work tree with origin (not a parent monorepo)
	contentPath = fsutil.Abs(contentPath)
	if root, ok := fsutil.GitWorkTreeRoot(contentPath); ok && root == contentPath {
		remote, err := gitRemoteURL(root)
		if err == nil && remote != "" {
			ent.Provenance = ProvGit
			ent.Remote = remote
			local, lerr := gitRevParse(root, "HEAD")
			if lerr != nil {
				ent.Status = StatusError
				ent.Message = lerr.Error()
				return ent
			}
			ent.LocalRef = shortSHA(local)
			remoteSHA, branch, rerr := gitLsRemoteHead(root)
			if rerr != nil {
				ent.Status = StatusError
				ent.Message = rerr.Error()
				return ent
			}
			ent.LocalRef = shortSHA(local)
			ent.RemoteRef = shortSHA(remoteSHA)
			if remoteSHA == local {
				ent.Status = StatusCurrent
				ent.Message = "HEAD matches origin/" + branch
			} else {
				ent.Status = StatusOutdated
				ent.Message = "HEAD behind or diverged from origin/" + branch + " (check-only; no pull)"
			}
			return ent
		}
	}

	// 2) global ~/.agents/.skill-lock.json (npx skills) — compare skillFolderHash via GitHub Trees API
	if lockIdx != nil {
		if le, ok := lockIdx.Lookup(e.Name); ok {
			if le.CanCheckGitHub() {
				return checkSkillLockGitHub(ent, le, trees)
			}
			// lock entry exists but not checkable yet
			ent.Provenance = ProvSkillLock
			ent.Remote = le.Source
			if le.SourceURL != "" {
				ent.Remote = le.SourceURL
			}
			ent.LocalRef = shortSHA(firstNonEmpty(le.SkillFolderHash, le.ComputedHash))
			ent.Status = StatusUnknown
			ent.Message = "in " + fsutil.RelHome(le.LockPath) + " but not GitHub-checkable (sourceType=" + le.SourceType + ")"
			return ent
		}
	}

	// 3) known sources not in skill-lock (e.g. dws via dingtalk-workspace-cli installer)
	if ks, ok := knownsource.Match(e.Name, contentPath); ok {
		return checkKnownSource(ent, contentPath, ks, trees)
	}

	// 4) frontmatter source/repository/version — record unknown how to fetch
	if sk, err := skill.Load(contentPath); err == nil {
		src := frontmatterSource(sk)
		if src != "" {
			ent.Provenance = ProvFrontmatter
			ent.Remote = src
			ent.Status = StatusUnknown
			ent.Message = "source declared in SKILL.md; no skill-lock entry to compare"
			return ent
		}
		if ver, ok := sk.Frontmatter["version"].(string); ok && strings.TrimSpace(ver) != "" {
			ent.Provenance = ProvFrontmatter
			ent.LocalRef = strings.TrimSpace(ver)
			ent.Status = StatusUnknown
			ent.Message = "version in frontmatter but no upstream URL"
			return ent
		}
	}

	// 5) project skills-lock.json nearby — detect only (content hash ≠ GitHub tree SHA)
	// Do not treat ~/.agents/.skill-lock.json as "nearby"; that is already handled in step 2.
	if lock := findProjectSkillsLock(contentPath); lock != "" {
		ent.Provenance = ProvSkillsLock
		ent.Status = StatusUnknown
		ent.Message = "project lock at " + fsutil.RelHome(lock) + "; content-hash compare not implemented yet"
		return ent
	}

	ent.Status = StatusUnknown
	ent.Message = "no provenance (manual install / not in skill-lock)"
	return ent
}

// checkKnownSource compares local skill files to GitHub blob SHAs under the known folder.
// Used for installers that copy skills without writing skill-lock (dws, etc.).
func checkKnownSource(ent Entry, contentPath string, ks knownsource.Source, trees *skilllock.TreeCache) Entry {
	ent.Provenance = ProvKnown
	ent.Remote = ks.SourceURL
	if ent.Remote == "" {
		ent.Remote = "https://github.com/" + ks.OwnerRepo
	}
	if trees == nil {
		trees = skilllock.NewTreeCache()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// dws (and similar) ship skills with CLI releases — compare to latest release, not main HEAD.
	ref := ""
	refLabel := "HEAD"
	if ks.PreferRelease {
		if tag, err := skilllock.FetchLatestReleaseTag(ctx, ks.OwnerRepo); err == nil && tag != "" {
			ref = tag
			refLabel = tag
		}
	}

	tree, err := trees.FetchRepoTree(ctx, ks.OwnerRepo, ref)
	if err != nil {
		ent.Status = StatusError
		ent.Message = err.Error()
		return ent
	}
	if refLabel == "HEAD" && tree.Branch != "" {
		refLabel = tree.Branch
	}

	folder := ks.FolderPath()
	remoteBlobs := tree.BlobsUnder(folder)
	if len(remoteBlobs) == 0 {
		ent.Status = StatusError
		ent.Message = "no blobs under " + folder + " on " + ks.OwnerRepo + "@" + refLabel
		return ent
	}

	var (
		checked, mismatched int
		firstLocal, firstRemote string
		firstPath               string
	)
	prefix := folder + "/"
	for repoPath, remoteSHA := range remoteBlobs {
		rel := strings.TrimPrefix(repoPath, prefix)
		if rel == repoPath {
			// exact folder file? shouldn't happen for blobs under folder
			rel = filepath.Base(repoPath)
		}
		localFile := filepath.Join(contentPath, filepath.FromSlash(rel))
		localSHA, err := knownsource.GitBlobSHA1(localFile)
		if err != nil {
			// missing local file that exists upstream
			mismatched++
			if firstPath == "" {
				firstPath = rel
				firstRemote = remoteSHA
			}
			continue
		}
		checked++
		if localSHA != remoteSHA {
			mismatched++
			if firstPath == "" {
				firstPath = rel
				firstLocal = localSHA
				firstRemote = remoteSHA
			}
		}
	}

	// Prefer SKILL.md as the displayed ref when available.
	skillMD := filepath.Join(contentPath, "SKILL.md")
	if localSHA, err := knownsource.GitBlobSHA1(skillMD); err == nil {
		ent.LocalRef = shortSHA(localSHA)
	} else if firstLocal != "" {
		ent.LocalRef = shortSHA(firstLocal)
	}
	remoteSkill := folder + "/SKILL.md"
	if sha, ok := remoteBlobs[remoteSkill]; ok {
		ent.RemoteRef = shortSHA(sha)
	} else if firstRemote != "" {
		ent.RemoteRef = shortSHA(firstRemote)
	}

	if mismatched == 0 {
		ent.Status = StatusCurrent
		ent.Message = fmt.Sprintf("known source %s@%s (%d files match)", ks.OwnerRepo, refLabel, checked)
		return ent
	}
	ent.Status = StatusOutdated
	ent.Message = fmt.Sprintf("known source %s@%s: %d/%d files differ (e.g. %s); %s",
		ks.OwnerRepo, refLabel, mismatched, checked+mismatched, firstPath, ks.OutdatedHint())
	return ent
}

func checkSkillLockGitHub(ent Entry, le skilllock.Entry, trees *skilllock.TreeCache) Entry {
	ent.Provenance = ProvSkillLock
	ent.Remote = le.Source
	if le.SourceURL != "" {
		ent.Remote = le.SourceURL
	}
	ent.LocalRef = shortSHA(le.SkillFolderHash)

	if trees == nil {
		trees = skilllock.NewTreeCache()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	remoteSHA, branch, err := trees.FetchSkillFolderHash(ctx, le)
	if err != nil {
		// Path gone upstream (renamed/deleted) is actionable, not a hard error.
		if strings.Contains(err.Error(), "skill path not found") {
			ent.Status = StatusOutdated
			ent.Message = "upstream path missing on " + le.Source + " (deleted or renamed); lock entry stale"
			return ent
		}
		ent.Status = StatusError
		ent.Message = err.Error()
		return ent
	}
	ent.RemoteRef = shortSHA(remoteSHA)
	if remoteSHA == le.SkillFolderHash {
		ent.Status = StatusCurrent
		ent.Message = "skill-lock hash matches " + le.Source + "@" + branch
	} else {
		ent.Status = StatusOutdated
		ent.Message = "upstream folder changed on " + le.Source + "@" + branch + " (check-only; use npx skills update)"
	}
	return ent
}

func frontmatterSource(sk *skill.Skill) string {
	if sk == nil || sk.Frontmatter == nil {
		return ""
	}
	for _, k := range []string{"source", "repository", "repo", "url", "homepage"} {
		if v, ok := sk.Frontmatter[k].(string); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				return v
			}
		}
	}
	return ""
}

func findProjectSkillsLock(start string) string {
	dir := start
	if st, err := os.Stat(dir); err == nil && !st.IsDir() {
		dir = filepath.Dir(dir)
	}
	home := fsutil.Abs(fsutil.Home())
	// walk up a few levels (skill → package → repo); stop at home
	for i := 0; i < 6; i++ {
		if home != "" && dir == home {
			break
		}
		cand := filepath.Join(dir, "skills-lock.json")
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func gitRemoteURL(repo string) (string, error) {
	return gitOut(repo, "remote", "get-url", "origin")
}

func gitRevParse(repo, rev string) (string, error) {
	return gitOut(repo, "rev-parse", rev)
}

func gitLsRemoteHead(repo string) (sha, branch string, err error) {
	// Prefer origin/HEAD if set
	if out, e := gitOut(repo, "ls-remote", "--symref", "origin", "HEAD"); e == nil && out != "" {
		// lines like: ref: refs/heads/main	HEAD
		//             <sha>	HEAD
		var headSHA string
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "ref:") {
				// ref: refs/heads/main	HEAD
				f := strings.Fields(line)
				if len(f) >= 2 {
					branch = strings.TrimPrefix(f[1], "refs/heads/")
				}
				continue
			}
			f := strings.Fields(line)
			if len(f) >= 1 && len(f[0]) >= 7 {
				headSHA = f[0]
			}
		}
		if headSHA != "" {
			if branch == "" {
				branch = "HEAD"
			}
			return headSHA, branch, nil
		}
	}
	// fallback: common branch names
	for _, b := range []string{"main", "master"} {
		out, e := gitOut(repo, "ls-remote", "origin", "refs/heads/"+b)
		if e != nil || strings.TrimSpace(out) == "" {
			continue
		}
		fields := strings.Fields(out)
		if len(fields) >= 1 {
			return fields[0], b, nil
		}
	}
	// last resort: any heads
	out, e := gitOut(repo, "ls-remote", "origin", "refs/heads/*")
	if e != nil {
		return "", "", e
	}
	fields := strings.Fields(out)
	if len(fields) < 1 {
		return "", "", fmt.Errorf("empty ls-remote")
	}
	return fields[0], "HEAD", nil
}

func gitOut(repo string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func shortSHA(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
