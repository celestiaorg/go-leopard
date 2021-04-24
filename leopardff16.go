package leopard

import (
	"bytes"
)

//------------------------------------------------------------------------------
// Datatypes and Constants

// Finite field element type
type ffe_t = uint16

// Number of bits per element
const kBits uint32 = 16

// Finite field order: Number of elements in the field
const kOrder uint32 = 65536

// Modulus for field operations
const kModulus ffe_t = 65535

// LFSR Polynomial that generates the field elements
const kPolynomial uint32 = 0x1002D

// Basis used for generating logarithm tables
// Using the Cantor basis here enables us to avoid a lot of extra calculations
// when applying the formal derivative in decoding.
var kCantorBasis = [...]ffe_t{
	0x0001, 0xACCA, 0x3C0E, 0x163E,
	0xC582, 0xED2E, 0x914C, 0x4012,
	0x6C98, 0x10D8, 0x6A72, 0xB900,
	0xFDB8, 0xFB34, 0xFF38, 0x991E,
}

//------------------------------------------------------------------------------
// Field Operations

// z = x + y (mod kModulus)
func addMod(a, b ffe_t) ffe_t {
	sum := uint32(a) + uint32(b)

	// Partial reduction step, allowing for kModulus to be returned
	return ffe_t(sum + (sum >> kBits))
}

// z = x - y (mod kModulus)
func subMod(a, b ffe_t) ffe_t {
	dif := uint32(a) - uint32(b)

	// Partial reduction step, allowing for kModulus to be returned
	return ffe_t(dif + (dif >> kBits))
}

//------------------------------------------------------------------------------
// Fast Walsh-Hadamard Transform (FWHT) (mod kModulus)

// {a, b} = {a + b, a - b} (Mod Q)
func fwht_2(a, b ffe_t) (ffe_t, ffe_t) {
	sum := addMod(a, b)
	dif := subMod(a, b)
	return sum, dif
}

func fwht_4(data []ffe_t, s uint32) {
	s2 := s << 1

	t0 := data[0]
	t1 := data[s]
	t2 := data[s2]
	t3 := data[s2+s]

	t0, t1 = fwht_2(t0, t1)
	t2, t3 = fwht_2(t2, t3)
	t0, t2 = fwht_2(t0, t2)
	t1, t3 = fwht_2(t1, t3)

	data[0] = t0
	data[s] = t1
	data[s2] = t2
	data[s2+s] = t3
}

// Decimation in time (DIT) Fast Walsh-Hadamard Transform
// Unrolls pairs of layers to perform cross-layer operations in registers
// m_truncated: Number of elements that are non-zero at the front of data
func fwht(data []ffe_t, m uint32, m_truncated uint32) {
	// Decimation in time: Unroll 2 layers at a time
	dist := uint32(1)
	dist4 := uint32(4)
	for ; dist4 <= m; dist4 <<= 2 {
		// For each set of dist*4 elements:
		for r := uint32(0); r < m_truncated; r += dist4 {
			// For each set of dist elements:
			i_end := r + dist
			for i := r; i < i_end; i++ {
				fwht_4(data[i:], dist)
			}
		}
		dist = dist4
	}

	// If there is one layer left:
	if dist < m {
		for i := uint32(0); i < dist; i++ {
			data[i], data[i+dist] = fwht_2(data[i], data[i+dist])
		}
	}
}

//------------------------------------------------------------------------------
// Logarithm Tables

var logLUT [kOrder]ffe_t
var expLUT [kOrder]ffe_t

// Returns a * Log(b)
//
// Note that this operation is not a normal multiplication in a finite
// field because the right operand is already a logarithm.  This is done
// because it moves K table lookups from the Decode() method into the
// initialization step that is less performance critical.  The LogWalsh[]
// table below contains precalculated logarithms so it is easier to do
// all the other multiplies in that form as well.
func multiplyLog(a, log_b ffe_t) ffe_t {
	if a == 0 {
		return 0
	}
	return expLUT[addMod(logLUT[a], log_b)]
}

// Initialize LogLUT[], ExpLUT[]
func initializeLogarithmTables() {
	// LFSR table generation:

	state := uint32(1)
	for i := uint16(0); i < kModulus; i++ {
		expLUT[state] = ffe_t(i)
		state <<= 1
		if state >= kOrder {
			state ^= kPolynomial
		}
	}
	expLUT[0] = kModulus

	// Conversion to Cantor basis:

	logLUT[0] = 0
	for i := uint32(0); i < kBits; i++ {
		basis := kCantorBasis[i]
		width := uint32(1) << i

		for j := uint32(0); j < width; j++ {
			logLUT[j+width] = logLUT[j] ^ basis
		}
	}

	for i := uint32(0); i < kOrder; i++ {
		logLUT[i] = expLUT[logLUT[i]]
	}

	for i := uint32(0); i < kOrder; i++ {
		expLUT[logLUT[i]] = ffe_t(i)
	}

	expLUT[kModulus] = expLUT[0]
}

