package tile_gdal

import "github.com/lukeroth/gdal"

func CreateSpatialReference(ds gdal.Dataset, epsg int) (string, error) {
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
