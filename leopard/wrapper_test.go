package leopard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLeo(t *testing.T) {
	assert.NoError(t, Init())
}

func TestEncodeSimple(t *testing.T) {
	const originalCount = 64
	const bufferBytes = 640

	originalData := make([][]byte, originalCount)
	for i := 0; i < originalCount; i++ {
		originalData[i] = make([]byte, bufferBytes)
		checkedRandBytes(originalData[i])
	}
	err := Init()
	require.NoError(t, err)
	res, err := Encode(originalData)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, originalCount, len(res))
}