//------------------------------------------------------------------------------
// Multiplies

// The multiplication algorithm used follows the approach outlined in {4}.
// Specifically section 7 outlines the algorithm used here for 16-bit fields.
// The ALTMAP memory layout is used since there is no need to convert in/out.

// 4 * 256-bit registers
type multiply256LUT_t struct {
	lo [4 * 32]byte
	hi [4 * 32]byte
}

var multiply256LUT [kOrder]multiply256LUT_t

/*
#define LEO_MUL_TABLES_256(table, log_m) \
        const LEO_M256 T0_lo_##table = _mm256_loadu_si256(&Multiply256LUT[log_m].Lo[0]); \
        const LEO_M256 T1_lo_##table = _mm256_loadu_si256(&Multiply256LUT[log_m].Lo[1]); \
        const LEO_M256 T2_lo_##table = _mm256_loadu_si256(&Multiply256LUT[log_m].Lo[2]); \
        const LEO_M256 T3_lo_##table = _mm256_loadu_si256(&Multiply256LUT[log_m].Lo[3]); \
        const LEO_M256 T0_hi_##table = _mm256_loadu_si256(&Multiply256LUT[log_m].Hi[0]); \
        const LEO_M256 T1_hi_##table = _mm256_loadu_si256(&Multiply256LUT[log_m].Hi[1]); \
        const LEO_M256 T2_hi_##table = _mm256_loadu_si256(&Multiply256LUT[log_m].Hi[2]); \
        const LEO_M256 T3_hi_##table = _mm256_loadu_si256(&Multiply256LUT[log_m].Hi[3]);

// 256-bit {prod_lo, prod_hi} = {value_lo, value_hi} * log_m
#define LEO_MUL_256(value_lo, value_hi, table) { \
            LEO_M256 data_1 = _mm256_srli_epi64(value_lo, 4); \
            LEO_M256 data_0 = _mm256_and_si256(value_lo, clr_mask); \
            data_1 = _mm256_and_si256(data_1, clr_mask); \
            prod_lo = _mm256_shuffle_epi8(T0_lo_##table, data_0); \
            prod_hi = _mm256_shuffle_epi8(T0_hi_##table, data_0); \
            prod_lo = _mm256_xor_si256(prod_lo, _mm256_shuffle_epi8(T1_lo_##table, data_1)); \
            prod_hi = _mm256_xor_si256(prod_hi, _mm256_shuffle_epi8(T1_hi_##table, data_1)); \
            data_0 = _mm256_and_si256(value_hi, clr_mask); \
            data_1 = _mm256_srli_epi64(value_hi, 4); \
            data_1 = _mm256_and_si256(data_1, clr_mask); \
            prod_lo = _mm256_xor_si256(prod_lo, _mm256_shuffle_epi8(T2_lo_##table, data_0)); \
            prod_hi = _mm256_xor_si256(prod_hi, _mm256_shuffle_epi8(T2_hi_##table, data_0)); \
            prod_lo = _mm256_xor_si256(prod_lo, _mm256_shuffle_epi8(T3_lo_##table, data_1)); \
            prod_hi = _mm256_xor_si256(prod_hi, _mm256_shuffle_epi8(T3_hi_##table, data_1)); }

// {x_lo, x_hi} ^= {y_lo, y_hi} * log_m
#define LEO_MULADD_256(x_lo, x_hi, y_lo, y_hi, table) { \
            LEO_M256 prod_lo, prod_hi; \
            LEO_MUL_256(y_lo, y_hi, table); \
            x_lo = _mm256_xor_si256(x_lo, prod_lo); \
            x_hi = _mm256_xor_si256(x_hi, prod_hi); }
*/

// Stores the partial products of x * y at offset x + y * 65536
// Repeated accesses from the same y value are faster
type product16Table struct {
	lut [4 * 16]ffe_t
}

var multiply16LUT product16Table

