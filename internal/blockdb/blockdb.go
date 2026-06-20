// Package blockdb implements a per-level block change history log.
//
// Storage: append-only binary file with 16-byte fixed records.
// In-memory cache buffers writes and is flushed periodically.
// Format: 16-byte header (magic + version + dims) + N × 16-byte entries.
package blockdb

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/solar-mc/solar/plugin/blockdb"
)

const (
	magic     = "CBDB_SOL"
	entrySize = 16
	headerLen = 16
	epoch     = 1262304000 // 2010-01-01 00:00:00 UTC
	bulkRead  = 4096
)

// fileDB is a per-level BlockDB backed by a .cbdb binary file.
type fileDB struct {
	mu        sync.RWMutex
	path      string
	width     int
	height    int
	length    int
	enabled   bool
	cache     []blockdb.Entry
	cacheMu   sync.Mutex
	fileCount int64
}

// New creates a BlockDB for a level with the given dimensions.
func New(path string, width, height, length int) (blockdb.BlockDB, error) {
	db := &fileDB{
		path:    path,
		width:   width,
		height:  height,
		length:  length,
		enabled: true,
	}
	if err := db.loadHeader(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *fileDB) loadHeader() error {
	f, err := os.Open(db.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open blockdb %s: %w", db.path, err)
	}
	defer f.Close()

	header := make([]byte, headerLen)
	if _, err := f.Read(header); err != nil {
		return nil
	}
	if string(header[0:8]) != magic {
		return nil
	}
	db.width = int(binary.BigEndian.Uint16(header[8:10]))
	db.height = int(binary.BigEndian.Uint16(header[10:12]))
	db.length = int(binary.BigEndian.Uint16(header[12:14]))

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat blockdb: %w", err)
	}
	if size := stat.Size(); size > headerLen {
		db.fileCount = (size - headerLen) / entrySize
	}
	return nil
}

func (db *fileDB) Add(e blockdb.Entry) {
	if !db.Enabled() {
		return
	}
	db.cacheMu.Lock()
	db.cache = append(db.cache, e)
	db.cacheMu.Unlock()
}

func (db *fileDB) ChangesAt(x, y, z int) []blockdb.Entry {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []blockdb.Entry

	db.cacheMu.Lock()
	for _, e := range db.cache {
		if e.X == x && e.Y == y && e.Z == z {
			result = append(result, e)
		}
	}
	db.cacheMu.Unlock()

	f, err := os.Open(db.path)
	if err != nil {
		return result
	}
	defer f.Close()

	_, _ = f.Seek(headerLen, 0)
	buf := make([]byte, entrySize*bulkRead)
	for {
		n, err := f.Read(buf)
		for i := 0; i < n/entrySize; i++ {
			ex, ey, ez := db.unpack(binary.BigEndian.Uint32(buf[i*entrySize+8:]))
			if ex == x && ey == y && ez == z {
				result = append(result, db.decodeEntry(buf[i*entrySize:]))
			}
		}
		if err != nil {
			break
		}
	}
	return result
}

func (db *fileDB) ChangesBy(playerID int32, since, until time.Time, maxResults int) []blockdb.Entry {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []blockdb.Entry

	db.cacheMu.Lock()
	for i := len(db.cache) - 1; i >= 0; i-- {
		e := db.cache[i]
		if e.PlayerID != playerID {
			continue
		}
		if !since.IsZero() && e.Time.Before(since) {
			continue
		}
		if !until.IsZero() && e.Time.After(until) {
			continue
		}
		result = append(result, e)
		if maxResults > 0 && len(result) >= maxResults {
			db.cacheMu.Unlock()
			return result
		}
	}
	db.cacheMu.Unlock()

	f, err := os.Open(db.path)
	if err != nil {
		return result
	}
	defer f.Close()

	stat, _ := f.Stat()
	if stat.Size() <= headerLen {
		return result
	}

	total := (stat.Size() - headerLen) / entrySize
	buf := make([]byte, entrySize*bulkRead)

	for chunkStart := total; chunkStart > 0 && (maxResults == 0 || len(result) < maxResults); {
		chunk := int64(bulkRead)
		if chunk > chunkStart {
			chunk = chunkStart
		}
		chunkStart -= chunk
		f.Seek(headerLen+chunkStart*entrySize, 0)
		n, err := f.Read(buf[:chunk*entrySize])
		for i := n/entrySize - 1; i >= 0; i-- {
			e := db.decodeEntry(buf[i*entrySize:])
			if e.PlayerID != playerID {
				continue
			}
			if !since.IsZero() && e.Time.Before(since) {
				return result
			}
			if !until.IsZero() && e.Time.After(until) {
				continue
			}
			result = append(result, e)
			if maxResults > 0 && len(result) >= maxResults {
				return result
			}
		}
		if err != nil {
			break
		}
	}
	return result
}

