// Package generator ports MCGalaxy map generators to Solar.
package generator

import (
	"github.com/solar-mc/solar/internal/generator/classic"
	"github.com/solar-mc/solar/internal/generator/core"
	"github.com/solar-mc/solar/internal/generator/fcraft"
	"github.com/solar-mc/solar/internal/generator/heightmap"
	"github.com/solar-mc/solar/internal/generator/simple"
)

// Type aliases keep the public API unchanged.
type (
	Generator = core.Generator
	Args      = core.Args
	Module    = core.Module
	GenType   = core.GenType
	Biome     = core.Biome
	Level     = core.Level
	Spawn     = core.Spawn
	Registry  = core.Registry
	Tree      = core.Tree
	Noise     = core.Noise
)

// Re-exported constants.
const (
	Air           = core.Air
	Stone         = core.Stone
	Grass         = core.Grass
	Dirt          = core.Dirt
	Cobblestone   = core.Cobblestone
	WoodPlank     = core.WoodPlank
	Sapling       = core.Sapling
	Bedrock       = core.Bedrock
	Water         = core.Water
	StillWater    = core.StillWater
	Lava          = core.Lava
	StillLava     = core.StillLava
	Sand          = core.Sand
	Gravel        = core.Gravel
	GoldOre       = core.GoldOre
	IronOre       = core.IronOre
	CoalOre       = core.CoalOre
	Log           = core.Log
	Leaves        = core.Leaves
	Sponge        = core.Sponge
	Glass         = core.Glass
	RedWool       = core.RedWool
	OrangeWool    = core.OrangeWool
	YellowWool    = core.YellowWool
	LimeWool      = core.LimeWool
	GreenWool     = core.GreenWool
	TealWool      = core.TealWool
	AquaWool      = core.AquaWool
	BlueWool      = core.BlueWool
	PurpleWool    = core.PurpleWool
	IndigoWool    = core.IndigoWool
	VioletWool    = core.VioletWool
	MagentaWool   = core.MagentaWool
	PinkWool      = core.PinkWool
	BlackWool     = core.BlackWool
	WhiteWool     = core.WhiteWool
	Dandelion     = core.Dandelion
	Rose          = core.Rose
	BrownMushroom = core.BrownMushroom
	RedMushroom   = core.RedMushroom
	GoldBlock     = core.GoldBlock
	IronBlock     = core.IronBlock
	DoubleSlab    = core.DoubleSlab
	Slab          = core.Slab
	Brick         = core.Brick
	TNT           = core.TNT
	Bookshelf     = core.Bookshelf
	MossyStone    = core.MossyStone
	Obsidian      = core.Obsidian
)

// Re-exported functions.
var (
	ParseArgs           = core.ParseArgs
	Generate            = core.Generate
	NewLevel            = core.NewLevel
	ValidateDimensions  = core.ValidateDimensions
	SetBlock            = core.SetBlock
	GetBlock            = core.GetBlock
	PackIndex           = core.PackIndex
	FillCuboid          = core.FillCuboid
	FillCuboidFn        = core.FillCuboidFn
	FillLayer           = core.FillLayer
	Clamp               = core.Clamp
	Floor               = core.Floor
	Floor32             = core.Floor32
	Round               = core.Round
	Max                 = core.Max
	Min                 = core.Min
	MaxF                = core.MaxF
	MinF                = core.MinF
	Sqr                 = core.Sqr
	Dist3               = core.Dist3
	FindBiome           = core.FindBiome
	BiomeNames          = core.BiomeNames
	FindTree            = core.FindTree
	TreeNames           = core.TreeNames
	RegisterTree        = core.RegisterTree
	NewJavaRandom       = core.NewJavaRandom
	NewNoise            = core.NewNoise
	PerlinNoise         = core.PerlinNoise
	NormalizeFloat      = core.NormalizeFloat
	Marble              = core.Marble
	Invert              = core.Invert
	ApplyBias           = core.ApplyBias
	CalculateSlope      = core.CalculateSlope
	GaussianBlur5X5     = core.GaussianBlur5X5
	FindThresholdSorted = core.FindThresholdSorted
	SortCopy            = core.SortCopy
	NormalizeToRange    = core.NormalizeToRange
	NewRegistry         = core.NewRegistry
)

// Re-exported biomes.
var (
	Forest = core.Forest
	Arctic = core.Arctic
	Desert = core.Desert
	Hell   = core.Hell
	Swamp  = core.Swamp
	Mine   = core.Mine
	Plains = core.Plains
	Sandy  = core.Sandy
	Space  = core.Space
)

// defaultRegistry is the package-level registry initialised by RegisterDefaults.
var defaultRegistry = core.NewRegistry()

// RegisterDefaults populates the default registry with all built-in generator
// modules. It is safe to call multiple times.
func RegisterDefaults() {
	for _, module := range BuiltinModules() {
		defaultRegistry.RegisterModule(module)
	}
}

// BuiltinModules returns the built-in generator families.
func BuiltinModules() []Module {
	return []Module{
		simple.Module,
		classic.Module,
		fcraft.Module,
		heightmap.Module,
	}
}

// DefaultRegistry returns the package-level registry.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Find looks up a generator by case-insensitive name in the default registry.
func Find(name string) (Generator, bool) {
	return defaultRegistry.Find(name)
}

// Names returns all registered generator names grouped by type from the
// default registry.
func Names() map[GenType][]string {
	return defaultRegistry.Names()
}

// AllGenerators returns all registered generators from the default registry.
func AllGenerators() []Generator {
	return defaultRegistry.All()
}
