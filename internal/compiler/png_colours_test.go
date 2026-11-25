package compiler

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"tesserpack/internal/types"
)

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pixelsEqual(a, b image.Image) bool {
	if a.Bounds().Size() != b.Bounds().Size() {
		return false
	}
	for y := 0; y < a.Bounds().Dy(); y++ {
		for x := 0; x < a.Bounds().Dx(); x++ {
			p := color.NRGBA64Model.Convert(a.At(a.Bounds().Min.X+x, a.Bounds().Min.Y+y))
			q := color.NRGBA64Model.Convert(b.At(b.Bounds().Min.X+x, b.Bounds().Min.Y+y))
			if p != q {
				return false
			}
		}
	}
	return true
}

func compressedPixelsMatch(t *testing.T, data []byte) {
	t.Helper()
	original, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	out, err := CompressPNG(&data, nil, nil, &types.CompilerConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		out = data
	}
	decoded, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if !pixelsEqual(original, decoded) {
		t.Fatal("compressed PNG changed pixels")
	}
}

func TestCompressPNGKeepsMoreThan256Colours(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x * 4), uint8(y * 4), uint8((x * y) & 0xFF), 255})
		}
	}
	compressedPixelsMatch(t, encodePNG(t, img))
}

func TestCompressPNGUsesPaletteWhenLossless(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8((x % 4) * 60), uint8((y % 4) * 60), 0, 255})
		}
	}
	data := encodePNG(t, img)
	compressedPixelsMatch(t, data)
	out, err := CompressPNG(&data, nil, nil, &types.CompilerConfig{})
	if err != nil || out == nil {
		t.Fatalf("expected smaller PNG, got %v", err)
	}
	if decoded, _ := png.Decode(bytes.NewReader(out)); decoded != nil {
		if _, ok := decoded.(*image.Paletted); !ok {
			t.Errorf("expected palette PNG, got %T", decoded)
		}
	}
}
