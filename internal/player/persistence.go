package player

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type policySnapshot struct {
	WhitelistEnabled bool                   `json:"whitelist_enabled"`
	Bans             []policyEntry          `json:"bans"`
	Whitelist        []policyEntry          `json:"whitelist"`
	Operators        []string               `json:"operators"`
	PlayerProps      map[string]PlayerProps `json:"player_props,omitempty"`
}

// LoadPolicy restores whitelist and ban policy from disk.
func (r *Registry) LoadPolicy(path string) error {
	if r == nil || r.policy == nil {
		return nil
	}
	return r.policy.Load(path)
}

// SavePolicy persists whitelist and ban policy to disk.
func (r *Registry) SavePolicy(path string) error {
	if r == nil || r.policy == nil {
		return nil
	}
	return r.policy.Save(path)
}

// Load reads policy data from disk.
func (p *Policy) Load(path string) error {
	if p == nil {
		return nil
	}
	if path == "" {
		return fmt.Errorf("player policy path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read player policy %s: %w", path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		p.mu.Lock()
		p.whitelistEnabled = false
		p.bans = make(map[string]policyEntry)
		p.whitelist = make(map[string]policyEntry)
		p.operators = make(map[string]string)
		p.props = make(map[string]PlayerProps)
		p.mu.Unlock()
		return nil
	}

	var snapshot policySnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("decode player policy %s: %w", path, err)
	}

	p.mu.Lock()
	p.whitelistEnabled = snapshot.WhitelistEnabled
	p.bans = make(map[string]policyEntry, len(snapshot.Bans))
	for _, entry := range snapshot.Bans {
		key := normalizeName(entry.Name)
		if key == "" {
			continue
		}
		p.bans[key] = policyEntry{Name: strings.TrimSpace(entry.Name), Reason: entry.Reason}
	}
	p.whitelist = make(map[string]policyEntry, len(snapshot.Whitelist))
	for _, entry := range snapshot.Whitelist {
		key := normalizeName(entry.Name)
		if key == "" {
			continue
		}
		p.whitelist[key] = policyEntry{Name: strings.TrimSpace(entry.Name), Reason: entry.Reason}
	}
	p.operators = make(map[string]string, len(snapshot.Operators))
	for _, name := range snapshot.Operators {
		key := normalizeName(name)
		if key == "" {
			continue
		}
		p.operators[key] = strings.TrimSpace(name)
	}
	p.props = make(map[string]PlayerProps, len(snapshot.PlayerProps))
	for name, props := range snapshot.PlayerProps {
		key := normalizeName(name)
		if key == "" {
			continue
		}
		p.props[key] = props
	}
	p.mu.Unlock()
	return nil
}

// Save persists policy data to disk.
func (p *Policy) Save(path string) error {
	if p == nil {
		return nil
	}
	if path == "" {
		return fmt.Errorf("player policy path is empty")
	}

	snapshot := p.snapshot()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create player policy directory for %s: %w", path, err)
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode player policy %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create player policy temp for %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write player policy temp %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close player policy temp %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace player policy %s: %w", path, err)
	}
	return nil
}

func (p *Policy) snapshot() policySnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	snapshot := policySnapshot{WhitelistEnabled: p.whitelistEnabled}
	snapshot.Bans = make([]policyEntry, 0, len(p.bans))
	for _, entry := range p.bans {
		snapshot.Bans = append(snapshot.Bans, entry)
	}
	snapshot.Whitelist = make([]policyEntry, 0, len(p.whitelist))
	for _, entry := range p.whitelist {
		snapshot.Whitelist = append(snapshot.Whitelist, entry)
	}
	snapshot.Operators = make([]string, 0, len(p.operators))
	for _, name := range p.operators {
		snapshot.Operators = append(snapshot.Operators, name)
	}
	if len(p.props) > 0 {
		snapshot.PlayerProps = maps.Clone(p.props)
	}
	sort.Slice(snapshot.Bans, func(i, j int) bool {
		return strings.TrimSpace(snapshot.Bans[i].Name) < strings.TrimSpace(snapshot.Bans[j].Name)
	})
	sort.Slice(snapshot.Whitelist, func(i, j int) bool {
		return strings.TrimSpace(snapshot.Whitelist[i].Name) < strings.TrimSpace(snapshot.Whitelist[j].Name)
	})
	sort.Strings(snapshot.Operators)
	return snapshot
}
