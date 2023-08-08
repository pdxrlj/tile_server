package tile_gdal

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/lukeroth/gdal"
	"golang.org/x/sync/errgroup"
)

type Tile struct {
	read          *Read
	inputFilename string
	outFolder     string
	Mercator      *Mercator
	TMinMax       map[int]TileMinMax
	ZoomMax       int
	ZoomMin       int
	TileJobInfo   chan *GeoQueryGdalJobInfo
	EG            *errgroup.Group
}

type TileOption func(*Tile)

func SetTileRead(r *Read) TileOption {
	return func(t *Tile) {
		t.read = r
	}
}

func SetTileInputFilename(inputFilename string) TileOption {
	return func(t *Tile) {
		t.inputFilename = inputFilename
	}
}

func SetTileOutFolder(outFolder string) TileOption {
	return func(t *Tile) {
		t.outFolder = outFolder
	}
}

func SetTileZoomMax(zoomMax int) TileOption {
	return func(t *Tile) {
		t.ZoomMax = zoomMax
	}
}

func SetTileZoomMin(zoomMin int) TileOption {
	return func(t *Tile) {
		t.ZoomMin = zoomMin
	}
}

func DefaultOpenTile() *Tile {
	return &Tile{
		Mercator: NewMercator(),
		ZoomMax:  16,
		ZoomMin:  16,
		EG:       &errgroup.Group{},
	}
}

func OpenTile(options ...TileOption) (*Tile, error) {
	defaultTile := DefaultOpenTile()
	for _, option := range options {
		option(defaultTile)
	}

	if defaultTile.inputFilename == "" {
		return nil, ErrInputFilename
	}

	if defaultTile.read == nil {
		read, err := Open(
			SetInputFilename(defaultTile.inputFilename),
			SetOutFolder(defaultTile.outFolder),
		)
		if err != nil {
			return nil, err
		}
		if err := read.WrapVrt().Execute(); err != nil {
			return nil, err
		}
		defaultTile.read = read

		defaultTile.TileJobInfo = make(chan *GeoQueryGdalJobInfo, 100)
	}

	return defaultTile, nil
}

func (t *Tile) Close() {
	t.read.Close()
}

type GeoQueryGdalJobInfo struct {
	// 瓦片在源图上的 x/y 像素坐标
	Rx, Ry int
	// 瓦片在源图读取瓦片的宽高
	RxSize, RySize int
	// 写入文件的位置
	Wx, Wy int
	// 写入文件宽高
	WxSize, WySize int

	TileFilename string
	VrtFilename  string

	ImgData   [][]byte
	BandCount int

	QuerySize int
	dsQuery   gdal.Dataset
	dsTile    gdal.Dataset
}

// GeoQuery
// minx, maxy, maxx, miny uint meters
func (t *Tile) GeoQuery(minx, maxy, maxx, miny float64) *GeoQueryGdalJobInfo {
	geoTransform := t.read.vrt.GeoTransform()
	// 计算该瓦片的左上角在源图上的 x/y 像素坐标
	rx := int((minx-geoTransform[0])/geoTransform[1] + 0.001)
	ry := int((maxy-geoTransform[3])/geoTransform[5] + 0.001)
	// 计算该瓦片在源图读取瓦片的宽高
	rxSize := int((maxx-minx)/geoTransform[1] + 0.5)
	rySize := int((miny-maxy)/geoTransform[5] + 0.5)

	// 写入文件宽高
	wxSize, wySize := 4*t.read.tileSize, 4*t.read.tileSize
	// 写入文件的位置
	wx := 0

	if rx < 0 {
		rxShift := math.Abs(float64(rx))
		wx = int(float64(wxSize) * (rxShift / float64(rxSize)))
		wxSize = wxSize - wx
		rxSize = rxSize - int(float64(rxSize)*(rxShift/float64(rxSize)))
		rx = 0
	}

	if rx+rxSize > t.read.vrt.RasterXSize() {
		wxSize = int(float64(wxSize) * (float64(t.read.vrt.RasterXSize()-rx) / float64(rxSize)))
		rxSize = t.read.vrt.RasterXSize() - rx
	}

	wy := 0

	if ry < 0 {
		ryShift := math.Abs(float64(ry))
		wy = int(float64(wySize) * (ryShift / float64(rySize)))
		wySize = wySize - wy
		rySize = rySize - int(float64(rySize)*(ryShift/float64(rySize)))
		ry = 0
	}

	if ry+rySize > t.read.vrt.RasterYSize() {
		wySize = int(float64(wySize) * (float64(t.read.vrt.RasterYSize()-ry) / float64(rySize)))
		rySize = t.read.vrt.RasterYSize() - ry
	}

	return &GeoQueryGdalJobInfo{
		Rx:     rx,
		Ry:     ry,
		RxSize: rxSize,
		RySize: rySize,
		Wx:     wx,
		Wy:     wy,
		WxSize: wxSize,
		WySize: wySize,
	}
}

