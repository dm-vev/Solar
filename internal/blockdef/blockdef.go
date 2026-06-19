package blockdef

import "math"

// Collide types (matches ClassiCube CPE spec).
const (
	CollideWalkThrough = 0
	CollideSwimThrough = 1
	CollideSolid       = 2
	CollideIce         = 3
	CollideSlipperyIce = 4
	CollideLiquidWater = 5
	CollideLiquidLava  = 6
	CollideClimbRope   = 7
)

// Draw types.
const (
	DrawOpaque           = 0
	DrawTransparent      = 1
	DrawTransparentThick = 2
	DrawTranslucent      = 3
	DrawGas              = 4
	DrawSprite           = 5
)

// Sound types.
const (
	SoundNone   = 0
	SoundWood   = 1
	SoundGravel = 2
	SoundGrass  = 3
	SoundStone  = 4
	SoundMetal  = 5
	SoundGlass  = 6
	SoundCloth  = 7
	SoundSand   = 8
	SoundSnow   = 9
)

// FirstCustomBlock is the lowest ID usable for custom blocks.
// IDs 0-65 are reserved for classic + CPE built-in blocks.
const FirstCustomBlock = 66

// MaxBlockID is the highest block ID supported by the protocol.
const MaxBlockID = 255

// BlockDefinition describes a custom block sent to CPE clients.
type BlockDefinition struct {
	ID             byte    `json:"id"`
	Name           string  `json:"name"`
	CollideType    byte    `json:"collide_type"`
	Speed          float64 `json:"speed"`
	TopTex         byte    `json:"top_tex"`
	BottomTex      byte    `json:"bottom_tex"`
	LeftTex        byte    `json:"left_tex"`
	RightTex       byte    `json:"right_tex"`
	FrontTex       byte    `json:"front_tex"`
	BackTex        byte    `json:"back_tex"`
	BlocksLight    bool    `json:"blocks_light"`
	WalkSound      byte    `json:"walk_sound"`
	FullBright     bool    `json:"full_bright"`
	Shape          byte    `json:"shape"`
	BlockDraw      byte    `json:"block_draw"`
	FallBack       byte    `json:"fallback"`
	FogDensity     byte    `json:"fog_density"`
	FogR           byte    `json:"fog_r"`
	FogG           byte    `json:"fog_g"`
	FogB           byte    `json:"fog_b"`
	MinX           byte    `json:"min_x"`
	MinY           byte    `json:"min_y"`
	MinZ           byte    `json:"min_z"`
	MaxX           byte    `json:"max_x"`
	MaxY           byte    `json:"max_y"`
	MaxZ           byte    `json:"max_z"`
	InventoryOrder int     `json:"inventory_order,omitempty"`
	Brightness     int     `json:"brightness,omitempty"`
}

// Default returns a BlockDefinition with sane defaults for a new custom block.
func Default(id byte) BlockDefinition {
	return BlockDefinition{
		ID:             id,
		Name:           "custom",
		CollideType:    CollideSolid,
		Speed:          1.0,
		TopTex:         0,
		BottomTex:      0,
		LeftTex:        0,
		RightTex:       0,
		FrontTex:       0,
		BackTex:        0,
		BlocksLight:    true,
		WalkSound:      SoundStone,
		FullBright:     false,
		Shape:          16,
		BlockDraw:      DrawOpaque,
		FallBack:       1,
		FogDensity:     0,
		MinX:           0,
		MinY:           0,
		MinZ:           0,
		MaxX:           16,
		MaxY:           16,
		MaxZ:           16,
		InventoryOrder: -1,
		Brightness:     -1,
	}
}

// SetAllTex sets all six textures to the same value.
func (b *BlockDefinition) SetAllTex(tex byte) {
	b.TopTex = tex
	b.BottomTex = tex
	b.LeftTex = tex
	b.RightTex = tex
	b.FrontTex = tex
	b.BackTex = tex
}

// SetSideTex sets the four side textures to the same value.
func (b *BlockDefinition) SetSideTex(tex byte) {
	b.LeftTex = tex
	b.RightTex = tex
	b.FrontTex = tex
	b.BackTex = tex
}

// RawSpeed encodes Speed into the wire format byte.
// Formula: 64 * log2(speed) + 128
func (b BlockDefinition) RawSpeed() byte {
	if b.Speed <= 0 {
		return 128
	}
	return byte(64*math.Log2(b.Speed) + 128)
}

// BrightnessByte encodes Brightness and FullBright into the wire format.
// Bit 7 = modern brightness flag, bit 6 = use lamplight, bits 0-3 = value.
func (b BlockDefinition) BrightnessByte() byte {
	if b.FullBright {
		return 0
	}
	if b.Brightness < 0 {
		return 0
	}
	return byte(b.Brightness) | 0x80
}

// IsSprite reports whether the block is a sprite (shape == 0).
func (b BlockDefinition) IsSprite() bool {
	return b.Shape == 0
}

// CollideTypeName returns a human-readable collide type name.
func CollideTypeName(t byte) string {
	switch t {
	case CollideWalkThrough:
		return "walkthrough"
	case CollideSwimThrough:
		return "swim"
	case CollideSolid:
		return "solid"
	case CollideIce:
		return "ice"
	case CollideSlipperyIce:
		return "slippery_ice"
	case CollideLiquidWater:
		return "water"
	case CollideLiquidLava:
		return "lava"
	case CollideClimbRope:
		return "rope"
	default:
		return "unknown"
	}
}

// DrawTypeName returns a human-readable draw type name.
func DrawTypeName(t byte) string {
	switch t {
	case DrawOpaque:
		return "opaque"
	case DrawTransparent:
		return "transparent"
	case DrawTransparentThick:
		return "transparent_thick"
	case DrawTranslucent:
		return "translucent"
	case DrawGas:
		return "gas"
	case DrawSprite:
		return "sprite"
	default:
		return "unknown"
	}
}

// SoundTypeName returns a human-readable sound type name.
func SoundTypeName(t byte) string {
	switch t {
	case SoundNone:
		return "none"
	case SoundWood:
		return "wood"
	case SoundGravel:
		return "gravel"
	case SoundGrass:
		return "grass"
	case SoundStone:
		return "stone"
	case SoundMetal:
		return "metal"
	case SoundGlass:
		return "glass"
	case SoundCloth:
		return "cloth"
	case SoundSand:
		return "sand"
	case SoundSnow:
		return "snow"
	default:
		return "unknown"
	}
}
