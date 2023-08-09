package tile

import (
	"math"
)

type Window struct {
	// 瓦片在源图上的 x/y 像素坐标
	Rx, Ry int
	// 瓦片在源图读取瓦片的宽高
	RxSize, RySize int
	// 写入文件的位置
	Wx, Wy int
	// 写入文件宽高
	WxSize, WySize int
}

// WindowsReadBox 计算Dataset要读取瓦片的像素位置，根据给定的瓦片的地理范围(单位米)
type WindowsReadBox struct {
	Minx, Maxy, Maxx, Miny float64
	TileSize               int
	Height, Width          int
	GeoTransform           [6]float64
}

func NewWindows() *Window {
	return &Window{}
}

func (t *Window) ReadBox(box *WindowsReadBox) *Window {
	// 计算该瓦片的左上角在源图上的 x/y 像素坐标
	geoTransform := box.GeoTransform
	RasterXSize := box.Width
	RasterYSize := box.Height
	rx := int((box.Minx-geoTransform[0])/geoTransform[1] + 0.001)
	ry := int((box.Maxy-geoTransform[3])/geoTransform[5] + 0.001)
	// 计算该瓦片在源图读取瓦片的宽高
	rxSize := int((box.Maxx-box.Minx)/geoTransform[1] + 0.5)
	rySize := int((box.Miny-box.Maxy)/geoTransform[5] + 0.5)

	// 写入文件宽高
	wxSize, wySize := 4*box.TileSize, 4*box.TileSize
	// 写入文件的位置
	wx := 0

	if rx < 0 {
		rxShift := math.Abs(float64(rx))
		wx = int(float64(wxSize) * (rxShift / float64(rxSize)))
		wxSize = wxSize - wx
		rxSize = rxSize - int(float64(rxSize)*(rxShift/float64(rxSize)))
		rx = 0
	}

	if rx+rxSize > RasterXSize {
		wxSize = int(float64(wxSize) * (float64(RasterXSize-rx) / float64(rxSize)))
		rxSize = RasterXSize - rx
	}

	wy := 0

	if ry < 0 {
		ryShift := math.Abs(float64(ry))
		wy = int(float64(wySize) * (ryShift / float64(rySize)))
		wySize = wySize - wy
		rySize = rySize - int(float64(rySize)*(ryShift/float64(rySize)))
		ry = 0
	}

	if ry+rySize > RasterYSize {
		wySize = int(float64(wySize) * (float64(RasterYSize-ry) / float64(rySize)))
		rySize = RasterYSize - ry
	}

	return &Window{
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
