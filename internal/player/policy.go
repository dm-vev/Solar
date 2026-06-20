package player

import (
	"sort"
	"strings"
	"sync"
)

type policyEntry struct {
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func sortStrings(s []string) {
	sort.Strings(s)
}

// PlayerProps stores per-player properties that persist across reconnects.
type PlayerProps struct {
	Color      string `json:"color,omitempty"`
	Model      string `json:"model,omitempty"`
	Frozen     bool   `json:"frozen,omitempty"`
	Muted      bool   `json:"muted,omitempty"`
	AFK        bool   `json:"afk,omitempty"`
	AllowBuild *bool  `json:"allow_build,omitempty"`
}

// Policy stores ban, whitelist, operator, and per-player props state.
type Policy struct {
	mu               sync.RWMutex
	bans             map[string]policyEntry
	whitelist        map[string]policyEntry
	operators        map[string]string
	whitelistEnabled bool
	props            map[string]PlayerProps
}

// NewPolicy creates an empty policy store.
func NewPolicy() *Policy {
	return &Policy{
		bans:      make(map[string]policyEntry),
		whitelist: make(map[string]policyEntry),
		operators: make(map[string]string),
		props:     make(map[string]PlayerProps),
	}
}

// CanJoin reports whether the named player may connect.
func (p *Policy) CanJoin(name string) (bool, string) {
	if p == nil {
		return true, ""
	}

	key := normalizeName(name)
	if key == "" {
		return false, "invalid username"
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if ban, ok := p.bans[key]; ok {
		if ban.Reason != "" {
			return false, ban.Reason
		}
		return false, "banned"
	}
	if _, ok := p.operators[key]; ok {
		return true, ""
	}
	if !p.whitelistEnabled {
		return true, ""
	}
	if _, ok := p.whitelist[key]; ok {
		return true, ""
	}
	return false, "server is whitelisted"
}

// Ban records or updates a ban entry.
func (p *Policy) Ban(name, reason string) bool {
	if p == nil {
		return false
	}

	key := normalizeName(name)
	if key == "" {
		return false
	}
	if reason == "" {
		reason = "banned"
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	_, existed := p.bans[key]
	p.bans[key] = policyEntry{Name: strings.TrimSpace(name), Reason: reason}
	return !existed
}

// Unban removes a ban entry.
func (p *Policy) Unban(name string) bool {
	if p == nil {
		return false
	}

	key := normalizeName(name)
	if key == "" {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	_, existed := p.bans[key]
	delete(p.bans, key)
	return existed
}

// WhitelistAdd inserts a player into the whitelist.
func (p *Policy) WhitelistAdd(name string) bool {
	if p == nil {
		return false
	}

	key := normalizeName(name)
	if key == "" {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	_, existed := p.whitelist[key]
	p.whitelist[key] = policyEntry{Name: strings.TrimSpace(name)}
	return !existed
}

// WhitelistRemove removes a player from the whitelist.
func (p *Policy) WhitelistRemove(name string) bool {
	if p == nil {
		return false
	}

	key := normalizeName(name)
	if key == "" {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	_, existed := p.whitelist[key]
	delete(p.whitelist, key)
	return existed
}

// SetWhitelistEnabled toggles whitelist enforcement.
func (p *Policy) SetWhitelistEnabled(enabled bool) bool {
	if p == nil {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	changed := p.whitelistEnabled != enabled
	p.whitelistEnabled = enabled
	return changed
}

// WhitelistEnabled reports whether whitelist enforcement is active.
func (p *Policy) WhitelistEnabled() bool {
	if p == nil {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.whitelistEnabled
}

// WhitelistNames returns the configured whitelist entries.
func (p *Policy) WhitelistNames() []string {
	if p == nil {
		return nil
	}

	p.mu.RLock()
	names := make([]string, 0, len(p.whitelist))
	for _, entry := range p.whitelist {
		names = append(names, entry.Name)
	}
	p.mu.RUnlock()
	sortStrings(names)
	return names
}

// BanNames returns the configured ban list.
func (p *Policy) BanNames() []string {
	if p == nil {
		return nil
	}

	p.mu.RLock()
	names := make([]string, 0, len(p.bans))
	for _, entry := range p.bans {
		names = append(names, entry.Name)
	}
	p.mu.RUnlock()
	sortStrings(names)
	return names
}

// AddOperators adds operator names to the policy.
func (p *Policy) AddOperators(names ...string) bool {
	if p == nil {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.operators == nil {
		p.operators = make(map[string]string)
	}

	changed := false
	for _, name := range names {
		key := normalizeName(name)
		if key == "" {
			continue
		}

		trimmed := strings.TrimSpace(name)
		if _, ok := p.operators[key]; ok {
			continue
		}
		p.operators[key] = trimmed
		changed = true
	}
	return changed
}

// RemoveOperator removes an operator name from the policy.
func (p *Policy) RemoveOperator(name string) bool {
	if p == nil {
		return false
	}

	key := normalizeName(name)
	if key == "" {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	_, existed := p.operators[key]
	delete(p.operators, key)
	return existed
}

// IsOperator reports whether the named player is an operator.
func (p *Policy) IsOperator(name string) bool {
	if p == nil {
		return false
	}

	key := normalizeName(name)
	if key == "" {
		return false
	}

	p.mu.RLock()
	_, ok := p.operators[key]
	p.mu.RUnlock()
	return ok
}

// OperatorNames returns the configured operator names.
func (p *Policy) OperatorNames() []string {
	if p == nil {
		return nil
	}

	p.mu.RLock()
	names := make([]string, 0, len(p.operators))
	for _, name := range p.operators {
		names = append(names, name)
	}
	p.mu.RUnlock()
	sortStrings(names)
	return names
}

// GetProps returns the stored properties for a player, or zero value.
func (p *Policy) GetProps(name string) PlayerProps {
	if p == nil {
		return PlayerProps{}
	}
	key := normalizeName(name)
	p.mu.RLock()
	props := p.props[key]
	p.mu.RUnlock()
	return props
}

// SetProps updates the stored properties for a player.
func (p *Policy) SetProps(name string, props PlayerProps) {
	if p == nil {
		return
	}
	key := normalizeName(name)
	if key == "" {
		return
	}
	p.mu.Lock()
	p.props[key] = props
	p.mu.Unlock()
}
