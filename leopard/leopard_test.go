package leopard

import (
	"bytes"
	"crypto/md5"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeWorkCount(t *testing.T) {
	type args struct {
		origCount     uint32
		recoveryCount uint32
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{"edge case: 1, 1", args{1, 1}, 1},
		{"edge case: 1, 2", args{1, 2}, 2},
		{"edge case: 1, 255", args{1, 255}, 255},
		{"edge case 255, 1", args{255, 1}, 1},
		{"2*2", args{5, 2}, 4},
		{"2*4", args{5, 4}, 8},
		{"2*8", args{5, 5}, 16},
		{"2*8", args{5, 6}, 16},
		{"2*8", args{5, 7}, 16},
		{"2*8", args{5, 8}, 16},
		{"2*16", args{5, 9}, 32},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := leoEncodeWorkCount(tt.args.origCount, tt.args.recoveryCount); got != tt.want {
				t.Errorf("EncodeWorkCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Simple test that reproduces what leopard/tests/benchmark.cpp:378 does
// but much simpler and just to verify that we can successfully
// go from leoEncode back with leoDecode (using Golang `[][]byte`s).
func TestItWorks(t *testing.T) {
	const originalCount = 1024
	const recoveryCount = 512 // can lose upto half of the total data (orig or recovery)
	const bufferBytes = 64000
	originalData := make([][]byte, originalCount)
	for i := 0; i < originalCount; i++ {
		originalData[i] = make([]byte, bufferBytes)
		checkedRandBytes(originalData[i])
	}
	require.NoError(t, Init())

	encodeWorkCount := leoEncodeWorkCount(uint32(originalCount), uint32(recoveryCount))
	decodeWorkCount := leoDecodeWorkCount(uint32(originalCount), uint32(recoveryCount))
	assert.Equal(t, originalCount, int(encodeWorkCount))
	assert.Equal(t, originalCount*2, int(decodeWorkCount))

	var encodeWork = make([][]byte, encodeWorkCount)
	for i := uint(0); i < uint(encodeWorkCount); i++ {
		encodeWork[i] = make([]byte, bufferBytes)
	}
	var decodeWork = make([][]byte, decodeWorkCount)
	for i := uint(0); i < uint(decodeWorkCount); i++ {
		decodeWork[i] = make([]byte, bufferBytes)
	}

	origDataPtr := copyToCmallocedPtrs(originalData)
	encodeWorkPtr := copyToCmallocedPtrs(encodeWork)
	err := leoEncode(uint64(bufferBytes), uint32(originalCount), uint32(recoveryCount), encodeWorkCount, origDataPtr, encodeWorkPtr)
	require.Equal(t, err, leopardSuccess)

	decodeWorkPtr := copyToCmallocedPtrs(decodeWork)

	// lose half the orig data:
	for i := 0; i < originalCount/2; i++ {
		origDataPtr[i] = nil
	}

	// lose some recovery data:
	freeAndNilBuf(encodeWorkPtr[5])
	freeAndNilBuf(encodeWorkPtr[10])
	freeAndNilBuf(encodeWorkPtr[23])

	err = leoDecode(uint64(bufferBytes), uint32(originalCount), uint32(recoveryCount), decodeWorkCount, origDataPtr, encodeWorkPtr, decodeWorkPtr)
	require.Equal(t, leopardSuccess, err)
	for i := 0; i < originalCount; i++ {
		if origDataPtr[i] == nil {
			d := *(*[bufferBytes]byte)(decodeWorkPtr[i])
			assert.Equal(t, true, checkBytes(d[:]))
		}
	}
}

func checkedRandBytes(p []byte) {
	if len(p) <= md5.Size {
		panic("provided slice is too small")
	}
	raw := make([]byte, len(p)-md5.Size)
	rand.Read(raw)
	chksm := md5.Sum(raw)
	copy(p, raw)
	copy(p[len(p)-md5.Size:], chksm[:])
}

func checkBytes(p []byte) bool {
	if len(p) <= md5.Size {
		panic("provided slice is too small")
	}
	data := p[:len(p)-md5.Size]
	readChksm := p[len(p)-md5.Size:]
	chksm := md5.Sum(data)
	if !bytes.Equal(readChksm, chksm[:]) {
		return false
	}
	return true
}
