// Package blockdb defines the BlockDB interface for per-level block
// change history. Plugins use it to query who placed/broke blocks,
// implement undo/redo, and roll back player edits.
//
// The concrete implementation is internal; plugins only see this interface.
package blockdb
