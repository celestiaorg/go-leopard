package leopard

import (
	"bytes"
	"crypto/md5"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLeo(t *testing.T) {
	assert.NoError(t, Init())
}

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
			if got := EncodeWorkCount(tt.args.origCount, tt.args.recoveryCount); got != tt.want {
				t.Errorf("EncodeWorkCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Simple test that reproduces what leopard/tests/benchmark.cpp:378 does
// but much simpler and just to verify that we can succuesfully
// go from leoEncode back with leoDecode (unsing Golang `[][]byte`s).
func TestItWorks(t *testing.T) {
	const originalCount = 1000
	const recoveryCount = 100
	const bufferBytes = 64000
	originalData := make([][]byte, originalCount)
	for i := 0; i < originalCount; i++ {
		originalData[i] = make([]byte, bufferBytes)
		checkedRandBytes(originalData[i])
	}
	require.NoError(t, Init())

	workCount := EncodeWorkCount(uint32(originalCount), uint32(recoveryCount))
	decodeWorkCount := leoDecodeWorkCount(uint32(originalCount), uint32(recoveryCount))
	var encodeWork = make([][]byte, workCount)
	for i := uint(0); i < uint(workCount); i++ {
		encodeWork[i] = make([]byte, bufferBytes)
	}
	var decodeWork = make([][]byte, decodeWorkCount)
	for i := uint(0); i < uint(decodeWorkCount); i++ {
		decodeWork[i] = make([]byte, bufferBytes)
	}

	origDataPtr := convert(originalData)
	encodeWorkPtr := convert(encodeWork)
	err := leoEncode(uint64(bufferBytes), uint32(originalCount), uint32(recoveryCount), workCount, origDataPtr, encodeWorkPtr)
	require.Equal(t, err, leopardSuccess)

	decodeWorkPtr := convert(decodeWork)

	// lose some orig data:
	// Todo: we'd need to free the C.malloc'd memory
	origDataPtr[11] = nil
	origDataPtr[13] = nil
	origDataPtr[23] = nil

	// lose some recovery data:
	// Todo: we'd need to free the C.malloc'd memory
	encodeWorkPtr[5] = nil
	encodeWorkPtr[10] = nil
	encodeWorkPtr[23] = nil

	err = leoDecode(uint64(bufferBytes), uint32(originalCount), uint32(recoveryCount), decodeWorkCount, origDataPtr, encodeWorkPtr, decodeWorkPtr)
	require.Equal(t, err, leopardSuccess)
	for i := 0; i < int(originalCount); i++ {
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
