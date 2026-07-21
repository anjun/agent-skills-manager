package fsutil

import (
	"os"
	"path/filepath"
	"strings"
)

func Home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

func ExpandHome(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		return Home()
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(Home(), p[2:])
	}
	return p
}

func Abs(p string) string {
	p = ExpandHome(p)
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}

// GitWorkTreeRoot walks up from path looking for a .git entry.
// Returns the work tree root and true if found. Stops at filesystem root
// or home (home is never treated as a project root).
func GitWorkTreeRoot(path string) (string, bool) {
	path = Abs(path)
	home := Abs(Home())
	dir := path
	if st, err := os.Stat(dir); err == nil && !st.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		if dir == "" || dir == string(filepath.Separator) {
			return "", false
		}
		if home != "" && dir == home {
			return "", false
		}
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// IsInsideGitWorkTree reports whether path is inside a git work tree
// (directory containing .git, walking up). Home itself is not treated as a project.
func IsInsideGitWorkTree(path string) bool {
	_, ok := GitWorkTreeRoot(path)
	return ok
}

// RelHome shortens an absolute path under $HOME to ~/...
func RelHome(path string) string {
	path = Abs(path)
	home := Abs(Home())
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(filepath.Separator)
	if strings.HasPrefix(path, prefix) {
		return "~/" + path[len(prefix):]
	}
	return path
}

// IsSymlink returns true if path is a symlink.
func IsSymlink(path string) bool {
	st, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeSymlink != 0
}

// ReadLinkAbs returns absolute target of a symlink.
func ReadLinkAbs(path string) (string, error) {
	t, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(t) {
		return filepath.Clean(t), nil
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), t)), nil
}

// EnsureDir creates dir with 0755.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// MoveDir renames src to dst; if cross-device, falls back to copy+remove (simple).
func MoveDir(src, dst string) error {
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// fallback: copy
	if err := copyDir(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode())
	})
}

// SymlinkRel creates a symlink at linkPath pointing to targetPath, using relative path when possible.
func SymlinkRel(targetPath, linkPath string) error {
	if err := EnsureDir(filepath.Dir(linkPath)); err != nil {
		return err
	}
	// remove existing link or empty conflict handled by caller
	rel, err := filepath.Rel(filepath.Dir(linkPath), targetPath)
	if err != nil {
		rel = targetPath
	}
	return os.Symlink(rel, linkPath)
}
