// ranks.go defines the rank hierarchy for player permissions.
//
// Ranks are ordered by an integer Permission level (-20..120).
// Higher level = more power. Each rank has a name, color, draw limit,
// prefix, and other per-rank settings.
//
// Default ranks (matching MCGalaxy):
//   Banned (-20), Guest (0), Builder (30), AdvBuilder (50),
//   Operator (80), Admin (100), Owner (120)
//
// Rank membership is stored per-player in PlayerDB.Data["rank"].
// Per-command min rank is configurable via the Registry.

package ranks

import (
	"sort"
	"strconv"
	"sync"

	"github.com/solar-mc/solar/plugin/playerdb"
)

// Permission level constants matching MCGalaxy's LevelPermission enum.
const (
	PermBanned     = -20
	PermGuest      = 0
	PermBuilder    = 30
	PermAdvBuilder = 50
	PermOperator   = 80
	PermAdmin      = 100
	PermOwner      = 120
)

// Rank defines a single rank in the hierarchy.
type Rank struct {
	Name       string
	Permission int
	Color      string // MC color code, e.g. "&7"
	DrawLimit  int    // max blocks per drawing command
	Prefix     string // shown before player name in chat
}

// Registry holds all ranks, sorted by permission level.
type Registry struct {
	mu       sync.RWMutex
	ranks    []*Rank
	byName   map[string]*Rank
	byPerm   map[int]*Rank
	playerDB playerdb.PlayerDB
}

// NewRegistry creates a registry with default ranks.
func NewRegistry() *Registry {
	r := &Registry{
		byName: make(map[string]*Rank),
		byPerm: make(map[int]*Rank),
	}
	for _, rank := range DefaultRanks() {
		r.add(rank)
	}
	return r
}

// DefaultRanks returns the built-in rank hierarchy.
func DefaultRanks() []*Rank {
	return []*Rank{
		{Name: "banned", Permission: PermBanned, Color: "&8", DrawLimit: 1},
		{Name: "guest", Permission: PermGuest, Color: "&7", DrawLimit: 1},
		{Name: "builder", Permission: PermBuilder, Color: "&2", DrawLimit: 4096},
		{Name: "advbuilder", Permission: PermAdvBuilder, Color: "&3", DrawLimit: 262144},
		{Name: "operator", Permission: PermOperator, Color: "&c", DrawLimit: 2097152},
		{Name: "admin", Permission: PermAdmin, Color: "&e", DrawLimit: 16777216},
		{Name: "owner", Permission: PermOwner, Color: "&4", DrawLimit: 134217728},
	}
}

func (r *Registry) add(rank *Rank) {
	r.ranks = append(r.ranks, rank)
	r.byName[lower(rank.Name)] = rank
	r.byPerm[rank.Permission] = rank
	sort.Slice(r.ranks, func(i, j int) bool {
		return r.ranks[i].Permission < r.ranks[j].Permission
	})
}

// Get returns the rank with the given name (case-insensitive), or nil.
func (r *Registry) Get(name string) *Rank {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byName[lower(name)]
}

// GetByPerm returns the rank with the given permission level, or nil.
func (r *Registry) GetByPerm(perm int) *Rank {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byPerm[perm]
}

// All returns all ranks sorted by permission level (ascending).
func (r *Registry) All() []*Rank {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Rank, len(r.ranks))
	copy(out, r.ranks)
	return out
}

// DefaultRank returns the guest rank (default for new players).
func (r *Registry) DefaultRank() *Rank {
	return r.GetByPerm(PermGuest)
}

// HasRank checks if a player's permission level meets the minimum.
func HasRank(playerPerm, minPerm int) bool {
	return playerPerm >= minPerm
}

// IsOperator returns true if the permission level is >= operator (80).
func IsOperator(perm int) bool {
	return perm >= PermOperator
}

// GetPlayerRank returns the permission level for a player name.
// Reads from PlayerDB.Data["rank"]. Returns PermGuest if not set.
func (r *Registry) GetPlayerRank(name string) int {
	if r.playerDB == nil {
		return PermGuest
	}
	e := r.playerDB.Get(name)
	if e == nil || e.Data == nil {
		return PermGuest
	}
	val, ok := e.Data["rank"]
	if !ok || val == "" {
		return PermGuest
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return PermGuest
	}
	return n
}

// SetPlayerRank sets the permission level for a player name.
// Persists to PlayerDB.Data["rank"].
func (r *Registry) SetPlayerRank(name string, perm int) bool {
	if r.playerDB == nil {
		return false
	}
	e := r.playerDB.Get(name)
	if e == nil {
		e = &playerdb.PlayerEntry{Name: name}
	}
	if e.Data == nil {
		e.Data = make(map[string]string)
	}
	e.Data["rank"] = strconv.Itoa(perm)
	r.playerDB.Save(e)
	return true
}

// SetPlayerDB wires the PlayerDB for rank persistence.
func (r *Registry) SetPlayerDB(db playerdb.PlayerDB) {
	r.playerDB = db
}

func lower(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		out[i] = c
	}
	return string(out)
}
