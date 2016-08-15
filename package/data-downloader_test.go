package dataDownloader

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChs(t *testing.T) {
	assert.Equal(t, "cccc", chs(4, "c"), "they should be equal")
}

func TestNextChunkNumber(t *testing.T) {

	resumer := Resumer{}
	resumer.TotalElements = 100
	resumer.DoneElements = 0
	resumer.chunkSize = 100

	nextChunkNumber, skipNRows := resumer.nextChunkNumber()

	assert.Equal(t, int64(0), nextChunkNumber, "they should be equal")
	assert.Equal(t, int64(0), skipNRows, "they should be equal")

	///

	resumer.TotalElements = 100
	resumer.DoneElements = 0
	resumer.chunkSize = 10

	nextChunkNumber, skipNRows = resumer.nextChunkNumber()

	assert.Equal(t, int64(0), nextChunkNumber, "they should be equal")
	assert.Equal(t, int64(0), skipNRows, "they should be equal")

	///

	resumer.TotalElements = 100
	resumer.DoneElements = 10
	resumer.chunkSize = 10

	nextChunkNumber, skipNRows = resumer.nextChunkNumber()

	assert.Equal(t, int64(1), nextChunkNumber, "they should be equal")
	assert.Equal(t, int64(0), skipNRows, "they should be equal")

	///

	resumer.TotalElements = 100
	resumer.DoneElements = 9
	resumer.chunkSize = 10

	nextChunkNumber, skipNRows = resumer.nextChunkNumber()

	assert.Equal(t, int64(0), nextChunkNumber, "they should be equal")
	assert.Equal(t, int64(9), skipNRows, "they should be equal")

	///

	resumer.TotalElements = 100
	resumer.DoneElements = 11
	resumer.chunkSize = 10

	nextChunkNumber, skipNRows = resumer.nextChunkNumber()

	assert.Equal(t, int64(1), nextChunkNumber, "they should be equal")
	assert.Equal(t, int64(1), skipNRows, "they should be equal")

	///

	resumer.TotalElements = 99
	resumer.DoneElements = 98
	resumer.chunkSize = 10

	nextChunkNumber, skipNRows = resumer.nextChunkNumber()

	assert.Equal(t, int64(98), nextChunkNumber, "they should be equal")
	assert.Equal(t, int64(0), skipNRows, "they should be equal")

	///

	resumer.TotalElements = 99
	resumer.DoneElements = 99
	resumer.chunkSize = 10

	nextChunkNumber, skipNRows = resumer.nextChunkNumber()

	assert.Equal(t, int64(9), nextChunkNumber, "they should be equal")
	assert.Equal(t, int64(1), skipNRows, "they should be equal")
	assert.Equal(t, int64(1), resumer.chunkSize, "they should be equal")

}
