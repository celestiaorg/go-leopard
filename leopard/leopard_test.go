package leopard

import (
	"testing"
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
			if got := LeoEncodeWorkCount(tt.args.origCount, tt.args.recoveryCount); got != tt.want {
				t.Errorf("EncodeWorkCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
