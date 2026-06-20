package blockdb

import "time"

// Flags describe the source of a block change.
type Flags uint16

const (
	ManualPlace Flags = 1 << iota
	Painted
	Drawn
	Replaced
	Pasted
	Cut
	Filled
	Restored
	UndoOther
	UndoSelf
	RedoSelf
	FixGrass
)

// Entry records a single block change: who, when, where, old→new.
type Entry struct {
	PlayerID int32     // PlayerDB row ID (0 = server/console)
	Time     time.Time // When the change occurred
	X, Y, Z  int       // Block coordinates
	OldBlock byte      // Block ID before the change
	NewBlock byte      // Block ID after the change
	Flags    Flags     // Source of the change
}

// BlockDB records block changes for a single level.
// All methods are safe for concurrent use.
type BlockDB interface {
	// Add records a block change. Called automatically on every block
	// placement/removal. Buffered in memory and flushed periodically.
	Add(e Entry)

	// ChangesAt returns the full chronological history of changes at
	// the given coordinates, oldest first.
	ChangesAt(x, y, z int) []Entry

	// ChangesBy returns changes by the given player ID in the time
	// window [since, until], newest first. If since is zero, returns
	// all changes. Limited to maxResults (0 = unlimited).
	ChangesBy(playerID int32, since, until time.Time, maxResults int) []Entry

	// Count returns the total number of recorded changes.
	Count() int64

	// Flush writes all buffered entries to disk.
	Flush() error

	// Clear deletes all recorded history (both cache and file).
	Clear() error

	// Enabled reports whether recording is active for this level.
	Enabled() bool

	// SetEnabled toggles recording.
	SetEnabled(bool)
}
