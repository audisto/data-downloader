package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChs(t *testing.T) {
	assert.Equal(t, chs(4, "c"), "cccc", "they should be equal")
}
