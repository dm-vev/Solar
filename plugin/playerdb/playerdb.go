package playerdb

import "time"

// PlayerEntry holds persistent data for a player that survives disconnects.
// Populated on connect, updated on disconnect, queryable by plugins.
type PlayerEntry struct {
	Name          string        `json:"name"`
	FirstLogin    time.Time     `json:"first_login"`
	LastLogin     time.Time     `json:"last_login"`
	TotalTime     time.Duration `json:"total_time"`
	LoginCount    int           `json:"login_count"`
	IP            string        `json:"ip"`
	LastIP        string        `json:"last_ip"`
	Deaths        int           `json:"deaths"`
	Kicks         int           `json:"kicks"`
	MessagesSent  int           `json:"messages_sent"`
	BlocksPlaced  int64         `json:"blocks_placed"`
	BlocksDeleted int64         `json:"blocks_deleted"`
	Title         string        `json:"title,omitempty"`
	TitleColor    string        `json:"title_color,omitempty"`
	Money         int           `json:"money,omitempty"`

	// Data is an extensible key-value store for plugin-specific fields.
	// Plugins can store arbitrary string data here without schema changes.
	Data map[string]string `json:"data,omitempty"`
}

// PlayerDB provides persistent offline player data storage.
// All methods are safe for concurrent use.
type PlayerDB interface {
	// Get returns the entry for the named player, or nil if not found.
	Get(name string) *PlayerEntry

	// Save creates or updates the entry for a player.
	Save(entry *PlayerEntry)

	// Delete removes the entry for a player. Returns false if not found.
	Delete(name string) bool

	// List returns all entries, sorted by name.
	List() []*PlayerEntry

	// Search returns entries whose name starts with prefix (case-insensitive).
	Search(prefix string) []*PlayerEntry

	// Count returns the total number of entries.
	Count() int

	// Flush writes all pending data to disk.
	Flush() error
}
