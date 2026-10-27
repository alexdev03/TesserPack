package compiler

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"tesserpack/internal/helpers"
	"tesserpack/internal/helpers/config"
	"testing"
)

func TestCompilePreservesPNGAndLang(t *testing.T) {
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/a2kAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	lang := []byte("hello=Hello\n# Preserve comment\n")
	for _, cached := range []bool{false, true} {
		name := "uncached"
		if cached {
			name = "cached"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			previous := helpers.TempDir
			helpers.TempDir = root
			t.Cleanup(func() { helpers.TempDir = previous })
			input := filepath.Join(root, "input")
			for file, data := range map[string][]byte{"textures/tiny.png": png, "texts/en_US.lang": lang, "manifest.json": []byte(`{"format_version":2}`)} {
				file = filepath.Join(input, filepath.FromSlash(file))
				if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			conf := config.NewDefault()
			conf.Compiler.Cache = cached
			for run := 0; run < 2; run++ {
				stage := filepath.Join(root, name+string(rune('0'+run)))
				if err := os.Mkdir(stage, 0755); err != nil {
					t.Fatal(err)
				}
				out := stage + ".mcpack"
				if err := Compile(input, input, out, stage, &conf); err != nil {
					t.Fatal(err)
				}
				z, err := zip.OpenReader(out)
				if err != nil {
					t.Fatal(err)
				}
				found := map[string][]byte{}
				for _, f := range z.File {
					r, err := f.Open()
					if err != nil {
						t.Fatal(err)
					}
					b, err := io.ReadAll(r)
					r.Close()
					if err != nil {
						t.Fatal(err)
					}
					found[f.Name] = b
				}
				z.Close()
				if !bytes.Equal(found["textures/tiny.png"], png) {
					t.Errorf("run %d: missing or changed tiny PNG", run)
				}
				if !bytes.Equal(found["texts/en_US.lang"], lang) {
					t.Errorf("run %d: missing or changed language file", run)
				}
				if _, ok := found["texts/en_US.json"]; ok {
					t.Errorf("run %d: language file renamed to JSON", run)
				}
			}
		})
	}
}
