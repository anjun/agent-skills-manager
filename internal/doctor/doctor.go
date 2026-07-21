package doctor

import (
	"os"
	"path/filepath"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/roots"
	"github.com/anj/agent-skills-manager/internal/scan"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

// Issue found by doctor.
type Issue struct {
	Severity string `json:"severity"` // error|warn|info
	Path     string `json:"path"`
	Message  string `json:"message"`
}

// Run checks broken symlinks and duplicate real copies.
func Run(cfg *config.Config, vl *vendorlist.List) ([]Issue, error) {
	tr := cfg.T()
	var issues []Issue
	res, err := scan.Run(cfg, vl)
	if err != nil {
		return nil, err
	}
	byName := map[string][]scan.Entry{}
	for _, e := range res.Entries {
		if e.Class == scan.ClassVendor || e.Class == scan.ClassProject {
			continue
		}
		byName[e.Name] = append(byName[e.Name], e)
		if e.Symlink {
			if e.Target == "" {
				issues = append(issues, Issue{"error", e.Path, tr.S("doctor.broken_no_target")})
				continue
			}
			if _, err := os.Stat(e.Target); err != nil {
				issues = append(issues, Issue{"error", e.Path, tr.S("doctor.broken_target", e.Target)})
			}
		}
	}
	for name, list := range byName {
		var reals int
		for _, e := range list {
			if !e.Symlink && e.Class == scan.ClassUser {
				reals++
			}
		}
		if reals > 1 {
			issues = append(issues, Issue{"warn", name, tr.S("doctor.dup_reals", reals)})
		}
	}
	// wire roots missing
	for _, wr := range roots.WireRoots(cfg) {
		if _, err := os.Stat(wr.Path); err != nil {
			issues = append(issues, Issue{"info", wr.Path, tr.S("doctor.wire_missing")})
		}
	}
	// agents dir
	if _, err := os.Stat(cfg.AgentsSkills); err != nil {
		issues = append(issues, Issue{"warn", cfg.AgentsSkills, tr.S("doctor.agents_missing")})
	}
	_ = filepath.Separator
	_ = fsutil.Home
	return issues, nil
}
