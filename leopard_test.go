package leopard

import (
	"reflect"
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
	type args struct {
		bufferBytes   uint64
		originalData  [][]byte
		originalCount uint
		recoveryCount uint
	}
	tests := []struct {
		name    string
		args    args
		want    [][]byte
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encode(tt.args.bufferBytes, tt.args.originalData, tt.args.originalCount, tt.args.recoveryCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				//t.Errorf("Encode() got = %v, want %v", got, tt.want)
			}
		})
	}
}
