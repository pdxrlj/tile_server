package tile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/lukeroth/gdal"
	"github.com/pkg/errors"
)

type TileOverviewFn func(*Tile) error

type NextTileOverviewFn func(fn TileOverviewFn) TileOverviewFn

func Interceptor(data *Tile, fn TileOverviewFn, fns ...NextTileOverviewFn) error {
	for i := len(fns) - 1; i >= 0; i-- {
		fn = fns[i](fn)
	}
	return fn(data)
}

func BuildMapTiles(data *Tile) error {
	return Interceptor(data, func(tile *Tile) error {
		fmt.Printf("瓦片切片完成")
		return nil
	}, OverviewTile())
}

// BaseTile 生成基础瓦片
func BaseTile() NextTileOverviewFn {
	return func(next TileOverviewFn) TileOverviewFn {
		return func(data *Tile) error {
			sg := sync.WaitGroup{}
			baseZoom := data.ZoomMax

			fmt.Printf("[1/2] 开始生成底图瓦片数据 zoom=%d count=%d\n", baseZoom, data.TzCount[baseZoom])
			for _, tileId := range data.ZoomTileIds[baseZoom] {
				tileIdCopy := tileId
				sg.Add(1)
				go func() {
					defer sg.Done()
					dataset, err := gdal.Open(data.tempFileVrt, gdal.ReadOnly)
					if err != nil {
						data.err = append(data.err, err)
					}
					err = tileIdCopy.ReadTile(dataset)
					if err != nil {
						data.err = append(data.err, err)
					}
				}()
			}

			sg.Wait()
			if len(data.err) > 0 {
				return errors.WithStack(data.err[0])
			}

			return next(data)
		}
	}
}

// OverviewTile 生成缩略图瓦片数据
func OverviewTile() NextTileOverviewFn {
	return func(next TileOverviewFn) TileOverviewFn {
		return func(tile *Tile) error {
			overviewMaxZoom := tile.ZoomMax - 1
			memDriver, err := gdal.GetDriverByName("MEM")
			if err != nil {
				return err
			}

			for overview := overviewMaxZoom; overview >= tile.ZoomMin; overview-- {
				fmt.Printf("[2/2] 开始生成缩略图瓦片数据 zoom=%d count=%d\n", overview, tile.TzCount[overview])
				zoomTileIds := tile.ZoomTileIds[overview]
				for _, tileId := range zoomTileIds {

					tileIdCopy := tileId
					x := tileIdCopy.X
					y := tileIdCopy.Y
					z := tileIdCopy.Z

					dsQuery := memDriver.Create("", 2*256, 2*256, tile.bandCount, gdal.Byte, nil)
				SUB:
					for tx := 2 * x; tx < 2*x+2; tx++ {
						for ty := y * 2; ty < y*2+2; ty++ {
							tMinMax := tile.TZMinMax[z+1]
							minx, miny, maxx, maxy := tMinMax[0], tMinMax[1], tMinMax[2], tMinMax[3]
							if minx <= tx && tx <= maxx && miny <= ty && ty <= maxy {
								tyCopy := ty
								if tile.style == "tms" {
									tyCopy = (1 << (z + 1)) - ty - 1
								}

								baseTile := filepath.Join(tile.outFolder, fmt.Sprintf("%d/%d/%d.png", z+1, tx, tyCopy))
								if _, err := os.Stat(filepath.Dir(baseTile)); err != nil {
									tile.err = append(tile.err, err)
									break SUB
								}

								dataset, err := gdal.Open(baseTile, gdal.ReadOnly)
								if err != nil {
									tile.err = append(tile.err, err)
									break SUB
								}
								tilePoxY := 256
								if (y == 0 && ty == 1) || (y != 0 && (ty%(2*y) != 0)) {
									tilePoxY = 0
								}

								tilePoxX := 0
								if x != 0 {
									tilePoxX = tx % (2 * x) * 256
								} else if x == 0 && tx == 1 {
									tilePoxX = 256
								}

								for i := 0; i < dataset.RasterCount(); i++ {
									readTmp := make([]byte, 256*256)
									err = dataset.RasterBand(i+1).IO(gdal.Read, 0, 0, 256, 256, readTmp, 256, 256, 0, 0)
									if err != nil {
										tile.err = append(tile.err, errors.WithStack(err))
										break SUB
									}

									err = dsQuery.RasterBand(i+1).IO(gdal.Write, tilePoxX, tilePoxY, 256, 256, readTmp, 256, 256, 0, 0)
									if err != nil {
										tile.err = append(tile.err, errors.WithStack(err))
										break SUB
									}
								}
							}
						}
					}
					if err := RegenerateOverviews(tileIdCopy.Filename, &dsQuery); err != nil {
						tile.err = append(tile.err, errors.WithStack(err))
						continue
					}
				}
			}

			return next(tile)
		}
	}
}

func RegenerateOverviews(outFilename string, dst *gdal.Dataset) error {
	memDrv, err := gdal.GetDriverByName("MEM")
	if err != nil {
		return err
	}
	bands := dst.RasterCount()
	dsTile := memDrv.Create("", 256, 256, bands, gdal.Byte, nil)
	for i := 0; i < bands; i++ {
		dstBand := dsTile.RasterBand(i + 1)
		err := dst.RasterBand(i+1).RegenerateOverviews(1, &dstBand, "average", gdal.DummyProgress, nil)
		if err != nil {
			fmt.Printf("RegenerateOverviews err:%s\n", err)
			return err
		}
	}
	outDrv, err := gdal.GetDriverByName("PNG")
	if err != nil {
		return err
	}
	outDrv.CreateCopy(outFilename, dsTile, 0, nil, nil, nil)
	return nil
}
