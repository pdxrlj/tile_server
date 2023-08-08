package tile_gdal

import (
	"fmt"

	"github.com/lukeroth/gdal"
)

type TileJobFunc func(*GeoQueryGdalJobInfo) error

type NextTileJobFunc func(next TileJobFunc) TileJobFunc

func Execution(info *GeoQueryGdalJobInfo, jobFunc TileJobFunc, jobs ...NextTileJobFunc) error {
	for i := len(jobs) - 1; i >= 0; i-- {
		jobFunc = jobs[i](jobFunc)
	}
	return jobFunc(info)
}

func TileRead() NextTileJobFunc {
	return func(next TileJobFunc) TileJobFunc {
		return func(info *GeoQueryGdalJobInfo) error {
			vrtDs, err := gdal.Open(info.VrtFilename, gdal.ReadOnly)
			if err != nil {
				return err
			}
			bandCount := vrtDs.RasterCount()
			info.BandCount = bandCount
			fmt.Printf("[1/5] TileRead vrtFilename: %v bandCount:%d querySize:%d\n", info.VrtFilename, bandCount, info.QuerySize)

			tmp := make([][]byte, 4)

			for i := 0; i < bandCount; i++ {
				data := make([]byte, info.RxSize*info.RySize)
				for i := range data {
					data[i] = 255
				}
				band := vrtDs.RasterBand(i + 1)
				err := band.IO(gdal.Read, info.Rx, info.Ry, info.RxSize,
					info.RySize, data, info.WxSize, info.WySize, 0, 0)
				if err != nil {
					return err
				}

				tmp[i] = data
			}

			info.ImgData = tmp

			return next(info)
		}
	}
}

func TileToPNG() NextTileJobFunc {
	return func(next TileJobFunc) TileJobFunc {
		return func(info *GeoQueryGdalJobInfo) error {
			fmt.Printf("[2/5] ReadyTileToPng: %v\n", info.TileFilename)
			imgData := info.ImgData
			memDrv, err := gdal.GetDriverByName("MEM")
			if err != nil {
				return err
			}

			dsQuery := memDrv.Create("", info.QuerySize, info.QuerySize, info.BandCount, gdal.Byte, nil)

			for i := 0; i < info.BandCount; i++ {
				err := dsQuery.RasterBand(i+1).IO(gdal.Write, info.Wx, info.Wy, info.WxSize, info.WySize, imgData[i], info.WxSize, info.WySize, 0, 0)
				if err != nil {
					return err
				}

				if err != nil {
					return err
				}
			}

			if err := ScaleQueryToPNG(dsQuery, info.TileFilename); err != nil {
				return err
			}

			return next(info)
		}
	}
}

func ScaleQueryToPNG(query gdal.Dataset, filename string) error {
	fmt.Printf("[3/5] ScaleQueryToPNG\n")
	memDrv, err := gdal.GetDriverByName("MEM")
	if err != nil {
		return err
	}

	bandCount := query.RasterCount()

	dsTile := memDrv.Create("", 256, 256, bandCount, gdal.Byte, nil)
	for i := 0; i < bandCount; i++ {
		dsQueryBand := query.RasterBand(i + 1)
		dstBand := dsTile.RasterBand(i + 1)
		err := dsQueryBand.RegenerateOverviews(1, &dstBand, "average", gdal.DummyProgress, nil)
		if err != nil {
			return err
		}
	}
	outDrv, err := gdal.GetDriverByName("PNG")
	outDrv.CreateCopy(filename, dsTile, 0, nil, nil, nil)
	return nil
}
