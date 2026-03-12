package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gvm-tools/gvm/internal/config"
	"github.com/gvm-tools/gvm/internal/fsutil"
)

// Load reads a profile from disk by name.
// Returns an error if the profile doesn't exist.
func Load(name string) (*Profile, error) {
	path, err := profilePath(name)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile '%s' not found — run 'gvm list' to see available profiles or 'gvm add %s' to create one", name, name)
		}
		return nil, fmt.Errorf("reading profile '%s': %w", name, err)
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing profile '%s': %w", name, err)
	}
	return &p, nil
}

// Save writes a profile to disk atomically.
func (p *Profile) Save() error {
	path, err := profilePath(p.Name)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding profile '%s': %w", p.Name, err)
	}
	data = append(data, '\n')

	return fsutil.AtomicWrite(path, data, 0600)
}

// Delete removes a profile from disk.
func Delete(name string) error {
	path, err := profilePath(name)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting profile '%s': %w", name, err)
	}
	return nil
}

// Exists returns true if a profile with the given name exists on disk.
func Exists(name string) (bool, error) {
	path, err := profilePath(name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking profile '%s': %w", name, err)
}

// List returns all saved profiles, sorted by name.
func List() ([]*Profile, error) {
	dir, err := config.ProfilesDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing profiles: %w", err)
	}

	var profiles []*Profile
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		name := entry.Name()[:len(entry.Name())-5] // strip .json
		p, err := Load(name)
		if err != nil {
			continue // skip corrupt profile files
		}
		profiles = append(profiles, p)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// TouchLastUsed updates the last_used timestamp and saves.
func (p *Profile) TouchLastUsed() error {
	p.LastUsed = time.Now().UTC()
	return p.Save()
}

func profilePath(name string) (string, error) {
	dir, err := config.ProfilesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".json"), nil
}
