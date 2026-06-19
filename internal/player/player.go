package player

import "sync"

// Player stores the minimal online state tracked by the server.
type Player struct {
	Name     string
	EntityID uint32
	Spawned  bool
}

// Registry tracks connected players.
type Registry struct {
	mu      sync.RWMutex
	players map[string]Player
	policy  *Policy
}

// NewRegistry creates an empty player registry with an embedded policy store.
func NewRegistry() *Registry {
	return &Registry{
		players: make(map[string]Player),
		policy:  NewPolicy(),
	}
}

// Add registers or refreshes a player in the online snapshot.
func (r *Registry) Add(name string, entityID uint32) bool {
	if name == "" {
		return false
	}

	r.mu.Lock()
	r.players[name] = Player{Name: name, EntityID: entityID}
	r.mu.Unlock()
	return true
}

// MarkSpawned marks the player as spawned in the active world.
func (r *Registry) MarkSpawned(name string) {
	r.mu.Lock()
	if player, ok := r.players[name]; ok {
		player.Spawned = true
		r.players[name] = player
	}
	r.mu.Unlock()
}

// Remove removes a player from the online snapshot.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	delete(r.players, name)
	r.mu.Unlock()
}

// Count returns the current online player count.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.players)
}

// Get returns the tracked player snapshot by name.
func (r *Registry) Get(name string) (Player, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	player, ok := r.players[name]
	return player, ok
}

// OnlineNames returns the currently connected player names.
func (r *Registry) OnlineNames() []string {
	r.mu.RLock()
	names := make([]string, 0, len(r.players))
	for name := range r.players {
		names = append(names, name)
	}
	r.mu.RUnlock()
	sortStrings(names)
	return names
}

// Policy returns the moderation policy store.
func (r *Registry) Policy() *Policy {
	return r.policy
}

// CanJoin reports whether the named player may connect.
func (r *Registry) CanJoin(name string) (bool, string) {
	return r.policy.CanJoin(name)
}

// Ban records or updates a ban entry.
func (r *Registry) Ban(name, reason string) bool {
	return r.policy.Ban(name, reason)
}

// Unban removes a ban entry.
func (r *Registry) Unban(name string) bool {
	return r.policy.Unban(name)
}

// WhitelistAdd inserts a player into the whitelist.
func (r *Registry) WhitelistAdd(name string) bool {
	return r.policy.WhitelistAdd(name)
}

// WhitelistRemove removes a player from the whitelist.
func (r *Registry) WhitelistRemove(name string) bool {
	return r.policy.WhitelistRemove(name)
}

// SetWhitelistEnabled toggles whitelist enforcement.
func (r *Registry) SetWhitelistEnabled(enabled bool) bool {
	return r.policy.SetWhitelistEnabled(enabled)
}

// WhitelistEnabled reports whether whitelist enforcement is active.
func (r *Registry) WhitelistEnabled() bool {
	return r.policy.WhitelistEnabled()
}

// WhitelistNames returns the configured whitelist entries.
func (r *Registry) WhitelistNames() []string {
	return r.policy.WhitelistNames()
}

// BanNames returns the configured ban list.
func (r *Registry) BanNames() []string {
	return r.policy.BanNames()
}

// AddOperators adds operator names to the policy.
func (r *Registry) AddOperators(names ...string) bool {
	return r.policy.AddOperators(names...)
}

// RemoveOperator removes an operator name from the policy.
func (r *Registry) RemoveOperator(name string) bool {
	return r.policy.RemoveOperator(name)
}

// IsOperator reports whether the named player is an operator.
func (r *Registry) IsOperator(name string) bool {
	return r.policy.IsOperator(name)
}

// OperatorNames returns the configured operator names.
func (r *Registry) OperatorNames() []string {
	return r.policy.OperatorNames()
}