func initializeMultiplyTables() {
	// For each value we could multiply by:
	for log_m := uint32(0); log_m < kOrder; log_m++ {
		// For each 4 bits of the finite field width in bits:
		shift := 0
		for i := uint32(0); i < 4; i++ {
			// Construct 16 entry LUT for PSHUFB
			var prod_lo [16]byte
			var prod_hi [16]byte
			for x := ffe_t(0); x < 16; x++ {
				prod := multiplyLog(x<<shift, ffe_t(log_m))
				prod_lo[x] = byte(prod)
				prod_hi[x] = byte(prod >> 8)
			}

			// Store in 256-bit wide table

			// Copy 16 bytes to low and high 32 bytes
			copy(multiply256LUT[log_m].lo[i*16+0:i*16+16], prod_lo[:])
			copy(multiply256LUT[log_m].lo[i*16+16:i*16+32], prod_lo[:])
			copy(multiply256LUT[log_m].hi[i*16+0:i*16+16], prod_hi[:])
			copy(multiply256LUT[log_m].hi[i*16+16:i*16+32], prod_hi[:])

			shift += 4
		}
	}
}

func mul_mem(
	x []byte,
	y []byte,
	log_m ffe_t,
	bytes uint64,
) {
}

/*
static void mul_mem(
    void * LEO_RESTRICT x, const void * LEO_RESTRICT y,
    ffe_t log_m, uint64_t bytes)
{
        LEO_MUL_TABLES_256(0, log_m);

        const LEO_M256 clr_mask = _mm256_set1_epi8(0x0f);

        LEO_M256 * LEO_RESTRICT x32 = reinterpret_cast<LEO_M256 *>(x);
        const LEO_M256 * LEO_RESTRICT y32 = reinterpret_cast<const LEO_M256 *>(y);

        do
        {
#define LEO_MUL_256_LS(x_ptr, y_ptr) { \
            const LEO_M256 data_lo = _mm256_loadu_si256(y_ptr); \
            const LEO_M256 data_hi = _mm256_loadu_si256(y_ptr + 1); \
            LEO_M256 prod_lo, prod_hi; \
            LEO_MUL_256(data_lo, data_hi, 0); \
            _mm256_storeu_si256(x_ptr, prod_lo); \
            _mm256_storeu_si256(x_ptr + 1, prod_hi); }

            LEO_MUL_256_LS(x32, y32);
            y32 += 2, x32 += 2;

            bytes -= 64;
        } while (bytes > 0);

        return;
}
*/

//------------------------------------------------------------------------------
// FFT

// Twisted factors used in FFT
var fftSkew [kOrder]ffe_t

// Factors used in the evaluation of the error locator polynomial
var logWalsh [kOrder]ffe_t

func fftInitialize() {
	var temp [kBits - 1]ffe_t

	// Generate FFT skew vector {1}:

	for i := uint32(1); i < kBits; i++ {
		temp[i-1] = ffe_t(1 << i)
	}

	for m := uint32(0); m < (kBits - 1); m++ {
		step := uint32(1) << (m + 1)

		fftSkew[(1 << m)] = 0

		for i := m; i < (kBits - 1); i++ {
			s := uint32(1 << (i + 1))

			for j := uint32(1<<m) - 1; j < s; j += step {
				fftSkew[j+s+1] = fftSkew[j+1] ^ temp[i]
			}
		}

		temp[m] = kModulus - logLUT[multiplyLog(temp[m], logLUT[temp[m]^1])]

		for i := m + 1; i < (kBits - 1); i++ {
			sum := addMod(logLUT[temp[i]^1], temp[m])
			temp[i] = multiplyLog(temp[i], sum)
		}
	}

	for i := uint32(0); i < uint32(kModulus); i++ {
		fftSkew[i+1] = logLUT[fftSkew[i+1]]
	}

	// Precalculate FWHT(Log[i]):

	for i := uint32(0); i < kOrder; i++ {
		logWalsh[i] = logLUT[i]
	}
	logWalsh[0] = 0

	fwht(logWalsh[:], kOrder, kOrder)
}

// Decimation in time IFFT:
// The decimation in time IFFT algorithm allows us to unroll 2 layers at a time,
// performing calculations on local registers and faster cache memory.
// Each ^___^ below indicates a butterfly between the associated indices.
// The ifft_butterfly(x, y) operation:
//     y[] ^= x[]
//     if (log_m != kModulus)
//         x[] ^= exp(log(y[]) + log_m)
// Layer 0:
//     0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
//     ^_^ ^_^ ^_^ ^_^ ^_^ ^_^ ^_^ ^_^
// Layer 1:
//     0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
//     ^___^   ^___^   ^___^   ^___^
//       ^___^   ^___^   ^___^   ^___^

