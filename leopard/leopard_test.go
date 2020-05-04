package leopard

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestEncode(t *testing.T) {
	originalCount := 1
	recoveryCount := 1
	const bufferBytes = 640
	originalData := make([][]byte, originalCount)
	for i := 0; i < originalCount; i++ {
		originalData[i] = make([]byte, bufferBytes)
		rand.Read(originalData[i])
	}
	assert.NoError(t, Init())

	workCount := EncodeWorkCount(uint32(originalCount), uint32(recoveryCount))
	var encodeWork = make([][]byte, workCount)
	for i := uint(0); i < uint(workCount); i++ {
		encodeWork[i] = make([]byte, bufferBytes)
	}
	var decodeWork = make([][]byte, workCount)
	for i := uint(0); i < uint(workCount); i++ {
		decodeWork[i] = make([]byte, bufferBytes)
	}

	origDataPtr := convert(originalData)
	encodeWorkPtr := convert(encodeWork)
	err := leoEncode(uint64(bufferBytes), uint32(originalCount), uint32(recoveryCount), uint32(workCount), origDataPtr, encodeWorkPtr)
	assert.Equal(t, err, leopardSuccess)

	decodeWorkPtr := convert(decodeWork)
	err = leoDecode(uint64(bufferBytes), uint32(originalCount), uint32(recoveryCount), uint32(workCount), origDataPtr, encodeWorkPtr, decodeWorkPtr)
	fmt.Println(err)
	assert.Equal(t, err, leopardSuccess)
	for i := 0; i < int(workCount); i++ {
		assert.EqualValues(t, *(*[bufferBytes]byte)(decodeWorkPtr[i]), *(*[bufferBytes]byte)(origDataPtr[i]))
		fmt.Println("---", i)
	}
}
