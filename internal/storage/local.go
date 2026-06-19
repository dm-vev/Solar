package storage

import "path/filepath"

// LocalStore owns files on disk.
type LocalStore struct {
	root string
}

// NewLocalStore creates local file-backed storage.
func NewLocalStore(root string) *LocalStore {
	return &LocalStore{root: root}
}

// WorldsDir returns the directory for persisted worlds.
func (s *LocalStore) WorldsDir() string {
	return filepath.Join(s.root, "worlds")
}

// PlayersDir returns the directory for persisted player policy.
func (s *LocalStore) PlayersDir() string {
	return filepath.Join(s.root, "players")
}

// WorldFile returns the storage path for a named world snapshot.
func (s *LocalStore) WorldFile(name string) string {
	return filepath.Join(s.WorldsDir(), name+".json")
}

// PlayerPolicyFile returns the persisted whitelist/ban policy path.
func (s *LocalStore) PlayerPolicyFile() string {
	return filepath.Join(s.PlayersDir(), "policy.json")
}
