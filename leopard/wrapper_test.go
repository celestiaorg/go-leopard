package leopard

import (
	"testing"
	"unsafe"

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
	encoded, err := Encode(originalData)
	assert.NoError(t, err)
	assert.NotNil(t, encoded)
	assert.Equal(t, originalCount, len(encoded))
}

func TestEncodeDecodeRoundtripSimple(t *testing.T) {
	err := Init()
	require.NoError(t, err)
	const originalCount = 1024
	const bufferBytes = 6400
	originalData := make([][]byte, originalCount)
	for i := 0; i < originalCount; i++ {
		originalData[i] = make([]byte, bufferBytes)
		checkedRandBytes(originalData[i])
	}
	encoded, err := Encode(originalData)
	require.NoError(t, err)
	assert.EqualValues(t, len(encoded), originalCount)

	// lose half the orig data:
	for i := 0; i < originalCount/2; i++ {
		originalData[i] = nil
	}

	dec, err := Decode(originalData, encoded)
	require.NoError(t, err)
	assert.Equal(t, 2*originalCount, len(dec))
	for i := 0; i < originalCount; i++ {
		if originalData[i] == nil {
			// see if we recovered that missing data:
			assert.Equal(t, true, checkBytes(dec[i]))
		}
	}
}

func TestMemRoundTrip(t *testing.T) {
	const originalCount = 128
	const bufferBytes = 640

	originalData := make([][]byte, originalCount)
	for i := 0; i < originalCount; i++ {
		originalData[i] = make([]byte, bufferBytes)
		checkedRandBytes(originalData[i])
	}
	ptrs := mockScopeFunc1(originalData)
	result := mockScopeFunc2(originalCount, ptrs, bufferBytes)
	// free pointers and see if we run into any problem:
	free(ptrs)
	assert.EqualValues(t, originalData, result)
}

func mockScopeFunc2(originalCount int, ptrs []unsafe.Pointer, bufferBytes int) [][]byte {
	result := make([][]byte, originalCount)
	toGoByte(ptrs, result, bufferBytes)
	return result
}

func mockScopeFunc1(originalData [][]byte) []unsafe.Pointer {
	ptrs := copyToCmallocedPtrs(originalData)
	return ptrs
}
