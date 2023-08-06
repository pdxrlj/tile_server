package tile_gdal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapToVrt(t *testing.T) {
	gds, err := Open("C:\\Users\\ruanyu\\Desktop\\黄蜡湾新村.tif")
	assert.NoError(t, err)
	err = gds.WrapVrt()
	assert.NoError(t, err)
}
