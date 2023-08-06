package tile_gdal

import (
	"fmt"
	"os"

	"github.com/lukeroth/gdal"
)

type Gdal struct {
	ds           *gdal.Dataset
	outVrtFolder string
}

func Open(filename string) (*Gdal, error) {
	ds, err := gdal.Open(filename, gdal.ReadOnly)
	if err != nil {
		return nil, err
	}
	gd := &Gdal{ds: &ds}

	return gd, nil
}

// WrapVrt wraps the dataset in a VRT file
func (g *Gdal) WrapVrt() error {
	tempDir, err := os.MkdirTemp("tile_server", "*.vrt")
	if err != nil {
		return err
	}
	fmt.Printf("temp_dir: %s\n", tempDir)

	return nil
}
