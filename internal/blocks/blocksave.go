package blocks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GlobalFile is the JSON filename for global block definitions.
const GlobalFile = "global.json"

// LevelFile returns the JSON filename for a named level's block definitions.
func LevelFile(levelName string) string {
	return "lvl_" + safeLevelName(levelName) + ".json"
}

// LoadGlobal reads global.json from the registry directory.
// A missing file is not an error; the registry stays empty.
func (r *Registry) LoadGlobal() error {
	return r.loadFile(filepath.Join(r.dir, GlobalFile))
}

// SaveGlobal writes all definitions to global.json.
func (r *Registry) SaveGlobal() error {
	return r.saveFile(filepath.Join(r.dir, GlobalFile))
}

// LoadLevel reads a per-level block definition file.
func (r *Registry) LoadLevel(levelName string) error {
	return r.loadFile(filepath.Join(r.dir, LevelFile(levelName)))
}

// SaveLevel writes definitions to a per-level file.
func (r *Registry) SaveLevel(levelName string) error {
	return r.saveFile(filepath.Join(r.dir, LevelFile(levelName)))
}

func (r *Registry) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read block defs %s: %w", path, err)
	}

	var defs []BlockDefinition
	if err := json.Unmarshal(data, &defs); err != nil {
		return fmt.Errorf("parse block defs %s: %w", path, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range defs {
		d := defs[i]
		r.defs[d.ID] = &d
	}
	return nil
}

func safeLevelName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, `/\`) {
		return "_"
	}
	name = filepath.Base(filepath.Clean(name))
	if name == "." || name == ".." {
		return "_"
	}
	return name
}

func (r *Registry) saveFile(path string) error {
	r.mu.RLock()
	defs := make([]BlockDefinition, 0, len(r.defs))
	for _, d := range r.defs {
		defs = append(defs, *d)
	}
	r.mu.RUnlock()

	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create block defs dir: %w", err)
	}

	data, err := json.MarshalIndent(defs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal block defs: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create block defs temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write block defs temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync block defs temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close block defs temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace block defs: %w", err)
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("open block defs dir for sync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync block defs dir: %w", err)
	}
	return nil
}
