package storage

import "path/filepath"

// LocalStore owns files on disk.
type LocalStore struct {
	root         string
	worldsDir    string
	playersDir   string
	policyFile   string
	worldFileExt string
}

// NewLocalStore creates local file-backed storage with default paths.
func NewLocalStore(root string) *LocalStore {
	return &LocalStore{
		root:         root,
		worldsDir:    "worlds",
		playersDir:   "players",
		policyFile:   "policy.json",
		worldFileExt: ".swld",
	}
}

// Configure sets custom subdirectory and file names.
func (s *LocalStore) Configure(worldsDir, playersDir, policyFile, worldFileExt string) {
	if worldsDir != "" {
		s.worldsDir = worldsDir
	}
	if playersDir != "" {
		s.playersDir = playersDir
	}
	if policyFile != "" {
		s.policyFile = policyFile
	}
	if worldFileExt != "" {
		s.worldFileExt = worldFileExt
	}
}

// WorldsDir returns the directory for persisted worlds.
func (s *LocalStore) WorldsDir() string {
	return filepath.Join(s.root, s.worldsDir)
}

// PlayersDir returns the directory for persisted player policy.
func (s *LocalStore) PlayersDir() string {
	return filepath.Join(s.root, s.playersDir)
}

// WorldFile returns the storage path for a named world snapshot.
func (s *LocalStore) WorldFile(name string) string {
	return filepath.Join(s.WorldsDir(), name+s.worldFileExt)
}

// PlayerPolicyFile returns the persisted whitelist/ban policy path.
func (s *LocalStore) PlayerPolicyFile() string {
	return filepath.Join(s.PlayersDir(), s.policyFile)
}

// BlockDBsDir returns the directory for per-level block change logs.
func (s *LocalStore) BlockDBsDir() string {
	return filepath.Join(s.root, "blockdb")
}

// BlockDBFile returns the path for a level's block change log.
func (s *LocalStore) BlockDBFile(levelName string) string {
	return filepath.Join(s.BlockDBsDir(), levelName+".cbdb")
}
