package leopard

import (
	"fmt"
	"math/rand"
	"testing"
)

func fillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}

func benchmarkEncode(b *testing.B, dataShards, shardSize int) {
	shards := make([][]byte, dataShards)
	for s := range shards {
		shards[s] = make([]byte, shardSize)
	}

	rand.Seed(0)
	for s := 0; s < dataShards; s++ {
		fillRandom(shards[s])
	}

	b.SetBytes(int64(shardSize * dataShards * 2))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Encode(shards)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkDecode(b *testing.B, dataShards, shardSize int) {
	shards := make([][]byte, dataShards)
	for s := range shards {
		shards[s] = make([]byte, shardSize)
	}

	rand.Seed(0)
	for s := 0; s < dataShards; s++ {
		fillRandom(shards[s])
	}
	parity, err := Encode(shards)
	if err != nil {
		b.Fatal(err)
	}

	// Clear data shards.
	for s := range shards {
		shards[s] = nil
	}

	b.SetBytes(int64(shardSize * dataShards * 2))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Decode(shards, parity)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark 1K encode with symmetric shard sizes.
func BenchmarkEncode1K(b *testing.B) {
	for shards := 16; shards < 65536; shards *= 2 {
		b.Run(fmt.Sprintf("%vx%v", shards, shards), func(b *testing.B) {
			benchmarkEncode(b, shards, 1024)
		})
	}
}

// Benchmark 1K decode with symmetric shard sizes.
func BenchmarkDecode1K(b *testing.B) {
	for shards := 16; shards < 65536; shards *= 2 {
		b.Run(fmt.Sprintf("%vx%v", shards, shards), func(b *testing.B) {
			benchmarkDecode(b, shards, 1024)
		})
	}
}
