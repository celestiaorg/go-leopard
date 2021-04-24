package leopard

import "math/bits"

//------------------------------------------------------------------------------
// Portable Intrinsics

// Returns highest bit index 0..31 where the first non-zero bit is found
// Precondition: x != 0
func lastNonzeroBit32(x uint32) uint32 {
	// Note: Ignoring return value of 0 because x != 0
	return 31 - uint32(bits.LeadingZeros32(x))
}

// Returns next power of two at or above given value
func nextPow2(n uint32) uint32 {
	return 2 << lastNonzeroBit32(n-1)
}

//------------------------------------------------------------------------------
// XOR Memory
//
// This works for both 8-bit and 16-bit finite fields

// x[] ^= y[]
func xor_mem(
	vx []byte,
	vy []byte,
	bytes uint64,
) {
	asm_xor_mem(vx, vy, bytes)
}

func vectorXOR(bytes uint64, count uint32, x [][]byte, y [][]byte) {
	// TODO vector optimizations
	for i := uint32(0); i < count; i++ {
		xor_mem(x[i], y[i], bytes)
	}
}

func vectorXOR_Threads(bytes uint64, count uint32, x [][]byte, y [][]byte) {
	// TODO goroutines
	vectorXOR_Threads(bytes, count, x, y)
}
