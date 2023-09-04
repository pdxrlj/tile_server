package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var C *Config

type Config struct {
	Tile Tile
}

type Tile struct {
	ZoomMax       int    `mapstructure:"zoom_max"`
	ZoomMin       int    `mapstructure:"zoom_min"`
	InputFilename string `mapstructure:"input_filename"`
	OutFolder     string `mapstructure:"out_folder"`
	Style         string `mapstructure:"style"`
	Concurrency   int    `mapstructure:"concurrency"`
}

func (a *Config) Marsh() error {
	return viper.Unmarshal(a)
}

func (a *Config) GetZoomMax() int {
	return a.Tile.ZoomMax
}

func (a *Config) GetZoomMin() int {
	return a.Tile.ZoomMin
}

func (a *Config) GetInputFilename() string {
	return a.Tile.InputFilename
}

func (a *Config) GetTileStyle() string {
	return a.Tile.Style
}

func (a *Config) GetOutFolder() string {
	return a.Tile.OutFolder
}

func (a *Config) GetConcurrency() int {
	return a.Tile.Concurrency
}

func ViperBindFlagsAlias(command cobra.Command) error {
	err := viper.BindPFlag("tile.zoom_max", command.PersistentFlags().Lookup("zoom_max"))
	if err != nil {
		return err
	}

	err = viper.BindPFlag("tile.zoom_min", command.PersistentFlags().Lookup("zoom_min"))
	if err != nil {
		return err
	}

	err = viper.BindPFlag("tile.input_filename", command.PersistentFlags().Lookup("input_filename"))
	if err != nil {
		return err
	}

	err = viper.BindPFlag("tile.out_folder", command.PersistentFlags().Lookup("out_folder"))
	if err != nil {
		return err
	}

	err = viper.BindPFlag("tile.style", command.PersistentFlags().Lookup("style"))
	if err != nil {
		return err
	}

	err = viper.BindPFlag("tile.concurrency", command.PersistentFlags().Lookup("concurrency"))
	if err != nil {
		return err
	}

	return nil
}

func UnmarshalToConfig(dst interface{}) error {
	err := viper.Unmarshal(dst)
	if err != nil {
		return err
	}
	return nil
}
