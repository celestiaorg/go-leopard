package leopard

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/klauspost/reedsolomon"
)

// FuzzCompareImplementations is a fuzzer that compares go-leopard with
// the Go port from github.com/klauspost/reedsolomon.
func FuzzCompareImplementations(f *testing.F) {
	f.Fuzz(func(t *testing.T, seed int64, nshards, nloss, shardLen uint16) {
		r := rand.New(rand.NewSource(seed))
		// Ceil nshards to an even number.
		nshards = (nshards + 1) &^ 1
		// Ceil to multiple of 64 bytes.
		shardLen = (shardLen + 63) &^ 63
		shards := make([][]byte, nshards)
		ndata := len(shards) / 2
		for i := range shards {
			shards[i] = make([]byte, shardLen)
			if i < ndata {
				r.Read(shards[i])
			}
		}
		cppleoChecksum, cpperr := Encode(shards[:ndata])
		enc, goerr := reedsolomon.New(ndata, ndata, reedsolomon.WithLeopardGF(true))
		if goerr == nil {
			goerr = enc.Encode(shards)
		}
		if (goerr != nil) != (cpperr != nil) {
			t.Fatalf("one implementation returned an error, while the other didn't: Go error: %q vs C++ error: %v", goerr, cpperr)
		}
		if goerr != nil {
			return
		}
		goleoChecksum := shards[ndata:]
		if !reflect.DeepEqual(cppleoChecksum, goleoChecksum) {
			t.Fatal("checksum mismatch")
		}
	})
}