// Layer 2:
//     0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
//     ^_______^       ^_______^
//       ^_______^       ^_______^
//         ^_______^       ^_______^
//           ^_______^       ^_______^
// Layer 3:
//     0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
//     ^_______________^
//       ^_______________^
//         ^_______________^
//           ^_______________^
//             ^_______________^
//               ^_______________^
//                 ^_______________^
//                   ^_______________^
// DIT layer 0-1 operations, grouped 4 at a time:
//     {0-1, 2-3, 0-2, 1-3},
//     {4-5, 6-7, 4-6, 5-7},
// DIT layer 1-2 operations, grouped 4 at a time:
//     {0-2, 4-6, 0-4, 2-6},
//     {1-3, 5-7, 1-5, 3-7},
// DIT layer 2-3 operations, grouped 4 at a time:
//     {0-4, 0'-4', 0-0', 4-4'},
//     {1-5, 1'-5', 1-1', 5-5'},

func ifft_DIT2(
	x []byte,
	y []byte,
	log_m ffe_t,
	bytes uint64,
) {
}

/*
// 2-way butterfly
static void IFFT_DIT2(
    void * LEO_RESTRICT x, void * LEO_RESTRICT y,
    ffe_t log_m, uint64_t bytes)
{
        LEO_MUL_TABLES_256(0, log_m);

        const LEO_M256 clr_mask = _mm256_set1_epi8(0x0f);

        LEO_M256 * LEO_RESTRICT x32 = reinterpret_cast<LEO_M256 *>(x);
        LEO_M256 * LEO_RESTRICT y32 = reinterpret_cast<LEO_M256 *>(y);

        do
        {
#define LEO_IFFTB_256(x_ptr, y_ptr) { \
            LEO_M256 x_lo = _mm256_loadu_si256(x_ptr); \
            LEO_M256 x_hi = _mm256_loadu_si256(x_ptr + 1); \
            LEO_M256 y_lo = _mm256_loadu_si256(y_ptr); \
            LEO_M256 y_hi = _mm256_loadu_si256(y_ptr + 1); \
            y_lo = _mm256_xor_si256(y_lo, x_lo); \
            y_hi = _mm256_xor_si256(y_hi, x_hi); \
            _mm256_storeu_si256(y_ptr, y_lo); \
            _mm256_storeu_si256(y_ptr + 1, y_hi); \
            LEO_MULADD_256(x_lo, x_hi, y_lo, y_hi, 0); \
            _mm256_storeu_si256(x_ptr, x_lo); \
            _mm256_storeu_si256(x_ptr + 1, x_hi); }

            LEO_IFFTB_256(x32, y32);
            y32 += 2, x32 += 2;

            bytes -= 64;
        } while (bytes > 0);

        return;
}
*/

func ifft_DIT4(
	bytes uint64,
	work [][]byte,
	dist uint32,
	log_m01 ffe_t,
	log_m23 ffe_t,
	log_m02 ffe_t,
) {
}

