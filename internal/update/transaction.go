package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Plan struct {
	Target       string
	Work         string
	Version      string
	Architecture string
	Hash         string
	Args         []string
	ParentPID    int
}

type Result struct {
	Version string
	Error   string
	// Preserve backups when restoration itself failed.
	RecoveryRequired bool
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err == nil {
		err = out.Sync()
	}
	ce := out.Close()
	if err != nil {
		return err
	}
	return ce
}
func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func readPlan(work string) (*Plan, error) {
	b, err := os.ReadFile(filepath.Join(work, "plan.json"))
	if err != nil {
		return nil, err
	}
	var p Plan
	if err = json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	if !samePath(work, p.Work) {
		return nil, errors.New("update plan directory mismatch")
	}
	if err = p.validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

func samePath(a, b string) bool {
	// Windows may report an existing path using either its long or 8.3 name.
	if resolved, err := filepath.EvalSymlinks(a); err == nil {
		a = resolved
	}
	if resolved, err := filepath.EvalSymlinks(b); err == nil {
		b = resolved
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func rejectSymlinks(path string) error {
	for {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symbolic links are not supported for updates")
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}
func (p *Plan) validate() error {
	if !filepath.IsAbs(p.Work) || !filepath.IsAbs(p.Target) || !samePath(filepath.Dir(p.Work), filepath.Dir(p.Target)) || !strings.HasPrefix(filepath.Base(p.Work), ".squirrel-update-") || !strings.EqualFold(filepath.Ext(p.Target), ".exe") {
		return errors.New("invalid update paths")
	}
	for _, path := range []string{p.Target, p.Work} {
		if err := rejectSymlinks(path); err != nil {
			return err
		}
	}
	if _, err := versionParts(p.Version); err != nil {
		return err
	}
	if p.Architecture != "amd64" && p.Architecture != "arm64" {
		return errors.New("unsupported architecture")
	}
	if p.ParentPID <= 0 {
		return errors.New("invalid parent process")
	}
	return nil
}

// Preserve every marker field, including shortcut ownership. Merely finding a
// marker is insufficient: it must belong to this repository and installation.
func (p *Plan) updatedMarker() ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(p.Target), ".squirrel-install.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var marker map[string]any
	if err = json.Unmarshal(data, &marker); err != nil {
		return nil, err
	}
	dir, _ := marker["directory"].(string)
	if marker["repository"] != Repository || !samePath(dir, filepath.Dir(p.Target)) {
		return nil, errors.New("installation ownership check failed")
	}
	marker["version"] = p.Version
	marker["architecture"] = p.Architecture
	return json.Marshal(marker)
}

type replacement struct {
	name    string
	existed bool
}

// RecoveryError means no usable application could be restarted. The GUI
// launcher must surface this even though its helper has no console window.
type RecoveryError struct{ Work, Reason string }

func (e *RecoveryError) Error() string { return e.Reason }

// apply keeps originals on the same volume until replacement, metadata updates
// and process creation succeed. User data and unknown files are never touched.
func (p *Plan) apply(restart func() error) (result Result) {
	result.Version = p.Version
	var changed []replacement
	var undoMetadata func() error
	rollback := func() error {
		var errs []error
		if undoMetadata != nil {
			errs = append(errs, undoMetadata())
		}
		for i := len(changed) - 1; i >= 0; i-- {
			f := changed[i]
			target := filepath.Join(filepath.Dir(p.Target), f.name)
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				errs = append(errs, err)
				continue
			}
			if f.existed {
				errs = append(errs, os.Rename(filepath.Join(p.Work, "backup", f.name), target))
			}
		}
		return errors.Join(errs...)
	}
	err := func() error {
		if err := p.validate(); err != nil {
			return err
		}
		// Check the archive again in the helper after the app has exited.
		if err := p.unpack(); err != nil {
			return err
		}
		marker, err := p.updatedMarker()
		if err != nil {
			return err
		}
		names := append([]string{}, packageFiles...)
		if marker != nil {
			if err = os.WriteFile(filepath.Join(p.Work, "new", ".squirrel-install.json"), marker, 0600); err != nil {
				return err
			}
			names = append(names, ".squirrel-install.json")
		}
		if err = os.Mkdir(filepath.Join(p.Work, "backup"), 0700); err != nil {
			return err
		}
		for _, name := range names {
			destName := name
			if name == packageFiles[0] {
				destName = filepath.Base(p.Target)
			}
			target := filepath.Join(filepath.Dir(p.Target), destName)
			info, err := os.Lstat(target)
			exists := err == nil
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if exists && !info.Mode().IsRegular() {
				return fmt.Errorf("not a regular installed file: %s", destName)
			}
			if exists {
				if err = os.Rename(target, filepath.Join(p.Work, "backup", destName)); err != nil {
					return err
				}
			}
			changed = append(changed, replacement{destName, exists})
			if err = os.Rename(filepath.Join(p.Work, "new", name), target); err != nil {
				return err
			}
		}
		undoMetadata, err = p.updateRegistration()
		if err != nil {
			return err
		}
		if err = writeJSON(filepath.Join(p.Work, "result.json"), result); err != nil {
			return err
		}
		return restart()
	}()
	if err == nil {
		return result
	}
	result.Error = err.Error()
	if undoErr := rollback(); undoErr != nil {
		result.RecoveryRequired = true
		result.Error += "; restoring backup failed: " + undoErr.Error()
	}
	_ = writeJSON(filepath.Join(p.Work, "result.json"), result)
	return result
}
