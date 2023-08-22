package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/pdxrlj/tile_server/config"
	"github.com/pdxrlj/tile_server/pkg/tile"
)

const (
	testImgPath = "/Users/ruanlianjun/Desktop/tif/老寨新寨1.tif"
)

var root = cobra.Command{
	Use:   "tile",
	Short: "tile is a tile map server",
	Long:  "tile is a tile map server",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := config.UnmarshalToConfig(&config.C)
		if err != nil {
			return err
		}

		if _, err := os.Stat(testImgPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return err
			}
		}

		if err := tile.NewTile(
			tile.SetInputFilename(config.C.GetInputFilename()),
			tile.SetZoomMaxMin(config.C.GetZoomMax(), config.C.GetZoomMin()),
			tile.SetOutFolder(config.C.GetOutFolder()),
		).GenerateGdalReadWindows().CuttingToImg().Close(); err != nil {
			fmt.Printf("tile.NewTile error:%v\n", err)
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
	configPath, err := config.GetConfigPath()
	if err != nil {
		panic(err)
	}

	viper.AddConfigPath(configPath)
	viper.SetConfigName("app")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	err = viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		panic(err)
	}

	err = viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	// app command line flags
	CommandLine()

	// app command bind flags alias
	err = config.ViperBindFlagsAlias(root)
	if err != nil {
		panic(err)
	}
}

func CommandLine() {
	root.PersistentFlags().IntP("zoom_max", "u", 10, "maxzoom")
	root.PersistentFlags().IntP("zoom_min", "l", 0, "minzoom")
	root.PersistentFlags().StringP("input_filename", "i", "", "input_filename")
	root.PersistentFlags().StringP("out_folder", "o", "", "out_folder")
}