/*
// 4-way butterfly
static void IFFT_DIT4(
    uint64_t bytes,
    void** work,
    unsigned dist,
    const ffe_t log_m01,
    const ffe_t log_m23,
    const ffe_t log_m02)
{
        LEO_MUL_TABLES_256(01, log_m01);
        LEO_MUL_TABLES_256(23, log_m23);
        LEO_MUL_TABLES_256(02, log_m02);

        const LEO_M256 clr_mask = _mm256_set1_epi8(0x0f);

        LEO_M256 * LEO_RESTRICT work0 = reinterpret_cast<LEO_M256 *>(work[0]);
        LEO_M256 * LEO_RESTRICT work1 = reinterpret_cast<LEO_M256 *>(work[dist]);
        LEO_M256 * LEO_RESTRICT work2 = reinterpret_cast<LEO_M256 *>(work[dist * 2]);
        LEO_M256 * LEO_RESTRICT work3 = reinterpret_cast<LEO_M256 *>(work[dist * 3]);

        do
        {
            LEO_M256 work_reg_lo_0 = _mm256_loadu_si256(work0);
            LEO_M256 work_reg_hi_0 = _mm256_loadu_si256(work0 + 1);
            LEO_M256 work_reg_lo_1 = _mm256_loadu_si256(work1);
            LEO_M256 work_reg_hi_1 = _mm256_loadu_si256(work1 + 1);

            // First layer:
            work_reg_lo_1 = _mm256_xor_si256(work_reg_lo_0, work_reg_lo_1);
            work_reg_hi_1 = _mm256_xor_si256(work_reg_hi_0, work_reg_hi_1);
            if (log_m01 != kModulus)
                LEO_MULADD_256(work_reg_lo_0, work_reg_hi_0, work_reg_lo_1, work_reg_hi_1, 01);

            LEO_M256 work_reg_lo_2 = _mm256_loadu_si256(work2);
            LEO_M256 work_reg_hi_2 = _mm256_loadu_si256(work2 + 1);
            LEO_M256 work_reg_lo_3 = _mm256_loadu_si256(work3);
            LEO_M256 work_reg_hi_3 = _mm256_loadu_si256(work3 + 1);

            work_reg_lo_3 = _mm256_xor_si256(work_reg_lo_2, work_reg_lo_3);
            work_reg_hi_3 = _mm256_xor_si256(work_reg_hi_2, work_reg_hi_3);
            if (log_m23 != kModulus)
                LEO_MULADD_256(work_reg_lo_2, work_reg_hi_2, work_reg_lo_3, work_reg_hi_3, 23);

            // Second layer:
            work_reg_lo_2 = _mm256_xor_si256(work_reg_lo_0, work_reg_lo_2);
            work_reg_hi_2 = _mm256_xor_si256(work_reg_hi_0, work_reg_hi_2);
            work_reg_lo_3 = _mm256_xor_si256(work_reg_lo_1, work_reg_lo_3);
            work_reg_hi_3 = _mm256_xor_si256(work_reg_hi_1, work_reg_hi_3);
            if (log_m02 != kModulus)
            {
                LEO_MULADD_256(work_reg_lo_0, work_reg_hi_0, work_reg_lo_2, work_reg_hi_2, 02);
                LEO_MULADD_256(work_reg_lo_1, work_reg_hi_1, work_reg_lo_3, work_reg_hi_3, 02);
            }

            _mm256_storeu_si256(work0, work_reg_lo_0);
            _mm256_storeu_si256(work0 + 1, work_reg_hi_0);
            _mm256_storeu_si256(work1, work_reg_lo_1);
            _mm256_storeu_si256(work1 + 1, work_reg_hi_1);
            _mm256_storeu_si256(work2, work_reg_lo_2);
            _mm256_storeu_si256(work2 + 1, work_reg_hi_2);
            _mm256_storeu_si256(work3, work_reg_lo_3);
            _mm256_storeu_si256(work3 + 1, work_reg_hi_3);

            work0 += 2, work1 += 2, work2 += 2, work3 += 2;

            bytes -= 64;
        } while (bytes > 0);

        return;
}
*/

// Unrolled IFFT for encoder
func ifft_DIT_Encoder(
	bytes uint64,
	data [][]byte,
	work [][]byte,
	skewLUT []ffe_t,
) {
	m := uint32(len(data))

	// I tried rolling the memcpy/memset into the first layer of the FFT and
	// found that it only yields a 4% performance improvement, which is not
	// worth the extra complexity.
	for i := uint32(0); i < m; i++ {
		copy(work[i], data[i])
	}
	// I tried splitting up the first few layers into L3-cache sized blocks but
	// found that it only provides about 5% performance boost, which is not
	// worth the extra complexity.

	// Decimation in time: Unroll 2 layers at a time
	dist := uint32(1)
	dist4 := uint32(4)
	for ; dist4 <= m; dist4 <<= 2 {
		// For each set of dist*4 elements:
		for r := uint32(0); r < m; r += dist4 {
			i_end := r + dist
			log_m01 := skewLUT[i_end]
			log_m02 := skewLUT[i_end+dist]
			log_m23 := skewLUT[i_end+dist*2]

			// For each set of dist elements:
			for i := r; i < uint32(i_end); i++ {
				ifft_DIT4(
					bytes,
					work[i:],
					dist,
					log_m01,
					log_m23,
					log_m02,
				)
			}
		}

		// I tried alternating sweeps left->right and right->left to reduce cache misses.
		// It provides about 1% performance boost when done for both FFT and IFFT, so it
		// does not seem to be worth the extra complexity.

		dist = dist4
	}

	// If there is one layer left:
	if dist < m {
		// Assuming that dist = m / 2

		log_m := skewLUT[dist]

		if log_m == kModulus {
			vectorXOR_Threads(bytes, dist, work[dist:], work)
		} else {
			for i := uint32(0); i < dist; i++ {
				ifft_DIT2(
					work[i],
					work[i+dist],
					log_m,
					bytes,
				)
			}
		}
	}
}

