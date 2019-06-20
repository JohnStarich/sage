package math

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {
	assert.Equal(t, 0, MinInt(0, 1))
	assert.Equal(t, 0, MinInt(1, 0))
	assert.Equal(t, 1, MinInt(1, 1))
	assert.Equal(t, -2, MinInt(-2, -1))
}

func TestMax(t *testing.T) {
	assert.Equal(t, 1, MaxInt(0, 1))
	assert.Equal(t, 1, MaxInt(1, 0))
	assert.Equal(t, 1, MaxInt(1, 1))
	assert.Equal(t, -1, MaxInt(-2, -1))
}
