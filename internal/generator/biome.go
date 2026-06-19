package generator

// Biome describes the blocks and environment used by a generator.
type Biome struct {
	Name       string
	Surface    byte
	Ground     byte
	Cliff      byte
	Water      byte
	Bedrock    byte
	BeachSandy byte
	BeachRocky byte
	TreeType   string
}

// TreeDefault returns the preferred tree generator name for this biome, or
// fallback if the biome does not specify a tree type.
func (b Biome) TreeDefault(fallback string) string {
	if b.TreeType == "" {
		return fallback
	}
	return b.TreeType
}

var (
	Forest = Biome{
		Name:       "Forest",
		Surface:    Grass,
		Ground:     Dirt,
		Cliff:      Stone,
		Water:      StillWater,
		Bedrock:    Stone,
		BeachSandy: Sand,
		BeachRocky: Gravel,
		TreeType:   "Classic",
	}

	Arctic = Biome{
		Name:       "Arctic",
		Surface:    WhiteWool,
		Ground:     WhiteWool,
		Cliff:      Stone,
		Water:      StillWater,
		Bedrock:    Stone,
		BeachSandy: WhiteWool,
		BeachRocky: Stone,
	}

	Desert = Biome{
		Name:       "Desert",
		Surface:    Sand,
		Ground:     Sand,
		Cliff:      Gravel,
		Water:      Air,
		Bedrock:    Stone,
		BeachSandy: Sand,
		BeachRocky: Gravel,
		TreeType:   "Cactus",
	}

	Hell = Biome{
		Name:       "Hell",
		Surface:    Obsidian,
		Ground:     Stone,
		Cliff:      Stone,
		Water:      StillLava,
		Bedrock:    Stone,
		BeachSandy: Obsidian,
		BeachRocky: Obsidian,
	}

	Swamp = Biome{
		Name:       "Swamp",
		Surface:    Dirt,
		Ground:     Dirt,
		Cliff:      Stone,
		Water:      StillWater,
		Bedrock:    Stone,
		BeachSandy: Leaves,
		BeachRocky: Dirt,
	}

	Mine = Biome{
		Name:       "Mine",
		Surface:    Gravel,
		Ground:     Cobblestone,
		Cliff:      Stone,
		Water:      StillWater,
		Bedrock:    Bedrock,
		BeachSandy: Stone,
		BeachRocky: Cobblestone,
	}

	Plains = Biome{
		Name:       "Plains",
		Surface:    Grass,
		Ground:     Dirt,
		Cliff:      Stone,
		Water:      Air,
		Bedrock:    Stone,
		BeachSandy: Grass,
		BeachRocky: Grass,
	}

	Sandy = Biome{
		Name:       "Sandy",
		Surface:    Sand,
		Ground:     Sand,
		Cliff:      Gravel,
		Water:      StillWater,
		Bedrock:    Stone,
		BeachSandy: Sand,
		BeachRocky: Gravel,
	}

	Space = Biome{
		Name:       "Space",
		Surface:    Obsidian,
		Ground:     IronBlock,
		Cliff:      IronBlock,
		Water:      Air,
		Bedrock:    Bedrock,
		BeachSandy: Obsidian,
		BeachRocky: Obsidian,
	}
)

var biomes = map[string]Biome{
	"forest": Forest,
	"arctic": Arctic,
	"desert": Desert,
	"hell":   Hell,
	"swamp":  Swamp,
	"mine":   Mine,
	"plains": Plains,
	"sandy":  Sandy,
	"space":  Space,
}

// FindBiome returns a biome by case-insensitive name, defaulting to Forest.
func FindBiome(name string) (Biome, bool) {
	lower := lowerCase(name)
	b, ok := biomes[lower]
	if !ok {
		return Forest, false
	}
	return b, true
}

// BiomeNames returns the names of all registered biomes.
func BiomeNames() []string {
	names := make([]string, 0, len(biomes))
	for name := range biomes {
		names = append(names, name)
	}
	return names
}

func lowerCase(s string) string {
	// ASCII-only lowercase is sufficient for biome names.
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
