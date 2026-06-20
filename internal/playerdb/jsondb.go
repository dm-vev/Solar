package playerdb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/solar-mc/solar/plugin/playerdb"
)

// jsonDB is an in-memory PlayerDB backed by a single JSON file.
// All entries are loaded into memory at startup and written atomically on Flush.
// ponytail: single-file JSON — fine for hundreds of players; switch to SQLite
// if player count reaches thousands and linear scan becomes measurable.
type jsonDB struct {
	mu      sync.RWMutex
	path    string
	entries map[string]*playerdb.PlayerEntry
}

// New creates a PlayerDB backed by a JSON file at path.
// If the file exists, entries are loaded; otherwise an empty DB is created.
func New(path string) (playerdb.PlayerDB, error) {
	db := &jsonDB{
		path:    path,
		entries: make(map[string]*playerdb.PlayerEntry),
	}
	if err := db.load(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *jsonDB) load() error {
	data, err := os.ReadFile(db.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read playerdb %s: %w", db.path, err)
	}
	if len(data) == 0 {
		return nil
	}
	var raw map[string]*playerdb.PlayerEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode playerdb %s: %w", db.path, err)
	}
	db.mu.Lock()
	db.entries = raw
	if db.entries == nil {
		db.entries = make(map[string]*playerdb.PlayerEntry)
	}
	db.mu.Unlock()
	return nil
}

func (db *jsonDB) Get(name string) *playerdb.PlayerEntry {
	key := normalize(name)
	db.mu.RLock()
	e := db.entries[key]
	db.mu.RUnlock()
	if e == nil {
		return nil
	}
	cp := *e
	return &cp
}

func (db *jsonDB) Save(entry *playerdb.PlayerEntry) {
	if entry == nil || entry.Name == "" {
		return
	}
	key := normalize(entry.Name)
	cp := *entry
	cp.Name = entry.Name
	db.mu.Lock()
	db.entries[key] = &cp
	db.mu.Unlock()
}

func (db *jsonDB) Delete(name string) bool {
	key := normalize(name)
	db.mu.Lock()
	_, ok := db.entries[key]
	if ok {
		delete(db.entries, key)
	}
	db.mu.Unlock()
	return ok
}

func (db *jsonDB) List() []*playerdb.PlayerEntry {
	db.mu.RLock()
	out := make([]*playerdb.PlayerEntry, 0, len(db.entries))
	for _, e := range db.entries {
		cp := *e
		out = append(out, &cp)
	}
	db.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func (db *jsonDB) Search(prefix string) []*playerdb.PlayerEntry {
	p := strings.ToLower(prefix)
	db.mu.RLock()
	var out []*playerdb.PlayerEntry
	for _, e := range db.entries {
		if strings.HasPrefix(strings.ToLower(e.Name), p) {
			cp := *e
			out = append(out, &cp)
		}
	}
	db.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func (db *jsonDB) Count() int {
	db.mu.RLock()
	n := len(db.entries)
	db.mu.RUnlock()
	return n
}

func (db *jsonDB) Flush() error {
	db.mu.RLock()
	snapshot := make(map[string]*playerdb.PlayerEntry, len(db.entries))
	for k, v := range db.entries {
		cp := *v
		snapshot[k] = &cp
	}
	db.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(db.path), 0o755); err != nil {
		return fmt.Errorf("create playerdb dir: %w", err)
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode playerdb: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(db.path), filepath.Base(db.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create playerdb temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write playerdb temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close playerdb temp: %w", err)
	}
	if err := os.Rename(tmpPath, db.path); err != nil {
		return fmt.Errorf("replace playerdb: %w", err)
	}
	return nil
}

func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// EnsureEntry returns the existing entry for name, or creates a new one
// with FirstLogin set to now. Useful for the connect handler.
func EnsureEntry(db playerdb.PlayerDB, name, ip string) *playerdb.PlayerEntry {
	e := db.Get(name)
	if e == nil {
		e = &playerdb.PlayerEntry{
			Name:       name,
			FirstLogin: time.Now(),
		}
	}
	e.LastLogin = time.Now()
	e.LoginCount++
	e.LastIP = e.IP
	e.IP = ip
	db.Save(e)
	return e
}
