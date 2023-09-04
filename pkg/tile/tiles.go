package tile

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

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
	style         string
	bandCount     int
	querySize     int
	ZoomTileIds   [][]*Id
	ZoomMax       int
	ZoomMin       int
	Mercator      *pkgGdal.Mercator
	Gdal          *pkgGdal.Gdal
	Concurrency   int
	wg            *errgroup.Group
	TZMinMax      [][]int
	TzCount       map[int]int
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
	fmt.Printf("输入文件:%s\n", defaultTile.inputFilename)

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
	defaultTile.wg = &errgroup.Group{}
	defaultTile.wg.SetLimit(defaultTile.Concurrency)
	defaultTile.TzCount = make(map[int]int, defaultTile.ZoomMax-defaultTile.ZoomMin+1)
	defaultTile.bandCount = dataset.RasterCount()
	return defaultTile
}

func (tile *Tile) GenerateGdalReadWindows() *Tile {
	minx, miny, maxx, maxy := tile.Gdal.GetBoundsByTransform()
	tile.ZoomTileIds = make([][]*Id, tile.ZoomMax+1)
	tile.TZMinMax = make([][]int, tile.ZoomMax+1)

	for z := tile.ZoomMin; z <= tile.ZoomMax; z++ {
		tminx, tminy := tile.Mercator.MeterToTile(z, minx, miny)
		tmaxx, tmaxy := tile.Mercator.MeterToTile(z, maxx, maxy)
		tminx, tminy = int(math.Max(0, float64(tminx))), int(math.Max(0, float64(tminy)))
		tmaxx, tmaxy = int(math.Min(math.Pow(2, float64(z))-1, float64(tmaxx))), int(math.Min(math.Pow(2, float64(z))-1, float64(tmaxy)))
		//fmt.Printf("当前层级:%d,最小瓦片号:%d,%d,最大瓦片号:%d,%d\n", z, tminx, tminy, tmaxx, tmaxy)
		tile.windows(z, tminx, tminy, tmaxx, tmaxy)
		tile.TZMinMax[z] = []int{tminx, tminy, tmaxx, tmaxy}
	}
	return tile
}

func (tile *Tile) windows(tz, tminx, tminy, tmaxx, tmaxy int) *Tile {
	wg := sync.WaitGroup{}
	tcount := int((1.0 + math.Abs(float64(tmaxx-tminx))) * (1 + math.Abs(float64(tmaxy-tminy))))

	tile.TzCount[tz] = tcount

	tile.ZoomTileIds[tz] = make([]*Id, 0, tcount)
	fmt.Printf("当前层级:%d,最小瓦片号:%d,%d,最大瓦片号:%d,%d,总瓦片数:%d\n", tz, tminx, tminy, tmaxx, tmaxy, tcount)

	for x := tminx; x <= tmaxx; x++ {
		for y := tminy; y <= tmaxy; y++ {
			xCopy := x
			yCopy := y
			wg.Add(1)
			go func() {
				defer wg.Done()
				tmsY := yCopy
				if tile.style == "tms" {
					tmsY = (1 << tz) - yCopy - 1
				}

				filename := fmt.Sprintf("%s/%d/%d/%d.png", tile.outFolder, tz, xCopy, tmsY)
				if tile.outFolder == "" {
					filename = fmt.Sprintf("%d/%d/%d.png", tz, xCopy, tmsY)
				}
				if _, err := os.Stat(filename); err != nil {
					_ = os.MkdirAll(filepath.Dir(filename), os.ModePerm)
				}

				minx, miny, maxx, maxy := tile.Mercator.TileMetersBounds(tz, xCopy, yCopy)
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
				tile.ZoomTileIds[tz] = append(tile.ZoomTileIds[tz], &Id{
					Z:        tz,
					X:        xCopy,
					Y:        yCopy,
					Filename: filename,
					Windows:  windows,
				})

			}()

		}
	}

	wg.Wait()
	return tile
}

func (tile *Tile) Close() error {
	if len(tile.err) > 0 {
		return tile.err[0]
	}
	tile.Gdal.Close()
	return nil
}

// CuttingToImg 裁切影像
func (tile *Tile) CuttingToImg() *Tile {
	if len(tile.err) > 0 {
		return tile
	}
	fmt.Printf("开始裁切影像\n")
	if err := BuildMapTiles(tile); err != nil {
		tile.err = append(tile.err, err)
		return tile
	}

	return tile
}
