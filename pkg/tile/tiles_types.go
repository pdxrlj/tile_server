package tile

type TileOption func(*Tile)

func SetTileSize(tileSize int) TileOption {
	return func(r *Tile) {
		r.tileSize = tileSize
	}
}

func SetConcurrency(concurrency int) TileOption {
	return func(r *Tile) {
		r.Concurrency = concurrency
	}
}

func SetTileStyle(style string) TileOption {
	return func(r *Tile) {
		r.style = style
	}
}

func SetOutFolder(outFolder string) TileOption {
	return func(r *Tile) {
		r.outFolder = outFolder
	}
}

func SetInputFilename(inputFilename string) TileOption {
	return func(r *Tile) {
		r.inputFilename = inputFilename
	}
}

func SetZoomMaxMin(zoomMax, zoomMin int) TileOption {
	return func(r *Tile) {
		r.ZoomMax = zoomMax
		r.ZoomMin = zoomMin
	}
}

func DefaultTile() *Tile {
	return &Tile{
		tileSize:  256,
		outFolder: "",
		querySize: 256 * 4,
	}
}
