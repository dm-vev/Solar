package heightmap

import (
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/solar-mc/solar/internal/generator/core"
)

func TestGenHeightmapLoadsLocalImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "height.png")
	img := image.NewGray(image.Rect(0, 0, 2, 2))
	for z := 0; z < 2; z++ {
		for x := 0; x < 2; x++ {
			img.SetGray(x, z, color.Gray{Y: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close png: %v", err)
	}

	lvl := core.NewLevel("heightmap", 2, 4, 2)
	if err := genHeightmap(&core.Args{Raw: "123 Forest " + path, Biome: core.Forest}, lvl); err != nil {
		t.Fatalf("genHeightmap: %v", err)
	}
	if got := core.GetBlock(lvl, 0, 3, 0); got != core.Grass {
		t.Fatalf("highest heightmap block = %d, want grass", got)
	}
}

func TestGenHeightmapRequiresSource(t *testing.T) {
	lvl := core.NewLevel("heightmap", 2, 4, 2)
	if err := genHeightmap(&core.Args{Raw: "123 Forest", Biome: core.Forest}, lvl); err == nil {
		t.Fatal("genHeightmap accepted missing source")
	}
}

func TestLoadImageHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		img := image.NewGray(image.Rect(0, 0, 1, 1))
		if err := png.Encode(w, img); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	if !isHTTPURL(server.URL) || !isHTTPURL("https://example.test/a.png") || isHTTPURL("/tmp/a.png") {
		t.Fatal("isHTTPURL returned unexpected values")
	}
	if _, err := loadImage(server.URL); err != nil {
		t.Fatalf("loadImage http: %v", err)
	}
}

func TestLoadImageHTTPStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer server.Close()

	if _, err := loadImage(server.URL); err == nil {
		t.Fatal("loadImage accepted non-200 HTTP status")
	}
}

func TestNearestNeighborResize(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	img.SetGray(0, 0, color.Gray{Y: 99})

	resized := nearestNeighborResize(img, 3, 2)
	if resized.Bounds().Dx() != 3 || resized.Bounds().Dy() != 2 {
		t.Fatalf("resized bounds = %v", resized.Bounds())
	}
	if got := color.GrayModel.Convert(resized.At(2, 1)).(color.Gray).Y; got != 99 {
		t.Fatalf("resized pixel = %d, want 99", got)
	}
}
