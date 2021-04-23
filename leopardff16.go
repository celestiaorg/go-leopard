package leopard

//------------------------------------------------------------------------------
// Datatypes and Constants

// Finite field element type
// TODO use this
type ffe_t = uint16

// Number of bits per element
const kBits = 16

// Finite field order: Number of elements in the field
const kOrder = 65536

// Modulus for field operations
const kModulus = 65535

// LFSR Polynomial that generates the field elements
const kPolynomial = 0x1002D