func (db *fileDB) Count() int64 {
	db.cacheMu.Lock()
	n := int64(len(db.cache))
	db.cacheMu.Unlock()
	return db.fileCount + n
}

func (db *fileDB) Flush() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.cacheMu.Lock()
	cache := db.cache
	db.cache = nil
	db.cacheMu.Unlock()

	if len(cache) == 0 && db.fileCount > 0 {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(db.path), 0o755); err != nil {
		return fmt.Errorf("create blockdb dir: %w", err)
	}

	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if db.fileCount == 0 {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(db.path, flags, 0o644)
	if err != nil {
		return fmt.Errorf("open blockdb %s: %w", db.path, err)
	}
	defer f.Close()

	if db.fileCount == 0 {
		header := make([]byte, headerLen)
		copy(header[0:8], magic)
		binary.BigEndian.PutUint16(header[8:10], uint16(db.width))
		binary.BigEndian.PutUint16(header[10:12], uint16(db.height))
		binary.BigEndian.PutUint16(header[12:14], uint16(db.length))
		if _, err := f.Write(header); err != nil {
			return fmt.Errorf("write blockdb header: %w", err)
		}
	}

	buf := make([]byte, entrySize*bulkRead)
	for written := 0; written < len(cache); {
		end := written + bulkRead
		if end > len(cache) {
			end = len(cache)
		}
		for i, e := range cache[written:end] {
			db.encodeEntry(buf[i*entrySize:], e)
		}
		if _, err := f.Write(buf[:(end-written)*entrySize]); err != nil {
			return fmt.Errorf("write blockdb entries: %w", err)
		}
		written = end
	}
	db.fileCount += int64(len(cache))
	return nil
}

func (db *fileDB) Clear() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.cacheMu.Lock()
	db.cache = nil
	db.cacheMu.Unlock()

	db.fileCount = 0
	if err := os.Remove(db.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove blockdb %s: %w", db.path, err)
	}
	return nil
}

func (db *fileDB) Enabled() bool {
	db.cacheMu.Lock()
	e := db.enabled
	db.cacheMu.Unlock()
	return e
}

func (db *fileDB) SetEnabled(enabled bool) {
	db.cacheMu.Lock()
	db.enabled = enabled
	db.cacheMu.Unlock()
}

func (db *fileDB) pack(x, y, z int) uint32 {
	return uint32(x + db.width*(z+db.length*y))
}

func (db *fileDB) unpack(idx uint32) (x, y, z int) {
	i := int(idx)
	x = i % db.width
	rem := i / db.width
	z = rem % db.length
	y = rem / db.length
	return
}

func (db *fileDB) encodeEntry(buf []byte, e blockdb.Entry) {
	binary.BigEndian.PutUint32(buf[0:4], uint32(e.PlayerID))
	binary.BigEndian.PutUint32(buf[4:8], uint32(e.Time.Unix()-epoch))
	binary.BigEndian.PutUint32(buf[8:12], db.pack(e.X, e.Y, e.Z))
	buf[12] = e.OldBlock
	buf[13] = e.NewBlock
	binary.BigEndian.PutUint16(buf[14:16], uint16(e.Flags))
}

func (db *fileDB) decodeEntry(buf []byte) blockdb.Entry {
	idx := binary.BigEndian.Uint32(buf[8:12])
	x, y, z := db.unpack(idx)
	return blockdb.Entry{
		PlayerID: int32(binary.BigEndian.Uint32(buf[0:4])),
		Time:     time.Unix(int64(binary.BigEndian.Uint32(buf[4:8]))+epoch, 0),
		X:        x, Y: y, Z: z,
		OldBlock: buf[12],
		NewBlock: buf[13],
		Flags:    blockdb.Flags(binary.BigEndian.Uint16(buf[14:16])),
	}
}
