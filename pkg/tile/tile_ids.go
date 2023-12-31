package tile

import (
	"fmt"

	"github.com/lukeroth/gdal"
)

type Id struct {
	Z, X, Y   int
	Windows   *Window
	Filename  string
	querySize int
	dataset   gdal.Dataset
	imgBuf    [][]byte
	dsQuery   gdal.Dataset
}

func (t *Id) String() string {
	return fmt.Sprintf("Z:%d X:%d Y:%d", t.Z, t.X, t.Y)
}

func (t *Id) ReadTile(dataset gdal.Dataset) error {
	return ReadExec(t, func(info *Id) error {
		memDrv, err := gdal.GetDriverByName("MEM")
		if err != nil {
			return err
		}
		bandCount := info.dsQuery.RasterCount()
		dsTile := memDrv.Create("", 256, 256, bandCount, gdal.Byte, nil)
		for i := 0; i < bandCount; i++ {
			dsQueryBand := info.dsQuery.RasterBand(i + 1)
			dstBand := dsTile.RasterBand(i + 1)
			err := dsQueryBand.RegenerateOverviews(1, &dstBand, "average", gdal.DummyProgress, nil)
			if err != nil {
				return err
			}
		}
		outDrv, err := gdal.GetDriverByName("PNG")
		outDrv.CreateCopy(info.Filename, dsTile, 0, nil, nil, nil)

		return nil

	}, initTileRead(dataset), Read(), TileToPNG())
}

type ReadFunc func(*Id) error

type NextTileReadFunc func(next ReadFunc) ReadFunc

func ReadExec(info *Id, readFunc ReadFunc, next ...NextTileReadFunc) error {
	for i := len(next) - 1; i >= 0; i-- {
		readFunc = next[i](readFunc)
	}
	return readFunc(info)
}

func initTileRead(dataset gdal.Dataset) NextTileReadFunc {
	return func(next ReadFunc) ReadFunc {
		return func(info *Id) error {
			info.querySize = 256 * 4
			info.dataset = dataset
			info.imgBuf = make([][]byte, info.dataset.RasterCount())
			return next(info)
		}
	}
}

func Read() NextTileReadFunc {
	return func(next ReadFunc) ReadFunc {
		return func(info *Id) error {
			bandCount := info.dataset.RasterCount()

			for i := 0; i < bandCount; i++ {
				data := make([]byte, info.Windows.RxSize*info.Windows.RySize)
				for d := range data {
					data[d] = 255
				}
				band := info.dataset.RasterBand(i + 1)
				err := band.IO(gdal.Read, info.Windows.Rx, info.Windows.Ry, info.Windows.RxSize,
					info.Windows.RySize, data, info.Windows.WxSize, info.Windows.WySize, 0, 0)
				if err != nil {
					return err
				}

				info.imgBuf[i] = data
			}

			return next(info)
		}
	}
}

func TileToPNG() NextTileReadFunc {
	return func(next ReadFunc) ReadFunc {
		return func(info *Id) error {
			imgData := info.imgBuf
			memDrv, err := gdal.GetDriverByName("MEM")
			if err != nil {
				return err
			}
			bandCount := info.dataset.RasterCount()
			dsQuery := memDrv.Create("", info.querySize, info.querySize, bandCount, gdal.Byte, nil)

			for i := 0; i < bandCount; i++ {
				err := dsQuery.RasterBand(i+1).IO(gdal.Write, info.Windows.Wx, info.Windows.Wy, info.Windows.WxSize,
					info.Windows.WySize, imgData[i], info.Windows.WxSize, info.Windows.WySize, 0, 0)
				if err != nil {
					return err
				}
			}

			info.dsQuery = dsQuery

			return next(info)
		}
	}
}
