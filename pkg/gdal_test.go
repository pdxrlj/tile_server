package pkg

import (
	"testing"

	"github.com/lukeroth/gdal"
)

func TestOverView(t *testing.T) {
	memDrv, err := gdal.GetDriverByName("MEM")
	if err != nil {
		t.Errorf(err.Error())
	}
	// 4x4 pixel size
	ds, err := gdal.Open("testdata/smallgeo.tif", gdal.ReadOnly)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	//redband := ds.RasterBand(1)

	// create 1x1 pixelled tiff which would store overview of  redband
	dstile := memDrv.Create("", 256, 256, 3, gdal.Byte, nil)
	//redbandDsTile := dstile.RasterBand(1)
	//greenbandDsTile := dstile.RasterBand(2)
	//blueBandDsTile := dstile.RasterBand(3)
	//bands := []gdal.RasterBand{
	//	redbandDsTile,
	//	greenbandDsTile,
	//	blueBandDsTile,
	//}
	outDrv, err := gdal.GetDriverByName("PNG")
	for i := 0; i < dstile.RasterCount(); i++ {
		band := dstile.RasterBand(i + 1)

		readBand := ds.RasterBand(i + 1)
		readBand.RegenerateOverviews(1, &band, "average", gdal.DummyProgress, nil)
	}
	// generate PNG

	if err != nil {
		panic(err)
	}
	outDrv.CreateCopy("testdata/temp.png", dstile, 0, nil, nil, nil)

}
