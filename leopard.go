package leopard

/*
#cgo LDFLAGS: -L${SRCDIR}/libleopard/build -llibleopard -lstdc++

#include "./libleopard/leopard.h"

#define LEO_VERSION 2
*/
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
	return leopardResultToErr(int32(C.leo_init_(C.LEO_VERSION)))
}

// TODO can't we simply use this internally?
func EncodeWorkCount(origCount, recoveryCount uint64) uint64 {
	return uint64(C.leo_encode_work_count(C.uint(origCount), C.uint(recoveryCount)))
}

func Encode(bufferBytes uint64, originalData [][]byte, originalCount, recoveryCount uint) ([][]byte, error) {
	encodeWorkCount := C.leo_encode_work_count(C.uint(originalCount), C.uint(recoveryCount))
	var (
		encodeWork       = make([][]byte, encodeWorkCount)
		origninalDataPtr = (*unsafe.Pointer)(unsafe.Pointer(&originalData[0]))
		encodeWorkData   = (*unsafe.Pointer)(unsafe.Pointer(&encodeWork[0]))
	)
	err := leopardResultToErr(
		int32(
			C.leo_encode(C.uint64_t(bufferBytes),
				C.unsigned(originalCount),
				C.unsigned(recoveryCount),
				C.unsigned(encodeWorkCount),
				origninalDataPtr,
				encodeWorkData,
			),
		))
	if err != nil {
		return nil, err
	}
	return encodeWork, nil
}
