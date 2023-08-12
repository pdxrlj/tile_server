package tile

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/lukeroth/gdal"
	"golang.org/x/sync/errgroup"

	pkgGdal "github.com/pdxrlj/tile_server/pkg/gdal"
)

type Tile struct {
	outFolder     string
	tempFileVrt   string
	vrt           gdal.Dataset
	err           []error
	tileSize      int
	inputFilename string
	bandCount     int
	querySize     int
	ZoomTileIds   [][]*TileId
	ZoomMax       int
	ZoomMin       int
	Mercator      *pkgGdal.Mercator
	Gdal          *pkgGdal.Gdal
}

func NewTile(options ...TileOption) *Tile {
	defaultTile := DefaultTile()
	for _, option := range options {
		option(defaultTile)
	}

	if defaultTile.inputFilename == "" {
		defaultTile.err = append(defaultTile.err, pkgGdal.ErrInputFilename)
	}
	if len(defaultTile.err) > 0 {
		return defaultTile
	}
	dataset, err := gdal.Open(defaultTile.inputFilename, gdal.ReadOnly)
	if err != nil {
		defaultTile.err = append(defaultTile.err, err)
		return defaultTile
	}

	vrt, err := pkgGdal.WrapGdalVrt(dataset, 3857)
	if err != nil {
		defaultTile.err = append(defaultTile.err, err)
		return defaultTile
	}

	defaultTile.vrt = vrt.Ds
	defaultTile.tempFileVrt = vrt.Filename
	defaultTile.Mercator = pkgGdal.NewMercator()
	defaultTile.Gdal, err = pkgGdal.NewGdal(defaultTile.tempFileVrt)
	if err != nil {
		defaultTile.err = append(defaultTile.err, err)
		return defaultTile
	}
	defaultTile.Gdal.AdvanceCalculate()

	return defaultTile
}

func (tile *Tile) GenerateGdalReadWindows() *Tile {
	minx, miny, maxx, maxy := tile.Gdal.GetBoundsByTransform()
	tile.ZoomTileIds = make([][]*TileId, tile.ZoomMax+1)

	for z := tile.ZoomMin; z <= tile.ZoomMax; z++ {
		tminx, tminy := tile.Mercator.MeterToTile(z, minx, miny)
		tmaxx, tmaxy := tile.Mercator.MeterToTile(z, maxx, maxy)
		tminx, tminy = int(math.Max(0, float64(tminx))), int(math.Max(0, float64(tminy)))
		tmaxx, tmaxy = int(math.Min(math.Pow(2, float64(z))-1, float64(tmaxx))), int(math.Min(math.Pow(2, float64(z))-1, float64(tmaxy)))
		fmt.Printf("当前层级:%d,最小瓦片号:%d,%d,最大瓦片号:%d,%d\n", z, tminx, tminy, tmaxx, tmaxy)
		tile.windows(z, tminx, tminy, tmaxx, tmaxy)
	}
	return tile
}

func (tile *Tile) windows(tz, tminx, tminy, tmaxx, tmaxy int) *Tile {
	wg := errgroup.Group{}
	tcount := int((1.0 + math.Abs(float64(tmaxx-tminx))) * (1 + math.Abs(float64(tmaxy-tminy))))
	fmt.Printf("当前层级:%d,总共瓦片数:%d\n", tz, tcount)

	wg.SetLimit(int(math.Ceil(float64(tcount / 2))))

	tile.ZoomTileIds[tz] = make([]*TileId, 0, tcount)
	wg.Go(func() error {
		for x := tminx; x <= tmaxx; x++ {
			for y := tminy; y <= tmaxy; y++ {
				filename := fmt.Sprintf("%s/%d/%d/%d.png", tile.outFolder, tz, x, y)
				if tile.outFolder == "" {
					filename = fmt.Sprintf("%d/%d/%d.png", tz, x, y)
				}
				if _, err := os.Stat(filename); err != nil {
					_ = os.MkdirAll(filepath.Dir(filename), os.ModePerm)
				}

				minx, miny, maxx, maxy := tile.Mercator.TileMetersBounds(tz, x, y)
				windows := NewWindows().ReadBox(&WindowsReadBox{
					Minx:         minx,
					Maxy:         maxy,
					Maxx:         maxx,
					Miny:         miny,
					TileSize:     tile.tileSize,
					GeoTransform: tile.Gdal.GetGeoTransform(),
					Height:       tile.Gdal.GetHeight(),
					Width:        tile.Gdal.GetWidth(),
				})
				tile.ZoomTileIds[tz] = append(tile.ZoomTileIds[tz], &TileId{
					Z:        tz,
					X:        x,
					Y:        y,
					Filename: filename,
					Windows:  windows,
				})
			}
		}
		return nil
	})
	err := wg.Wait()
	if err != nil {
		tile.err = append(tile.err, err)
	}
	return tile
}

func (tile *Tile) Close() error {
	if len(tile.err) > 0 {
		return tile.err[0]
	}
	tile.Gdal.Close()
	return nil
}

func (tile *Tile) CuttingToImg() *Tile {
	if len(tile.err) > 0 {
		return tile
	}

	wg := errgroup.Group{}
	wg.SetLimit(1)
	for z := tile.ZoomMin; z <= tile.ZoomMax; z++ {
		for _, tileId := range tile.ZoomTileIds[z] {
			wg.Go(func() error {
				dataset, err := gdal.Open(tile.tempFileVrt, gdal.ReadOnly)
				if err != nil {
					return err
				}

				err = tileId.ReadTile(dataset)
				if err != nil {
					return err
				}
				return nil
			})
		}
	}

	err := wg.Wait()
	if err != nil {
		tile.err = append(tile.err, err)
		return tile
	}

	return tile
}
