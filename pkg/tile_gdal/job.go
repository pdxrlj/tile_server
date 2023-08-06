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
			tmp := make([][]byte, info.QuerySize*info.QuerySize)
			info.BandCount = bandCount
			fmt.Printf("[1/5] TileRead vrtFilename: %v bandCount:%d querySize:%d\n", info.VrtFilename, bandCount, info.QuerySize)

			err = vrtDs.IO(gdal.Read, info.Rx, info.Ry, info.RxSize, info.RySize, tmp, info.WxSize, info.WySize, bandCount, nil, 0, 0, 0)
			if err != nil {
				return err
			}

			//for i := 0; i < bandCount; i++ {
			//	data := make([]byte, info.QuerySize*info.QuerySize)
			//	for i := range data {
			//		data[i] = 255
			//	}
			//	band := vrtDs.RasterBand(i + 1)
			//	err := band.IO(gdal.Read, info.Rx, info.Ry, info.RxSize,
			//		info.RySize, data, info.WxSize, info.WySize, 0, 0)
			//	if err != nil {
			//		return err
			//	}
			//
			//	tmp[i] = data
			//}

			info.ImgData = tmp

			return next(info)
		}
	}
}

func ReadyTile() NextTileJobFunc {
	return func(next TileJobFunc) TileJobFunc {
		return func(info *GeoQueryGdalJobInfo) error {
			fmt.Printf("[2/5] ReadyTile: %v\n", info.TileFilename)
			imgData := info.ImgData
			memDrv, err := gdal.GetDriverByName("MEM")
			if err != nil {
				return err
			}
			ds, err := gdal.Open(info.VrtFilename, gdal.ReadOnly)
			if err != nil {
				return err
			}
			alphaBand := ds.RasterBand(1).GetMaskBand()
			alphaData := make([][]byte, info.QuerySize*info.QuerySize)
			err = alphaBand.IO(gdal.Read, info.Rx, info.Ry, info.RxSize, info.RySize, alphaData, info.WxSize, info.WySize, 0, 0)
			if err != nil {
				return err
			}
			dsQuery := memDrv.Create("", info.WxSize, info.WySize, info.BandCount, gdal.Byte, nil)
			err = dsQuery.IO(gdal.Write, info.Wx, info.Wy, info.WxSize, info.WySize, imgData, info.WxSize, info.WySize, info.BandCount, nil, 0, 0, 0)
			if err != nil {
				return err
			}
			err = dsQuery.IO(gdal.Write, info.Wx, info.Wy, info.WxSize, info.WySize, alphaData, info.WxSize, info.WySize, 1, nil, 0, 0, 0)
			if err != nil {
				return err
			}
			info.dsQuery = dsQuery
			info.dsTile = memDrv.Create("", 256, 256, info.BandCount, gdal.Byte, nil)
			return next(info)
		}
	}
}

func ScaleQueryToTile() NextTileJobFunc {
	return func(next TileJobFunc) TileJobFunc {
		return func(info *GeoQueryGdalJobInfo) error {
			fmt.Printf("[3/5] ScaleQueryToTile\n")
			for i := 0; i < info.BandCount; i++ {
				dstBand := info.dsTile.RasterBand(i + 1)
				err := info.dsQuery.RasterBand(i+1).RegenerateOverviews(3, &dstBand, "average", nil, nil)
				if err != nil {
					return err
				}
			}
			return next(info)
		}
	}
}

func ToPNG() NextTileJobFunc {
	return func(next TileJobFunc) TileJobFunc {
		return func(info *GeoQueryGdalJobInfo) error {
			fmt.Printf("[4/5] ToPNG: %v\n", info.TileFilename)
			outDrv, err := gdal.GetDriverByName("PNG")
			if err != nil {
				return err
			}

			outDs := outDrv.CreateCopy(info.TileFilename, info.dsTile, 0, nil, nil, nil)
			outDs.FlushCache()
			outDs.Close()
			return next(info)
		}
	}
}
