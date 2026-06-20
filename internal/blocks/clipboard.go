// copy.go implements the clipboard for copy/paste build commands.
//
// CopyState stores a cuboid region of blocks captured by /copy.
// /paste replays the stored blocks at a new location.
//
// The clipboard is per-player (stored on the session) and persists
// until overwritten by a new /copy.

package blocks

// CopyState holds a captured region of blocks for paste.
type CopyState struct {
	Width  int
	Height int
	Length int
	Blocks []byte // Width*Height*Length, indexed as x + width*(z + length*y)
}

// NewCopyState creates a clipboard for the given dimensions.
func NewCopyState(width, height, length int) *CopyState {
	return &CopyState{
		Width:  width,
		Height: height,
		Length: length,
		Blocks: make([]byte, width*height*length),
	}
}

// Set stores a block at the given local coordinates.
func (c *CopyState) Set(x, y, z int, block byte) {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height || z < 0 || z >= c.Length {
		return
	}
	c.Blocks[x+c.Width*(z+c.Length*y)] = block
}

// Get returns the block at the given local coordinates.
func (c *CopyState) Get(x, y, z int) byte {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height || z < 0 || z >= c.Length {
		return 0
	}
	return c.Blocks[x+c.Width*(z+c.Length*y)]
}

// PasteCallback is called for each block during paste.
type PasteCallback func(x, y, z int, block byte)

// Paste replays the copied blocks at the given world origin.
// If pasteAir is false, air blocks in the clipboard are skipped.
func (c *CopyState) Paste(originX, originY, originZ int, pasteAir bool, cb PasteCallback) {
	for y := 0; y < c.Height; y++ {
		for z := 0; z < c.Length; z++ {
			for x := 0; x < c.Width; x++ {
				b := c.Get(x, y, z)
				if b == 0 && !pasteAir {
					continue
				}
				cb(originX+x, originY+y, originZ+z, b)
			}
		}
	}
}