type TileMinMax struct {
	tminx, tminy, tmaxx, tmaxy int
}

func (t *Tile) TileRange() *Tile {
	minx, miny, maxx, maxy := t.read.TileBoundsByTransform()
	t.TMinMax = make(map[int]TileMinMax)
	for z := t.ZoomMin; z <= t.ZoomMax; z++ {
		tminx, tminy := t.Mercator.MeterToTile(z, minx, miny)
		tmaxx, tmaxy := t.Mercator.MeterToTile(z, maxx, maxy)
		tminx, tminy = int(math.Max(0, float64(tminx))), int(math.Max(0, float64(tminy)))
		tmaxx, tmaxy = int(math.Min(math.Pow(2, float64(z))-1, float64(tmaxx))), int(math.Min(math.Pow(2, float64(z))-1, float64(tmaxy)))
		t.TMinMax[z] = TileMinMax{
			tminx: tminx,
			tminy: tminy,
			tmaxx: tmaxx,
			tmaxy: tmaxy,
		}
	}
	fmt.Printf("[1/3]TileRange: %v\n", t.TMinMax)
	return t
}

func (t *Tile) MakeTileJobInfo() *Tile {
	tileMinMax := t.TMinMax[t.ZoomMax]
	tcount := int((1.0 + math.Abs(float64(tileMinMax.tmaxx-tileMinMax.tminx))) * (1 + math.Abs(float64(tileMinMax.tmaxy-tileMinMax.tminy))))
	fmt.Printf("[2/3]MakeTileJobInfo tcount: %d\n", tcount)
	t.EG.SetLimit(2)

	go func() {
		for x := tileMinMax.tminx; x <= tileMinMax.tmaxx; x++ {
			for y := tileMinMax.tminy; y <= tileMinMax.tmaxy; y++ {
				tileFilename := fmt.Sprintf("%s/%d/%d/%d.png", t.outFolder, t.ZoomMax, x, y)
				if t.outFolder == "" {
					tileFilename = fmt.Sprintf("%d/%d/%d.png", t.ZoomMax, x, y)
				}

				if _, err := os.Stat(tileFilename); err != nil {
					_ = os.MkdirAll(filepath.Dir(tileFilename), 0755)
				}
				minx, miny, maxx, maxy := t.Mercator.TileBounds(t.ZoomMax, x, y)

				geoQuery := t.GeoQuery(minx, maxy, maxx, miny)
				geoQuery.TileFilename = tileFilename
				geoQuery.VrtFilename = t.read.tempFileVrt.Name()
				geoQuery.QuerySize = t.read.querySize
				//fmt.Printf("[4/5]GeoQuery: %+v\n", geoQuery)
				t.TileJobInfo <- geoQuery
			}
		}
	}()

	return t
}

func (t *Tile) GenerateBaseTile() error {
	for info := range t.TileJobInfo {
		jobInfoCopy := info
		fmt.Printf("[3/3]GenerateBaseTile: %+v\n", jobInfoCopy)
		t.EG.Go(func() error {
			if jobInfoCopy.RxSize == 0 || jobInfoCopy.RySize == 0 || jobInfoCopy.WxSize == 0 || jobInfoCopy.WySize == 0 {
				return nil
			}
			err := Execution(jobInfoCopy, func(info *GeoQueryGdalJobInfo) error {
				fmt.Printf("[5/5] Execution: %+v\n", info.TileFilename)
				return nil
			}, TileRead(), TileToPNG())
			if err != nil {
				return err
			}
			return nil
		})
	}

	if err := t.EG.Wait(); err != nil {
		return err
	}
	close(t.TileJobInfo)
	return nil
}
