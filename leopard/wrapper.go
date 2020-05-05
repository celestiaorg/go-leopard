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

func EncodeWorkCount(origCount, recoveryCount uint32) uint32 {
	return leoEncodeWorkCount(origCount, recoveryCount)
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

// wrapper around C.free (can also be used in tests)
func freeAndNilBuf(p unsafe.Pointer) {
	// if p represents NULL this is a nop:
	C.free(p)
	p = nil
}