// Basic no-frills version for decoder
func ifft_DIT_Decoder(
	bytes uint64,
	m uint32,
	work [][]byte,
	skewLUT []ffe_t,
) {
	// Decimation in time: Unroll 2 layers at a time
	dist := uint32(1)
	dist4 := uint32(4)
	for ; dist4 <= m; dist4 <<= 2 {
		// For each set of dist*4 elements:
		for r := uint32(0); r < m; r += dist4 {
			i_end := r + dist
			log_m01 := skewLUT[i_end]
			log_m02 := skewLUT[i_end+dist]
			log_m23 := skewLUT[i_end+dist*2]

			// For each set of dist elements:
			for i := r; i < i_end; i++ {
				ifft_DIT4(
					bytes,
					work[i:],
					dist,
					log_m01,
					log_m23,
					log_m02,
				)
			}
		}

		dist = dist4
	}

	// If there is one layer left:
	if dist < m {
		// Assuming that dist = m / 2

		log_m := skewLUT[dist]

		if log_m == kModulus {
			vectorXOR_Threads(bytes, dist, work[dist:], work)
		} else {
			for i := uint32(0); i < dist; i++ {
				ifft_DIT2(
					work[i],
					work[i+dist],
					log_m,
					bytes,
				)
			}
		}
	}
}

// Decimation in time FFT:
// The decimation in time FFT algorithm allows us to unroll 2 layers at a time,
// performing calculations on local registers and faster cache memory.
// Each ^___^ below indicates a butterfly between the associated indices.
// The fft_butterfly(x, y) operation:
//     if (log_m != kModulus)
//         x[] ^= exp(log(y[]) + log_m)
//     y[] ^= x[]
// Layer 0:
//     0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
//     ^_______________^
//       ^_______________^
//         ^_______________^
//           ^_______________^
//             ^_______________^
//               ^_______________^
//                 ^_______________^
//                   ^_______________^
// Layer 1:
//     0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
//     ^_______^       ^_______^
//       ^_______^       ^_______^
//         ^_______^       ^_______^
//           ^_______^       ^_______^

// Layer 2:
//     0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
//     ^___^   ^___^   ^___^   ^___^
//       ^___^   ^___^   ^___^   ^___^
// Layer 3:
//     0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
//     ^_^ ^_^ ^_^ ^_^ ^_^ ^_^ ^_^ ^_^
// DIT layer 0-1 operations, grouped 4 at a time:
//     {0-0', 4-4', 0-4, 0'-4'},
//     {1-1', 5-5', 1-5, 1'-5'},
// DIT layer 1-2 operations, grouped 4 at a time:
//     {0-4, 2-6, 0-2, 4-6},
//     {1-5, 3-7, 1-3, 5-7},
// DIT layer 2-3 operations, grouped 4 at a time:
//     {0-2, 1-3, 0-1, 2-3},
//     {4-6, 5-7, 4-5, 6-7},

func fft_DIT2(
	x []byte,
	y []byte,
	log_m ffe_t,
	bytes uint64,
) {
}

/*
// 2-way butterfly
static void FFT_DIT2(
    void * LEO_RESTRICT x, void * LEO_RESTRICT y,
    ffe_t log_m, uint64_t bytes)
{

        LEO_MUL_TABLES_256(0, log_m);

        const LEO_M256 clr_mask = _mm256_set1_epi8(0x0f);

        LEO_M256 * LEO_RESTRICT x32 = reinterpret_cast<LEO_M256 *>(x);
        LEO_M256 * LEO_RESTRICT y32 = reinterpret_cast<LEO_M256 *>(y);

        do
        {
#define LEO_FFTB_256(x_ptr, y_ptr) { \
            LEO_M256 x_lo = _mm256_loadu_si256(x_ptr); \
            LEO_M256 x_hi = _mm256_loadu_si256(x_ptr + 1); \
            LEO_M256 y_lo = _mm256_loadu_si256(y_ptr); \
            LEO_M256 y_hi = _mm256_loadu_si256(y_ptr + 1); \
            LEO_MULADD_256(x_lo, x_hi, y_lo, y_hi, 0); \
            _mm256_storeu_si256(x_ptr, x_lo); \
            _mm256_storeu_si256(x_ptr + 1, x_hi); \
            y_lo = _mm256_xor_si256(y_lo, x_lo); \
            y_hi = _mm256_xor_si256(y_hi, x_hi); \
            _mm256_storeu_si256(y_ptr, y_lo); \
            _mm256_storeu_si256(y_ptr + 1, y_hi); }

            LEO_FFTB_256(x32, y32);
            y32 += 2, x32 += 2;

            bytes -= 64;
        } while (bytes > 0);

        return;
}
*/

func fft_DIT4(
	bytes uint64,
	work [][]byte,
	dist uint32,
	log_m01 ffe_t,
	log_m23 ffe_t,
	log_m02 ffe_t,
) {
}

