package cmd

import (
	"fmt"
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
			if !os.IsExist(err) {
				fmt.Printf("file %s not exist\n", testImgPath)
				return err
			}
		}

		gds, err := tile_gdal.Open(testImgPath)
		if err != nil {
			fmt.Printf("Open error: %s\n", testImgPath)
			return err
		}
		err = gds.WrapVrt()
		return err
	},
}

// Execute executes the root command.
func Execute() error {
	return root.Execute()
}

func init() {

}
