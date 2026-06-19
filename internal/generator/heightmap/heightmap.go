package heightmap

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/solar-mc/solar/internal/generator/core"
)

// Module exposes image-based terrain generation.
var Module = core.Module{
	Name: "heightmap",
	Generators: func() []core.Generator {
		return []core.Generator{{
			Name: "Heightmap",
			Type: core.GenTypeAdvanced,
			Desc: "Seed specifies the URL of a heightmap image",
			Func: genHeightmap,
		}}
	},
}

func genHeightmap(args *core.Args, lvl *core.Level) error {
	fields := strings.Fields(args.Raw)
	var source string
	for _, f := range fields {
		if _, err := strconv.ParseInt(f, 10, 64); err == nil {
			continue
		}
		if _, ok := core.FindBiome(f); ok {
			continue
		}
		if source == "" {
			source = f
		}
	}
	if source == "" {
		return fmt.Errorf("heightmap requires an image URL or path as the seed")
	}

	img, err := loadImage(source)
	if err != nil {
		return fmt.Errorf("load heightmap: %w", err)
	}

	biome := core.BiomeOrDefault(args)
	return applyHeightmap(lvl, img, biome)
}

func loadImage(source string) (image.Image, error) {
	if isHTTPURL(source) {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(source)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		img, _, err := image.Decode(resp.Body)
		return img, err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// Apply renders a heightmap image onto a level using the given biome.
func Apply(lvl *core.Level, img image.Image, biome core.Biome) error {
	return applyHeightmap(lvl, img, biome)
}

func applyHeightmap(lvl *core.Level, img image.Image, biome core.Biome) error {
	width := lvl.Width
	length := lvl.Length
	oneY := width * length

	srcBounds := img.Bounds()
	resized := srcBounds.Dx() != width || srcBounds.Dy() != length
	if resized {
		img = nearestNeighborResize(img, width, length)
	}

	index := 0
	for z := 0; z < length; z++ {
		for x := 0; x < width; x++ {
			height := heightValue(img, x, z)
			layer := biome.Ground
			top := biome.Surface

			if isCliff(height, img, x, z, width, length) {
				layer = biome.Cliff
				top = biome.Cliff
			}

			height = height * lvl.Height / 255
			for y := 0; y < height-1; y++ {
				lvl.Blocks[index+oneY*y] = layer
			}
			if height > 0 {
				lvl.Blocks[index+oneY*(height-1)] = top
			}
			index++
		}
	}
	lvl.Spawn.Y = lvl.Height/2 + 2
	return nil
}

func heightValue(img image.Image, x, z int) int {
	c := color.GrayModel.Convert(img.At(x, z)).(color.Gray)
	return int(c.Y)
}

func isCliff(height int, img image.Image, x, z, width, length int) bool {
	check := func(dx, dz int) bool {
		nx := x + dx
		nz := z + dz
		if nx < 0 || nx >= width || nz < 0 || nz >= length {
			return false
		}
		return height >= heightValue(img, nx, nz)+2
	}
	return check(-1, 0) || check(1, 0) || check(0, -1) || check(0, 1)
}

func nearestNeighborResize(img image.Image, dstWidth, dstHeight int) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	minX := bounds.Min.X
	minY := bounds.Min.Y

	out := image.NewGray(image.Rect(0, 0, dstWidth, dstHeight))
	for y := 0; y < dstHeight; y++ {
		srcY := minY + y*srcHeight/dstHeight
		for x := 0; x < dstWidth; x++ {
			srcX := minX + x*srcWidth/dstWidth
			out.SetGray(x, y, color.GrayModel.Convert(img.At(srcX, srcY)).(color.Gray))
		}
	}
	return out
}

// Unused import guard for side-effect registrations.
var (
	_ = png.Encode
	_ = jpeg.Encode
	_ = gif.Encode
)
