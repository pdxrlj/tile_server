package tile_gdal

import "errors"

var (
	ErrInputFilename = errors.New("input filename is empty")
)

type RunError struct {
	message string
}

func NewRunError() RunError {
	return RunError{}
}

func (e RunError) SetMessage(message string) RunError {
	e.message = message
	return e
}

func (e RunError) Error() string {
	return e.message
}