/*
// 4-way butterfly
static void FFT_DIT4(
    uint64_t bytes,
    void** work,
    unsigned dist,
    const ffe_t log_m01,
    const ffe_t log_m23,
    const ffe_t log_m02)
{

        LEO_MUL_TABLES_256(01, log_m01);
        LEO_MUL_TABLES_256(23, log_m23);
        LEO_MUL_TABLES_256(02, log_m02);

        const LEO_M256 clr_mask = _mm256_set1_epi8(0x0f);

        LEO_M256 * LEO_RESTRICT work0 = reinterpret_cast<LEO_M256 *>(work[0]);
        LEO_M256 * LEO_RESTRICT work1 = reinterpret_cast<LEO_M256 *>(work[dist]);
        LEO_M256 * LEO_RESTRICT work2 = reinterpret_cast<LEO_M256 *>(work[dist * 2]);
        LEO_M256 * LEO_RESTRICT work3 = reinterpret_cast<LEO_M256 *>(work[dist * 3]);

        do
        {
            LEO_M256 work_reg_lo_0 = _mm256_loadu_si256(work0);
            LEO_M256 work_reg_hi_0 = _mm256_loadu_si256(work0 + 1);
            LEO_M256 work_reg_lo_1 = _mm256_loadu_si256(work1);
            LEO_M256 work_reg_hi_1 = _mm256_loadu_si256(work1 + 1);
            LEO_M256 work_reg_lo_2 = _mm256_loadu_si256(work2);
            LEO_M256 work_reg_hi_2 = _mm256_loadu_si256(work2 + 1);
            LEO_M256 work_reg_lo_3 = _mm256_loadu_si256(work3);
            LEO_M256 work_reg_hi_3 = _mm256_loadu_si256(work3 + 1);

            // First layer:
            if (log_m02 != kModulus)
            {
                LEO_MULADD_256(work_reg_lo_0, work_reg_hi_0, work_reg_lo_2, work_reg_hi_2, 02);
                LEO_MULADD_256(work_reg_lo_1, work_reg_hi_1, work_reg_lo_3, work_reg_hi_3, 02);
            }
            work_reg_lo_2 = _mm256_xor_si256(work_reg_lo_0, work_reg_lo_2);
            work_reg_hi_2 = _mm256_xor_si256(work_reg_hi_0, work_reg_hi_2);
            work_reg_lo_3 = _mm256_xor_si256(work_reg_lo_1, work_reg_lo_3);
            work_reg_hi_3 = _mm256_xor_si256(work_reg_hi_1, work_reg_hi_3);

            // Second layer:
            if (log_m01 != kModulus)
                LEO_MULADD_256(work_reg_lo_0, work_reg_hi_0, work_reg_lo_1, work_reg_hi_1, 01);
            work_reg_lo_1 = _mm256_xor_si256(work_reg_lo_0, work_reg_lo_1);
            work_reg_hi_1 = _mm256_xor_si256(work_reg_hi_0, work_reg_hi_1);

            _mm256_storeu_si256(work0, work_reg_lo_0);
            _mm256_storeu_si256(work0 + 1, work_reg_hi_0);
            _mm256_storeu_si256(work1, work_reg_lo_1);
            _mm256_storeu_si256(work1 + 1, work_reg_hi_1);

            if (log_m23 != kModulus)
                LEO_MULADD_256(work_reg_lo_2, work_reg_hi_2, work_reg_lo_3, work_reg_hi_3, 23);
            work_reg_lo_3 = _mm256_xor_si256(work_reg_lo_2, work_reg_lo_3);
            work_reg_hi_3 = _mm256_xor_si256(work_reg_hi_2, work_reg_hi_3);

            _mm256_storeu_si256(work2, work_reg_lo_2);
            _mm256_storeu_si256(work2 + 1, work_reg_hi_2);
            _mm256_storeu_si256(work3, work_reg_lo_3);
            _mm256_storeu_si256(work3 + 1, work_reg_hi_3);

            work0 += 2, work1 += 2, work2 += 2, work3 += 2;

            bytes -= 64;
        } while (bytes > 0);

        return;
}
*/

