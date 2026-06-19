package world

const (
	bootstrapWidth  = 128
	bootstrapHeight = 64
	bootstrapLength = 128

	classicAir   = 0
	classicDirt  = 3
	classicGrass = 2
)

func bootstrapLevel() Level {
	return generateFlatLevel("main", bootstrapWidth, bootstrapHeight, bootstrapLength)
}

func generateFlatLevel(name string, width, height, length int) Level {
	level := normalizeLevel(Level{
		Name:   name,
		Width:  width,
		Height: height,
		Length: length,
		Blocks: make([]byte, width*height*length),
		Spawn: Spawn{
			X:     width / 2,
			Y:     (height * 3) / 4,
			Z:     length / 2,
			Yaw:   0,
			Pitch: 0,
		},
	})

	grassHeight := height / 2
	grassY := grassHeight - 1
	if grassY < 0 {
		grassY = 0
	}

	volumePerLayer := width * length
	for y := 0; y < grassY; y++ {
		off := y * volumePerLayer
		for i := 0; i < volumePerLayer; i++ {
			level.Blocks[off+i] = classicDirt
		}
	}
	if grassY < height {
		off := grassY * volumePerLayer
		for i := 0; i < volumePerLayer; i++ {
			level.Blocks[off+i] = classicGrass
		}
	}
	for i := grassY + 1; i < height; i++ {
		off := i * volumePerLayer
		for j := 0; j < volumePerLayer; j++ {
			level.Blocks[off+j] = classicAir
		}
	}

	return level
}
