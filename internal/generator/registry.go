package generator

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

// GenType categorizes generators.
type GenType string

const (
	GenTypeSimple   GenType = "Simple"
	GenTypeFCraft   GenType = "FCraft"
	GenTypeAdvanced GenType = "Advanced"
	GenTypeClassic  GenType = "Classic"
)

// Generator is a named map generator.
type Generator struct {
	Name string
	Type GenType
	Desc string
	Func func(args *Args, lvl *Level) error
}

// Module is a cohesive generator family that can register one or more named
// generators. New generator families should expose a Module instead of adding
// ad-hoc registration logic to the default registry.
type Module struct {
	Name       string
	Generators func() []Generator
}

// Args holds generator parameters and is parsed from command input.
type Args struct {
	Raw          string
	Seed         int
	Biome        Biome
	RandomSeed   bool
	HeightmapURL string
}

// ParseArgs parses a generator argument string. It extracts an integer seed
// and/or a biome name.
func ParseArgs(raw string) (Args, error) {
	args := Args{Raw: raw, Biome: Forest, RandomSeed: true}
	fields := strings.Fields(raw)

	for _, arg := range fields {
		if seed, err := strconv.Atoi(arg); err == nil {
			args.Seed = seed
			args.RandomSeed = false
			continue
		}
		biome, ok := FindBiome(arg)
		if !ok {
			return Args{}, fmt.Errorf("unknown biome %q; available: %s", arg, strings.Join(BiomeNames(), ", "))
		}
		args.Biome = biome
	}

	if args.RandomSeed {
		args.Seed = rand.Int()
	}
	return args, nil
}

// Registry stores named map generators.
type Registry struct {
	mu         sync.RWMutex
	generators map[string]Generator
}

// NewRegistry creates an empty generator registry.
func NewRegistry() *Registry {
	return &Registry{generators: make(map[string]Generator)}
}

// Register adds a generator to the registry.
func (r *Registry) Register(gen Generator) {
	if gen.Name == "" || gen.Func == nil {
		return
	}
	r.mu.Lock()
	r.generators[gen.Name] = gen
	r.mu.Unlock()
}

// RegisterModule adds all generators exposed by a module.
func (r *Registry) RegisterModule(module Module) {
	if module.Generators == nil {
		return
	}
	for _, gen := range module.Generators() {
		r.Register(gen)
	}
}

// Find looks up a generator by case-insensitive name.
func (r *Registry) Find(name string) (Generator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for k, v := range r.generators {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	return Generator{}, false
}

// Names returns all registered generator names grouped by type.
func (r *Registry) Names() map[GenType][]string {
	out := make(map[GenType][]string)
	r.mu.RLock()
	for _, gen := range r.generators {
		out[gen.Type] = append(out[gen.Type], gen.Name)
	}
	r.mu.RUnlock()
	return out
}

// All returns all registered generators.
func (r *Registry) All() []Generator {
	r.mu.RLock()
	out := make([]Generator, 0, len(r.generators))
	for _, gen := range r.generators {
		out = append(out, gen)
	}
	r.mu.RUnlock()
	return out
}

// Generate runs a generator on a new level.
func Generate(gen Generator, name string, width, height, length int, args Args) (*Level, error) {
	if err := ValidateDimensions(width, height, length); err != nil {
		return nil, err
	}
	lvl := NewLevel(name, width, height, length)
	if err := gen.Func(&args, lvl); err != nil {
		return nil, err
	}
	return lvl, nil
}

// defaultRegistry is the package-level registry initialised by RegisterDefaults.
var defaultRegistry = NewRegistry()

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
		SimpleModule,
		ClassicModule,
		FCraftModule,
		HeightmapModule,
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
