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
/*
type LEO_M256 = __m256i

// x[] ^= y[]
func xor_mem(
	vx []byte,
	vy []byte,
	bytes uint64,
) {
	x32 := []LEO_M256(vx)
	y32 := []LEO_M256(vy)
	for bytes >= uint64(128) {
		x0 LEO_M256 := _mm256_xor_si256(_mm256_loadu_si256(x32), _mm256_loadu_si256(y32))
		x1 LEO_M256 := _mm256_xor_si256(_mm256_loadu_si256(x32+1), _mm256_loadu_si256(y32+1))
		x2 LEO_M256 := _mm256_xor_si256(_mm256_loadu_si256(x32+2), _mm256_loadu_si256(y32+2))
		x3 LEO_M256 := _mm256_xor_si256(_mm256_loadu_si256(x32+3), _mm256_loadu_si256(y32+3))
		_mm256_storeu_si256(x32, x0)
		_mm256_storeu_si256(x32+1, x1)
		_mm256_storeu_si256(x32+2, x2)
		_mm256_storeu_si256(x32+3, x3)
		x32 += 4
		y32 += 4
		bytes -= 128
	}
	if bytes > 0 {
		const LEO_M256 x0 = _mm256_xor_si256(_mm256_loadu_si256(x32), _mm256_loadu_si256(y32))
		const LEO_M256 x1 = _mm256_xor_si256(_mm256_loadu_si256(x32+1), _mm256_loadu_si256(y32+1))
		_mm256_storeu_si256(x32, x0)
		_mm256_storeu_si256(x32+1, x1)
	}
	return
}

/*

//------------------------------------------------------------------------------
// SIMD-Safe Aligned Memory Allocations

static const unsigned kAlignmentBytes = LEO_ALIGN_BYTES;

static LEO_FORCE_INLINE uint8_t* SIMDSafeAllocate(size_t size)
{
    uint8_t* data = (uint8_t*)calloc(1, kAlignmentBytes + size);
    if (!data)
        return nullptr;
    unsigned offset = (unsigned)((uintptr_t)data % kAlignmentBytes);
    data += kAlignmentBytes - offset;
    data[-1] = (uint8_t)offset;
    return data;
}

static LEO_FORCE_INLINE void SIMDSafeFree(void* ptr)
{
    if (!ptr)
        return;
    uint8_t* data = (uint8_t*)ptr;
    unsigned offset = data[-1];
    if (offset >= kAlignmentBytes)
    {
        LEO_DEBUG_BREAK; // Should never happen
        return;
    }
    data -= kAlignmentBytes - offset;
    free(data);
}
*/
