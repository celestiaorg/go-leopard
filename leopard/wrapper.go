package leopard

//#include <stdlib.h>
import "C"

import (
	"errors"
	"unsafe"
)

var (
	ErrNeedMoreData  = errors.New("not enough recovery data received")
	ErrTooMuchData   = errors.New("buffer counts are too high")
	ErrInvalidSize   = errors.New("buffer size must be a multiple of 64 bytes")
	ErrInvalidCounts = errors.New("invalid counts provided")
	ErrInvalidInput  = errors.New("a function parameter was invalid")
	ErrPlatform      = errors.New("platform is unsupported")

	ErrCallInitialize = errors.New("call leopard.Init() first")
)

const version = 2

func leopardResultToErr(errCode leopardresult) error {
	switch errCode {
	case leopardSuccess:
		return nil
	case leopardNeedmoredata:
		return ErrNeedMoreData
	case leopardToomuchdata:
		return ErrTooMuchData
	case leopardInvalidsize:
		return ErrInvalidSize
	case leopardInvalidcounts:
		return ErrInvalidCounts
	case leopardInvalidinput:
		return ErrInvalidInput
	case leopardPlatform:
		return ErrPlatform
	case leopardCallinitialize:
		return ErrCallInitialize
	default:
		panic("unexpected leopard return code")
	}
}

func Init() error {
	return leopardResultToErr(leopardresult(leoInit(version)))
}

// Encode takes an slice of equally sized byte slices and computes len(data) parity shares.
func Encode(data [][]byte) (encodeWork [][]byte, err error) {
	origCount, bufferBytes, err := extract(data)
	if err != nil {
		return nil, err
	}
	recoveryCount := origCount / 2
	workCount := leoEncodeWorkCount(origCount, recoveryCount)
	origDataPtrs := copyToCmallocedPtrs(data)
	defer free(origDataPtrs)

	encodeWork = make([][]byte, workCount)
	for i := uint(0); i < uint(workCount); i++ {
		encodeWork[i] = make([]byte, bufferBytes)
	}
	encodeWorkPtr := copyToCmallocedPtrs(encodeWork)
	defer free(encodeWorkPtr)

	err = leopardResultToErr(leoEncode(
		bufferBytes,
		origCount,
		recoveryCount,
		workCount,
		origDataPtrs,
		encodeWorkPtr))
	if err != nil {
		return
	}
	toGoByte(encodeWorkPtr, encodeWork, int(bufferBytes))
	return encodeWork, nil
}

// Decode takes in what is left from the original data and the extended recovery data
// and recovers missing data (upto half of the total data can be missing).
// Missing data (either original or recovery) has to be nil.
func Decode(orig, recovery [][]byte) (decodeWork [][]byte, err error) {
	_, bufferBytes, err := extract(recovery)
	if err != nil {
		return
	}
	origCount := uint32(len(orig))
	recoveryCount := origCount / 2
	decodeWorkCount := leoDecodeWorkCount(origCount, recoveryCount)

	decodeWork = make([][]byte, decodeWorkCount)
	for i := uint(0); i < uint(decodeWorkCount); i++ {
		decodeWork[i] = make([]byte, bufferBytes)
	}
	decodeWorkPtr := copyToCmallocedPtrs(decodeWork)
	defer free(decodeWorkPtr)
	origDataPtr := copyToCmallocedPtrs(orig)
	defer free(origDataPtr)

	recoveryDataPtr := copyToCmallocedPtrs(recovery)
	defer free(recoveryDataPtr)

	err = leopardResultToErr(leoDecode(
		bufferBytes,
		origCount,
		recoveryCount,
		decodeWorkCount,
		origDataPtr,
		recoveryDataPtr,
		decodeWorkPtr))
	if err != nil {
		return
	}

	toGoByte(decodeWorkPtr, decodeWork, int(bufferBytes))

	return
}

func extract(data [][]byte) (origCount uint32, bufferBytes uint64, err error) {
	origCount = uint32(len(data))
	if origCount == 0 {
		err = errors.New("zero length data")
		return
	}
	bufferBytes = uint64(len(data[len(data)-1]))
	// TODO can we do without verifying that all buffers have the same size?
	for _, d := range data {
		if d != nil && uint64(len(d)) != bufferBytes {
			err = errors.New("each buffer should have the same size or can be nil")
			return
		}
	}
	return
}

// copy over to C allocated memory:
func copyToCmallocedPtrs(data [][]byte) []unsafe.Pointer {
	res := make([]unsafe.Pointer, len(data))
	for i, d := range data {
		if len(d) > 0 {
			cBuf := C.CBytes(d)
			res[i] = unsafe.Pointer(cBuf)
		} else {
			// keep this nil as leopard uses this internally
			res[i] = nil
		}
	}
	return res
}

func toGoByte(ps []unsafe.Pointer, res [][]byte, bufferBytes int) {
	if len(ps) != len(res) {
		panic("can't convert back to go slice")
	}
	for i, p := range ps {
		res[i] = make([]byte, bufferBytes)
		b := res[i]
		for j := 0; j < bufferBytes; j++ {
			// creates a copy of the data
			b[j] = *(*byte)(unsafe.Pointer(uintptr(p) + uintptr(j)))
		}
	}
}

// wrapper around C.free (can also be used in tests)
func freeAndNilBuf(p unsafe.Pointer) {
	// if p represents NULL this is a nop:
	C.free(p)
	p = nil
}

func free(ps []unsafe.Pointer) {
	for _, p := range ps {
		freeAndNilBuf(p)
	}
}
