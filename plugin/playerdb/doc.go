// Package playerdb defines the PlayerDB interface for persistent offline
// player data. Plugins use it to look up player history (login times,
// playtime, IP, stats) and to update custom fields.
//
// The concrete implementation is internal; plugins only see this interface.
package playerdb
