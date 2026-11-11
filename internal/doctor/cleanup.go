package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anj/agent-skills-manager/internal/fsutil"
)

// CleanupResult is the outcome of Cleanup.
type CleanupResult struct {
	Removed []string `json:"removed"`
	Failed  []string `json:"failed,omitempty"`
	Skipped []string `json:"skipped,omitempty"`
}

// Cleanup removes only cleanable doctor findings (bak leftovers, broken symlinks).
// Paths are re-validated for safety before any delete.
func Cleanup(issues []Issue) *CleanupResult {
	out := &CleanupResult{}
	seen := map[string]bool{}
	for _, is := range issues {
		if !is.Cleanable() {
			out.Skipped = append(out.Skipped, fmt.Sprintf("%s (not cleanable)", is.Path))
			continue
		}
		p := fsutil.Abs(is.Path)
		if p == "" || p == "/" || seen[p] {
			continue
		}
		seen[p] = true

		switch is.Code {
		case CodeBakLeftover:
			if err := removeBakLeftover(p); err != nil {
				out.Failed = append(out.Failed, fmt.Sprintf("%s: %v", p, err))
				continue
			}
			out.Removed = append(out.Removed, p)
		case CodeBrokenLink:
			if err := removeBrokenLink(p); err != nil {
				out.Failed = append(out.Failed, fmt.Sprintf("%s: %v", p, err))
				continue
			}
			out.Removed = append(out.Removed, p)
		default:
			out.Skipped = append(out.Skipped, p)
		}
	}
	return out
}

func removeBakLeftover(p string) error {
	base := filepath.Base(p)
	if !fsutil.IsASMBackupDir(base) {
		return fmt.Errorf("refusing: not an asm backup dir name")
	}
	// Must look like a real directory under a skills root (…/skills/<name>.bak-…)
	parent := filepath.Base(filepath.Dir(p))
	if parent != "skills" && !strings.HasSuffix(filepath.Dir(p), string(filepath.Separator)+"skills") {
		// allow any …/skills/name.bak-… (wire roots end with /skills)
		if !strings.HasSuffix(filepath.Clean(filepath.Dir(p)), "skills") {
			return fmt.Errorf("refusing: parent is not a skills directory")
		}
	}
	st, err := os.Lstat(p)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("refusing: not a directory")
	}
	return os.RemoveAll(p)
}

func removeBrokenLink(p string) error {
	st, err := os.Lstat(p)
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing: not a symlink")
	}
	// only remove if target is missing
	target, err := os.Readlink(p)
	if err != nil {
		return err
	}
	check := target
	if !filepath.IsAbs(check) {
		check = filepath.Join(filepath.Dir(p), target)
	}
	if _, err := os.Stat(check); !os.IsNotExist(err) {
		return fmt.Errorf("refusing: symlink target is not confirmed missing: %v", err)
	}
	return os.Remove(p)
}
