package gdal

import (
	"os"

	"github.com/lukeroth/gdal"
)

type Read struct {
	ds            gdal.Dataset
	outFolder     string
	tempFileVrt   *os.File
	vrt           gdal.Dataset
	err           []error
	tileSize      int
	inputFilename string
	bandCount     int
	querySize     int
}

type ReadOption func(*Read)

func SetTileSize(tileSize int) ReadOption {
	return func(r *Read) {
		r.tileSize = tileSize
	}
}

func SetOutFolder(outFolder string) ReadOption {
	return func(r *Read) {
		r.outFolder = outFolder
	}
}

func SetInputFilename(inputFilename string) ReadOption {
	return func(r *Read) {
		r.inputFilename = inputFilename
	}
}

func DefaultRead() *Read {
	return &Read{
		tileSize:  256,
		outFolder: "",
		querySize: 256 * 4,
	}
}

func Open(options ...ReadOption) (*Read, error) {
	defaultRead := DefaultRead()
	for _, option := range options {
		option(defaultRead)
	}

	if defaultRead.inputFilename == "" {
		return nil, ErrInputFilename
	}

	ds, err := gdal.Open(defaultRead.inputFilename, gdal.ReadOnly)
	if err != nil {
		return nil, err
	}
	defaultRead.ds = ds

	return defaultRead, nil
}

// WrapVrt wraps the dataset in a VRT file
func (r *Read) WrapVrt() *Read {
	tempFile, err := os.CreateTemp("", "*.vrt")
	if err != nil {
		r.err = append(r.err, err)
		return r
	}
	r.tempFileVrt = tempFile

	ds := r.ds

	srcWkt, err := SpatialReference(ds)
	if err != nil {
		r.err = append(r.err, err)
		return r
	}

	dstWkt, err := CreateSpatialReference(3857)
	if err != nil {
		r.err = append(r.err, err)
		return r
	}

	warpedVRT, err := ds.AutoCreateWarpedVRT(srcWkt, dstWkt, gdal.GRA_NearestNeighbour)
	if err != nil {
		r.err = append(r.err, err)
		return r
	}
	vrt, err := gdal.GetDriverByName("VRT")

	options := []string{
		"SKIP_NOSOURCE=YES",
		"INIT_DEST=INIT_DEST",
		"UNIFIED_SRC_NODATA=YES",
	}
	warpedVRT = vrt.CreateCopy(tempFile.Name(), warpedVRT, 0, options, nil, nil)

	r.vrt = warpedVRT
	r.bandCount = r.vrt.RasterCount()
	return r
}

func (r *Read) Execute() error {
	if len(r.err) > 0 {
		return NewRunError().SetMessage(r.err[0].Error())
	}
	return nil
}

func (r *Read) Close() {
	r.ds.Close()
	r.vrt.Close()
	_ = r.tempFileVrt.Close()
	_ = os.Remove(r.tempFileVrt.Name())
}

func (r *Read) TileBoundsByTransform() (float64, float64, float64, float64) {
	transform := r.vrt.GeoTransform()
	minx := transform[0]
	maxy := transform[3]
	maxx := minx + transform[1]*float64(r.vrt.RasterXSize())
	miny := maxy + transform[5]*float64(r.vrt.RasterYSize())
	return minx, miny, maxx, maxy
}
