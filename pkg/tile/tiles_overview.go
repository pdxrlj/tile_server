package tile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lukeroth/gdal"
)

type TileOverviewFn func(*Tile) error

type NextTileOverviewFn func(fn TileOverviewFn) TileOverviewFn

func Interceptor(data *Tile, fn TileOverviewFn, fns ...NextTileOverviewFn) error {
	for i := len(fns); i > 0; i-- {
		fn = fns[i](fn)
	}
	return fn(data)
}

// BaseTile 生成基础瓦片
func BaseTile() NextTileOverviewFn {
	return func(fn TileOverviewFn) TileOverviewFn {
		return func(data *Tile) error {
			baseZoom := data.ZoomMax
			for _, tileId := range data.ZoomTileIds[baseZoom] {
				tileIdCopy := tileId
				data.wg.Go(func() error {
					dataset, err := gdal.Open(data.tempFileVrt, gdal.ReadOnly)
					if err != nil {
						return err
					}

					err = tileIdCopy.ReadTile(dataset)
					if err != nil {
						return err
					}
					return nil
				})
			}

			if err := data.wg.Wait(); err != nil {
				return err
			}

			return fn(data)
		}
	}
}

// OverviewTile 生成缩略图瓦片数据
func OverviewTile() NextTileOverviewFn {
	return func(fn TileOverviewFn) TileOverviewFn {
		return func(tile *Tile) error {
			overviewMaxZoom := tile.ZoomMax - 1
			memDriver, err := gdal.GetDriverByName("MEM")
			if err != nil {
				return err
			}
			for overview := overviewMaxZoom; overview > tile.ZoomMin; overview++ {
				for _, tileId := range tile.ZoomTileIds[overview] {
					tileIdCopy := tileId

					tile.wg.Go(func() error {
						x := tileIdCopy.X
						y := tileIdCopy.Y
						z := tileIdCopy.Z
						dsquery := memDriver.Create("", 2*256, 2*256, tile.bandCount, gdal.Byte, nil)
						dsTile := memDriver.Create("", 256, 256, tile.bandCount, gdal.Byte, nil)
						for tx := 2 * x; tx < 2*x+2; tx++ {
							for ty := y * 2; ty < y*2+2; ty++ {
								tMinMax := tile.TZMinMax[z+1]
								minx, miny, maxx, maxy := tMinMax[0], tMinMax[1], tMinMax[2], tMinMax[3]
								if minx <= tx && tx <= maxx && miny <= ty && ty <= maxy {
									baseTile := filepath.Join(tile.outFolder, fmt.Sprintf("%d/%d/%d.png", z+1, i, j))
									if _, err := os.Stat(baseTile); err != nil {
										fmt.Printf("baseTile %s not exists err:%s\n", baseTile, err)
										return err
									}

									dataset, err := gdal.Open(baseTile, gdal.ReadOnly)
									if err != nil {
										return err
									}
									tilePoxY := 256
									if (y == 0 && ty == 1) || (y != 0 && (ty%(2*y) != 0)) {
										tilePoxY = 0
									}

									tilePoxX := 0
									if x != 0 {
										tilePoxX = x % (2 * tx) * 256
									} else if x == 0 && tx == 1 {
										tilePoxX = 256
									}

									readTmp := make([]byte, 256*256)
									err = dataset.IO(gdal.Read, 0, 0, 256, 256, &readTmp, 256, 256, dataset.RasterCount(), nil, 0, 0, 0)
									if err != nil {
										return err
									}

									err = dsquery.IO(gdal.Write, tilePoxX, tilePoxY, 256, 256, readTmp, 256, 256, dataset.RasterCount(), nil, 0, 0, 0)
									if err != nil {
										return err
									}
								}
							}
						}

						return nil
					})
				}
			}
			if err := tile.wg.Wait(); err != nil {
				return err
			}

			return fn(tile)
		}
	}
}

func RegenerateOverviews(outFilename string, dst gdal.Dataset) error {
	memDrv, err := gdal.GetDriverByName("MEM")
	if err != nil {
		return err
	}
	bands := dst.RasterCount()
	dsTile := memDrv.Create("", 256, 256, bands, gdal.Byte, nil)
	for i := 1; i <= bands; i++ {
		dstBand := dsTile.RasterBand(i)
		err := dst.RasterBand(i).RegenerateOverviews(1, &dstBand, "average", gdal.DummyProgress, nil)
		if err != nil {
			return err
		}
	}
	outDrv, err := gdal.GetDriverByName("PNG")
	outDrv.CreateCopy(outFilename, dsTile, 0, nil, nil, nil)
	return nil
}