// In-place FFT for encoder and decoder
func fft_DIT(
	bytes uint64,
	work [][]byte,
	skewLUT []ffe_t,
) {
	m := uint32(len(work)) / 2

	// Decimation in time: Unroll 2 layers at a time
	dist4 := m
	dist := m >> 2
	for ; dist != 0; dist >>= 2 {
		// For each set of dist*4 elements:
		for r := uint32(0); r < m; r += dist4 {
			i_end := r + dist
			log_m01 := skewLUT[i_end]
			log_m02 := skewLUT[i_end+dist]
			log_m23 := skewLUT[i_end+dist*2]

			// For each set of dist elements:
			for i := r; i < i_end; i++ {
				fft_DIT4(
					bytes,
					work[i:],
					dist,
					log_m01,
					log_m23,
					log_m02,
				)
			}
		}

		dist4 = dist
	}

	// If there is one layer left:
	if dist4 == 2 {
		for r := uint32(0); r < m; r += 2 {
			log_m := skewLUT[r+1]

			if log_m == kModulus {
				xor_mem(work[r+1], work[r], bytes)
			} else {
				fft_DIT2(
					work[r],
					work[r+1],
					log_m,
					bytes,
				)
			}
		}
	}
}

//------------------------------------------------------------------------------
// Reed-Solomon Encode

func reedSolomonEncode(
	bufferBytes uint64,
	data [][]byte,
	work [][]byte,
) {
	originalCount := uint32(len(data))

	// work <- IFFT(data, m, m)

	skewLUT := fftSkew[originalCount:]

	ifft_DIT_Encoder(
		bufferBytes,
		data,
		work,
		skewLUT,
	)

	// work <- FFT(work, m, 0)
	fft_DIT(
		bufferBytes,
		work,
		fftSkew[:],
	)
}

//------------------------------------------------------------------------------
// Reed-Solomon Decode

func reedSolomonDecode(
	bufferBytes uint64,
	original [][]byte, // original_count entries
	recovery [][]byte, // recovery_count entries
	work [][]byte, // n entries
) {
	originalCount := uint32(len(original))
	recoveryCount := uint32(len(recovery))
	m := recoveryCount
	n := uint32(len(work))

	zeroBytes := bytes.Repeat([]byte{0}, int(bufferBytes))

	// Fill in error locations

	var errorLocations [kOrder]ffe_t
	for i := uint32(0); i < recoveryCount; i++ {
		if recovery[i] == nil {
			errorLocations[i] = 1
		}
	}
	for i := recoveryCount; i < m; i++ {
		errorLocations[i] = 1
	}
	for i := uint32(0); i < originalCount; i++ {
		if original[i] == nil {
			errorLocations[i+m] = 1
		}
	}

	// Evaluate error locator polynomial

	fwht(errorLocations[:], kOrder, m+originalCount)

	for i := uint32(0); i < kOrder; i++ {
		errorLocations[i] = uint16((uint32(errorLocations[i]) * uint32(logWalsh[i])) % uint32(kModulus))
	}

	fwht(errorLocations[:], kOrder, kOrder)

	// work <- recovery data

	for i := uint32(0); i < recoveryCount; i++ {
		if recovery[i] != nil {
			mul_mem(work[i], recovery[i], errorLocations[i], bufferBytes)
		} else {
			copy(work[i], zeroBytes)
		}
	}
	for i := recoveryCount; i < m; i++ {
		copy(work[i], zeroBytes)
	}

	// work <- original data

	for i := uint32(0); i < originalCount; i++ {
		if original[i] != nil {
			mul_mem(work[m+i], original[i], errorLocations[m+i], bufferBytes)
		} else {
			copy(work[m+i], zeroBytes)
		}
	}
	for i := m + originalCount; i < n; i++ {
		copy(work[i], zeroBytes)
	}

	// work <- IFFT(work, n, 0)

	ifft_DIT_Decoder(
		bufferBytes,
		m+originalCount,
		work,
		fftSkew[:],
	)

	// work <- FormalDerivative(work, n)

	for i := uint32(1); i < n; i++ {
		width := ((i ^ (i - 1)) + 1) >> 1

		if width < 8 {
			vectorXOR(
				bufferBytes,
				width,
				work[i-width:],
				work[i:],
			)
		} else {
			vectorXOR_Threads(
				bufferBytes,
				width,
				work[i-width:],
				work[i:],
			)
		}
	}

	// work <- FFT(work, n, 0) truncated to m + original_count

	fft_DIT(
		bufferBytes,
		work,
		fftSkew[:],
	)

	// Reveal erasures

	for i := uint32(0); i < originalCount; i++ {
		if original[i] == nil {
			mul_mem(work[i], work[i+m], kModulus-errorLocations[i+m], bufferBytes)
		}
	}
}

//------------------------------------------------------------------------------
// API

func initializeFF16() {
	if isInitialized {
		return
	}

	initializeLogarithmTables()
	initializeMultiplyTables()
	fftInitialize()
}
