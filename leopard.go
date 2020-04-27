package leopard

/*
#cgo LDFLAGS: -L${SRCDIR}/libleopard/build -llibleopard -lstdc++

#include "./libleopard/leopard.h"

#define LEO_VERSION 2
*/
import "C"

import (
	"errors"
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

//func Encode(bufferBytes uint64, originalData [][]byte, originalCount, recoveryCount uint) ([][]byte, error) {
//	encodeWorkCount := C.leo_encode_work_count(C.uint(originalCount), C.uint(recoveryCount))
//	var encodeWork = make([][]byte, encodeWorkCount)
//	//encodeWork := CMatrix(int(encodeWorkCount), int(bufferBytes))
//	//cencodeWork := unsafe.Pointer(CMatrixPtr(encodeWork))
//	for i := uint(0); i < uint(encodeWorkCount); i++ {
//		encodeWork[i] = make([]byte, bufferBytes)
//	}
//	coriginalData, allocs := unpackArgSSByte(originalData)
//	//coriginalData :=  (unsafe.Pointer)(&originalData[0][0])
//	//cencodeWork :=  (unsafe.Pointer)(&encodeWork[0][0])
//
//	//origDataPtr := make([]*byte, len(originalData))
//	//for i := range originalData {
//	//	origDataPtr[i] = &originalData[i][0]
//	//}
//	//corigData := (*unsafe.Pointer)(unsafe.Pointer(origDataPtr[0]))
//
//	encodeWorkPtr := make([]*byte, len(encodeWork))
//	for i := range encodeWork {
//		encodeWorkPtr[i] = &encodeWork[i][0]
//	}
//	cencode := (*unsafe.Pointer)(unsafe.Pointer(encodeWorkPtr[0]))
//
//	err := leopardResultToErr(
//		int32(
//			C.leo_encode(C.uint64_t(bufferBytes),
//				C.unsigned(originalCount),
//				C.unsigned(recoveryCount),
//				C.unsigned(encodeWorkCount),
//				(*unsafe.Pointer)(*coriginalData),
//				cencode,
//			),
//		))
//	if err != nil {
//		return nil, err
//	}
//	fmt.Println(*allocs)
//	return encodeWork, nil
//}

func LeoEncode2(bufferBytes uint64, originalCount uint32, recoveryCount uint32, workCount uint32, originalData [][]byte, workData [][]byte) int {
	cbufferBytes, _ := (C.uint64_t)(bufferBytes), cgoAllocsUnknown
	coriginalCount, _ := (C.uint)(originalCount), cgoAllocsUnknown
	crecoveryCount, _ := (C.uint)(recoveryCount), cgoAllocsUnknown
	cworkCount, _ := (C.uint)(workCount), cgoAllocsUnknown
	coriginalData, _ := unpackArgSSByte(originalData)
	cworkData, _ := unpackArgSSByte(workData)
	__ret := C.leo_encode2(cbufferBytes, coriginalCount, crecoveryCount, cworkCount, coriginalData, cworkData)
	packSSByte(workData, cworkData)
	packSSByte(originalData, coriginalData)
	__v := (int)(__ret)
	return __v
}

// LeoDecode2 function as declared in leopard/leopard.h:252
func LeoDecode2(bufferBytes uint64,
	originalCount uint32,
	recoveryCount uint32,
	workCount uint32,
	originalData [][]byte,
	recoveryData [][]byte,
	workData [][]byte) int {

	cbufferBytes, _ := (C.uint64_t)(bufferBytes), cgoAllocsUnknown
	coriginalCount, _ := (C.uint)(originalCount), cgoAllocsUnknown
	crecoveryCount, _ := (C.uint)(recoveryCount), cgoAllocsUnknown
	cworkCount, _ := (C.uint)(workCount), cgoAllocsUnknown
	coriginalData, _ := unpackArgSSByte(originalData)
	crecoveryData, _ := unpackArgSSByte(recoveryData)
	cworkData, _ := unpackArgSSByte(workData)
	__ret := C.leo_decode2(cbufferBytes, coriginalCount, crecoveryCount, cworkCount, coriginalData, crecoveryData, cworkData)
	packSSByte(workData, cworkData)
	packSSByte(recoveryData, crecoveryData)
	packSSByte(originalData, coriginalData)
	__v := (int)(__ret)

	return __v
}
