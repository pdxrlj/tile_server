package gdal

import "math"

type Mercator struct {
	TileSize          int
	OriginShift       float64
	InitialResolution float64
}

type MercatorOptions func(mercator *Mercator)

func WithTileSize(tileSize int) MercatorOptions {
	return func(mercator *Mercator) {
		mercator.TileSize = tileSize
	}
}

func WithOriginShift(originShift float64) MercatorOptions {
	return func(mercator *Mercator) {
		mercator.OriginShift = originShift
	}
}

func WithInitialResolution(initialResolution float64) MercatorOptions {
	return func(mercator *Mercator) {
		mercator.InitialResolution = initialResolution
	}
}

func DefaultMercator() *Mercator {
	return &Mercator{
		TileSize:          256,
		OriginShift:       2 * math.Pi * 6378137 / 2,
		InitialResolution: 2 * math.Pi * 6378137 / 256,
	}
}

func NewMercator(options ...MercatorOptions) *Mercator {
	m := DefaultMercator()
	for _, opt := range options {
		opt(m)
	}
	return m
}

func (m *Mercator) Resolution(zoom int) float64 {
	return m.InitialResolution / math.Pow(2, float64(zoom))
}

// MeterToTile MetersToPixels converts meters to tx, ty
// tx, ty: tile coordinates
// mx, my: meters
// zoom: zoom level
func (m *Mercator) MeterToTile(zoom int, mx, my float64) (int, int) {
	px, py := m.MetersToPixels(zoom, mx, my)
	tx, ty := m.PixelsToTile(px, py)
	return tx, ty
}

// MetersToPixels converts meters to pixels
// mx, my: meters
// zoom: zoom level
func (m *Mercator) MetersToPixels(zoom int, mx, my float64) (float64, float64) {
	res := m.Resolution(zoom)
	px := (mx + m.OriginShift) / res
	py := (my + m.OriginShift) / res
	return px, py
}

// PixelsToTile converts pixels to tile coordinates
// px, py: pixels
// tx, ty: tile coordinates
func (m *Mercator) PixelsToTile(px, py float64) (int, int) {
	tx := int(math.Ceil(px/float64(m.TileSize)) - 1)
	ty := int(math.Ceil(py/float64(m.TileSize)) - 1)
	return tx, ty
}

func (m *Mercator) tileToLat(ty, tz int) float64 {
	n := math.Pi - 2.0*math.Pi*float64(ty)/math.Pow(2.0, float64(tz))
	lat := 180.0 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(-n)))
	return lat
}

func (m *Mercator) tileToLon(tx, tz int) float64 {
	lon := float64(tx)/math.Pow(2.0, float64(tz))*360.0 - 180.0
	return lon
}

func (m *Mercator) tileToLonLat(tx, ty, tz int) (float64, float64) {
	lat := m.tileToLat(ty, tz)
	lon := m.tileToLon(tx, tz)
	return lon, lat
}

// TileBounds returns the bounds of a tile in meters
// tz: zoom level
// tx, ty: tile coordinates
func (m *Mercator) TileBounds(tz, tx, ty int) (float64, float64, float64, float64) {
	minx, miny := m.PixelsToMeters(float64(tx*m.TileSize), float64(ty*m.TileSize), tz)
	maxx, maxy := m.PixelsToMeters(float64((tx+1)*m.TileSize), float64((ty+1)*m.TileSize), tz)
	return minx, miny, maxx, maxy
}

func (m *Mercator) PixelsToMeters(px, py float64, tz int) (float64, float64) {
	res := m.Resolution(tz)
	mx := px*res - m.OriginShift
	my := py*res - m.OriginShift
	return mx, my
}
