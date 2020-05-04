package leopard

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

	ErrCallInitialize = errors.New("call leo_init() first")
)

func leopardResultToErr(errCode int32) error {
	switch errCode {
	case 0:
		return nil
	case -1:
		return ErrNeedMoreData
	case -2:
		return ErrTooMuchData
	case -3:
		return ErrInvalidSize
	case -4:
		return ErrInvalidCounts
	case -5:
		return ErrInvalidInput
	case -6:
		return ErrPlatform
	case -7:
		return ErrCallInitialize
	default:
		panic("Yelp")
	}
}

func Init() error {
	return leopardResultToErr(leoInit(2))
}

// TODO We probably do not need to export this?
func EncodeWorkCount(origCount, recoveryCount uint32) uint32 {
	return leoEncodeWorkCount(origCount, recoveryCount)
}

func convert(data [][]byte) []unsafe.Pointer {
	res := make([]unsafe.Pointer, len(data))
	for i, d := range data {
		p := C.malloc(C.size_t(len(d)))
		cBuf := (*[1 << 30]byte)(p)
		copy(cBuf[:], d)
		res[i] = unsafe.Pointer(cBuf)
	}
	return res
}
