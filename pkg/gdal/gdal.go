package gdal

import (
	"os"

	"github.com/lukeroth/gdal"
)

func CreateSpatialReference(epsg int) (string, error) {
	sp := gdal.CreateSpatialReference("")
	if epsg != 0 {
		epsg = 3857
	}
	err := sp.FromEPSG(epsg)
	if err != nil {
		return "", nil
	}

	return sp.ToWKT()
}

func SpatialReference(ds gdal.Dataset) (string, error) {
	pro := ds.Projection()
	sp := gdal.CreateSpatialReference("")
	err := sp.FromProj4(pro)
	if err != nil {
		return "", err
	}
	return sp.ToWKT()
}

func CheckSpatialReferenceIsMercator(ds gdal.Dataset) bool {
	spatialRef := gdal.CreateSpatialReference(ds.Projection())
	if spatialRef.IsProjected() {
		return true
	}
	if spatialRef.IsGeocentric() {
		return false
	}
	return false
}

type VrtInfo struct {
	Filename string
	Ds       gdal.Dataset
}

func WrapGdalVrt(src gdal.Dataset, epsgCode int) (*VrtInfo, error) {
	tempFile, err := os.CreateTemp("", "*.vrt")
	if err != nil {
		return nil, err
	}

	ds := src

	srcWkt, err := SpatialReference(ds)
	if err != nil {
		return nil, err
	}

	dstWkt, err := CreateSpatialReference(epsgCode)
	if err != nil {
		return nil, err
	}

	warpedVRT, err := ds.AutoCreateWarpedVRT(srcWkt, dstWkt, gdal.GRA_NearestNeighbour)
	if err != nil {
		return nil, err
	}
	vrt, err := gdal.GetDriverByName("VRT")

	options := []string{
		"INIT_DEST=INIT_DEST",
		"UNIFIED_SRC_NODATA=YES",
	}
	err = warpedVRT.SetMetadataItem("NODATA_VALUES", "0 0 0 0", "IMAGE_STRUCTURE")
	if err != nil {
		return nil, err
	}
	warpedVRT = vrt.CreateCopy(tempFile.Name(), warpedVRT, 0, options, nil, nil)

	vrtInfo := &VrtInfo{
		Filename: tempFile.Name(),
		Ds:       warpedVRT,
	}
	_ = tempFile.Close()
	return vrtInfo, nil
}

func TileBoundsByTransform(dataset gdal.Dataset) (float64, float64, float64, float64) {
	transform := dataset.GeoTransform()
	minx := transform[0]
	maxy := transform[3]
	maxx := minx + transform[1]*float64(dataset.RasterXSize())
	miny := maxy + transform[5]*float64(dataset.RasterYSize())
	return minx, miny, maxx, maxy
}

type Gdal struct {
	Height       int
	Width        int
	GeoTransform [6]float64
	Dataset      gdal.Dataset
	BandCount    int
}

func NewGdal(filename string) (*Gdal, error) {
	dataset, err := gdal.Open(filename, gdal.ReadOnly)
	if err != nil {
		return nil, err
	}

	return &Gdal{
		Dataset: dataset,
	}, nil
}

func (g *Gdal) GetHeight() int {
	return g.Height
}

func (g *Gdal) GetWidth() int {
	return g.Width
}

func (g *Gdal) GetGeoTransform() [6]float64 {
	return g.GeoTransform
}

func (g *Gdal) GetBoundsByTransform() (float64, float64, float64, float64) {
	transform := g.GetGeoTransform()
	height := g.GetHeight()
	width := g.GetWidth()

	minx := transform[0]
	maxy := transform[3]
	maxx := minx + transform[1]*float64(width)
	miny := maxy + transform[5]*float64(height)
	return minx, miny, maxx, maxy
}

func (g *Gdal) RasterCount() int {
	return g.BandCount
}

func (g *Gdal) AdvanceCalculate() *Gdal {
	g.Height = g.Dataset.RasterYSize()
	g.Width = g.Dataset.RasterXSize()
	g.GeoTransform = g.Dataset.GeoTransform()
	g.BandCount = g.Dataset.RasterCount()
	return g
}

func (g *Gdal) Close() {
	g.Dataset.Close()
}
