package compiler

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"tesserpack/internal/types"

	"github.com/cshum/vipsgen/vips"
	"github.com/marusama/semaphore/v2"
)

// todo for tuxebro: compress assets concurrently without exploding the PC (TBD by above v1)
// I tried it with a simple go func() and wait group, but my PC will crap itself
// we're going to leave it synchronized first - tuxebro, 2025

var pngOptions = &vips.PngsaveBufferOptions{
	Q: 100,
	Compression: 9,
	Interlace: false,
	Palette: true,
	Keep: vips.KeepNone,
	Effort: 10,
}

var jpgOptions = &vips.JpegsaveBufferOptions{
	Interlace: false,
	OptimizeCoding: true,
	OptimizeScans: true,
}

func SetPngOptions(config *types.PNGConfig) {
	pngOptions.Q = int(config.Q)
	pngOptions.Compression = int(config.Compression)
	pngOptions.Effort = int(config.Effort)
}

func SetJpgOptions(config *types.JPGConfig) {
	jpgOptions.Q = int(config.Q)
}

var sem = func () semaphore.Semaphore {
	// only have one thread not doing anything by default (this is temporary)
	numCPU := runtime.NumCPU() - 1

	if (numCPU <= 0) {
		numCPU = 1;
	}
	
	return semaphore.New(numCPU)	
}()

var CompressPNG types.ProcessorFunc = func(data *[]byte, outFile *string, srcFile *string, conf *types.CompilerConfig) (processedData []byte, err error) {
	sem.Acquire(context.Background(), 1)
	defer sem.Release(1)
	
	img, err := vips.NewPngloadBuffer(*data, nil)
	if err != nil {
		return nil, err
	}
	defer img.Close()


	original, err := png.Decode(bytes.NewReader(*data))
	if err != nil {
		return nil, nil
	}

	buf, err := img.PngsaveBuffer(pngOptions)
	if err != nil {
		return nil, err
	}

	// A palette holds at most 256 colours; fall back to a full-colour PNG when quantization loses pixels.
	if !samePixels(original, buf) {
		options := *pngOptions
		options.Palette = false
		if buf, err = img.PngsaveBuffer(&options); err != nil {
			return nil, err
		}
		if !samePixels(original, buf) {
			return nil, nil
		}
	}

	if (len(buf) >= len(*data)) {
		return nil, nil
	}

	return buf, nil
}

// samePixels reports whether encoded shows the same RGBA pixels as original.
// The colour of fully transparent pixels is ignored.
func samePixels(original image.Image, encoded []byte) bool {
	decoded, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		return false
	}
	a, b := original.Bounds(), decoded.Bounds()
	if a.Dx() != b.Dx() || a.Dy() != b.Dy() {
		return false
	}
	for y := 0; y < a.Dy(); y++ {
		for x := 0; x < a.Dx(); x++ {
			p := color.NRGBA64Model.Convert(original.At(a.Min.X+x, a.Min.Y+y)).(color.NRGBA64)
			q := color.NRGBA64Model.Convert(decoded.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA64)
			if p.A != q.A || (p.A != 0 && p != q) {
				return false
			}
		}
	}
	return true
}

var CompressJPG types.ProcessorFunc = func(data *[]byte, outFile *string, srcFile *string, conf *types.CompilerConfig) (processedData []byte, err error) {
	sem.Acquire(context.Background(), 1)
	defer sem.Release(1)
	
	img, err := vips.NewJpegloadBuffer(*data, nil)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	buf, err := img.JpegsaveBuffer(jpgOptions)

	if err != nil {
		return nil, err
	}

	if (len(buf) >= len(*data)) {
		return nil, nil
	}

	return buf, nil
}