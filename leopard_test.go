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
		origCount     uint64
		recoveryCount uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
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
	originalCount := 10
	recoveryCount := 1
	bufferBytes := 640
	originalData := make([][]byte, originalCount)
	for i := 0; i < originalCount; i++ {
		originalData[i] = make([]byte, bufferBytes)
		rand.Read(originalData[i])
	}

	assert.NoError(t, Init())

	workCount := EncodeWorkCount(uint64(originalCount), uint64(recoveryCount))
	var encodeWork = make([][]byte, workCount)
	for i := uint(0); i < uint(workCount); i++ {
		encodeWork[i] = make([]byte, bufferBytes)
	}
	var decodeWork = make([][]byte, workCount)
	for i := uint(0); i < uint(workCount); i++ {
		decodeWork[i] = make([]byte, bufferBytes)
	}
	//res := LeoEncode(uint64(bufferBytes), uint32(originalCount), uint32(recoveryCount), uint32(workCount), originalData, encodeWork)
	//fmt.Println(res)

	err := LeoEncode2(uint64(bufferBytes), uint32(originalCount), uint32(recoveryCount), uint32(workCount), originalData, encodeWork)
	assert.Equal(t, err, 0)

	//fmt.Println(encodeWork)
	err = LeoDecode2(uint64(bufferBytes), uint32(originalCount), uint32(recoveryCount), uint32(workCount), originalData, encodeWork, decodeWork)
	//fmt.Println(originalData)
	//fmt.Println(decodeWork)
	fmt.Println(err)
	assert.Equal(t, err, 0)
	//assert.EqualValues(t, originalData, decodeWork)
	for i := 0; i < int(workCount); i++ {
		fmt.Println("---")
		fmt.Println(originalData[i])
		fmt.Println(decodeWork[i])
		fmt.Println("---")

	}
}
