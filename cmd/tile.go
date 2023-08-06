package cmd

import (
	"errors"
	"os"

	"github.com/pdxrlj/tile_server/pkg/tile_gdal"
	"github.com/spf13/cobra"
)

const (
	testImgPath = "/home/pdx/resource/黄蜡湾新村.tif"
)

var root = cobra.Command{
	Use:   "tile",
	Short: "tile is a tile map server",
	Long:  "tile is a tile map server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(testImgPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return err
			}
		}

		gds, err := tile_gdal.OpenTile(tile_gdal.SetTileInputFilename(testImgPath))
		if err != nil {
			return err
		}

		if err = gds.TileRange().MakeTileJobInfo().GenerateBaseTile(); err != nil {
			return err
		}
		return nil
	},
}

// Execute executes the root command.
func Execute() error {
	return root.Execute()
}

func init() {

}
