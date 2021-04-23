package leopard

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
