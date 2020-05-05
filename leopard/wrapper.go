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
	err = leopardResultToErr(leoEncode(bufferBytes, origCount, recoveryCount, workCount, origDataPtrs, encodeWorkPtr))
	if err != nil {
		return nil, err
	}
	toGoByte(encodeWorkPtr, encodeWork, int(bufferBytes))
	return encodeWork, nil
}

func Decode(data [][]byte) ([][]byte, error) {
	panic("implement")
}

func extract(data [][]byte) (origCount uint32, bufferBytes uint64, err error) {
	origCount = uint32(len(data))
	if origCount == 0 {
		err = errors.New("zero length data")
		return
	}
	bufferBytes = uint64(len(data[0]))
	// TODO can we do without verifying that all buffers have the same size?
	for _, d := range data {
		if uint64(len(d)) != bufferBytes {
			err = errors.New("each buffer should have the same size")
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
		res[i] = (*[1 << 30]byte)(p)[0:bufferBytes]
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
