package leopard

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/cpu"
)

func init() {
	_ = cpu.X86.HasAVX2
	_ = cpu.X86.HasSSE3
}

type LeopardResult int32

const (
	LeopardSuccess        LeopardResult = 0
	LeopardNeedmoredata   LeopardResult = -1
	LeopardToomuchdata    LeopardResult = -2
	LeopardInvalidsize    LeopardResult = -3
	LeopardInvalidcounts  LeopardResult = -4
	LeopardInvalidinput   LeopardResult = -5
	LeopardPlatform       LeopardResult = -6
	LeopardCallinitialize LeopardResult = -7
)

var (
	ErrNeedMoreData  = errors.New("not enough recovery data received")
	ErrTooMuchData   = errors.New("buffer counts are too high")
	ErrInvalidSize   = errors.New("buffer size must be a multiple of 64 bytes")
	ErrInvalidCounts = errors.New("invalid counts provided")
	ErrInvalidInput  = errors.New("a function parameter was invalid")
	ErrPlatform      = errors.New("platform is unsupported")

	ErrCallInitialize = errors.New("call Init() first")

	errAllBuffersEmpty = errors.New("all buffers are empty")
)

const version = 2

func init() {
	if err := Init(); err != nil {
		panic(fmt.Sprintf("Unexpected error while initializing leopard %v", err))
	}
}

func leopardResultToErr(errCode LeopardResult) error {
	switch errCode {
	case LeopardSuccess:
		return nil
	case LeopardNeedmoredata:
		return ErrNeedMoreData
	case LeopardToomuchdata:
		return ErrTooMuchData
	case LeopardInvalidsize:
		return ErrInvalidSize
	case LeopardInvalidcounts:
		return ErrInvalidCounts
	case LeopardInvalidinput:
		return ErrInvalidInput
	case LeopardPlatform:
		return ErrPlatform
	case LeopardCallinitialize:
		return ErrCallInitialize
	default:
		panic("unexpected Leopard return code")
	}
}

func Init() error {
	return leopardResultToErr(LeopardResult(LeoInit(version)))
}

// Encode takes an slice of equally sized byte slices and computes len(data) parity shares.
// This means you can lose half of (data || encodeWork) and still recover the data.
func Encode(data [][]byte) (encodeWork [][]byte, err error) {
	origCount, bufferBytes, err := extractCounts(data)
	if err != nil {
		return nil, err
	}
	recoveryCount := origCount
	workCount := LeoEncodeWorkCount(origCount, recoveryCount)
	origDataPtrs := copyToCmallocedPtrs(data)
	defer freeAll(origDataPtrs)

	encodeWork = make([][]byte, workCount)
	for i := uint(0); i < uint(workCount); i++ {
		encodeWork[i] = make([]byte, bufferBytes)
	}
	encodeWorkPtr := copyToCmallocedPtrs(encodeWork)
	defer freeAll(encodeWorkPtr)

	err = leopardResultToErr(LeoEncode(
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
	// XXX: We only return half the data here as the other half is all zeroes
	// and superfluous.
	// For details see: https://github.com/catid/leopard/issues/15#issuecomment-631391392
	return encodeWork[:origCount], nil
}

// Recover takes in what is left from the original data and the extended recovery data
// and recovers missing original data (upto half of the total data can be missing).
// Missing data (either original or recovery) has to be nil when passed in.
func Recover(orig, recovery [][]byte) (decodeWork [][]byte, err error) {
	if len(orig) != len(recovery) {
		err = fmt.Errorf(
			"recovery is only implemented for len(orig)==len(recovery), got: %v != %v",
			len(orig),
			len(recovery),
		)
		return
	}
	_, bufferBytesRecov, _ := extractCounts(recovery)
	_, bufferBytesOrig, _ := extractCounts(orig)
	bufferBytes := max(bufferBytesRecov, bufferBytesOrig)
	if bufferBytes == 0 {
		err = errAllBuffersEmpty
		return
	}
	origCount := uint32(len(orig))
	recoveryCount := origCount
	decodeWorkCount := LeoDecodeWorkCount(origCount, recoveryCount)

	decodeWork = make([][]byte, decodeWorkCount)
	for i := uint(0); i < uint(decodeWorkCount); i++ {
		decodeWork[i] = make([]byte, bufferBytes)
	}
	decodeWorkPtr := copyToCmallocedPtrs(decodeWork)
	defer freeAll(decodeWorkPtr)
	origDataPtr := copyToCmallocedPtrs(orig)
	defer freeAll(origDataPtr)

	recoveryDataPtr := copyToCmallocedPtrs(recovery)
	defer freeAll(recoveryDataPtr)

	err = leopardResultToErr(LeoDecode(
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
	// leopard only recovers missing original chunks:
	// decodeWork = decodeWork[:len(decodeWork)/2]
	// modified fork at:
	// https://github.com/Liamsi/leopard/tree/recover_parity_chunks_too
	return
}

// Decode takes in what is left from the original data and the extended recovery data
// and recovers missing original or missing recovery data
// (upto half of the total data can be missing).
// Note the only difference to Recover is that Decode also recovers missing parity shares.
// On success, it returns (orig || recovery) with all data recovered.
// Missing data (either original or recovery) has to be nil when passed in.
func Decode(orig, recovery [][]byte) (decoded [][]byte, err error) {
	decoded, err = Recover(orig, recovery)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(orig); i++ {
		if orig[i] != nil {
			decoded[i] = orig[i]
		}
	}
	for i := 0; i < len(recovery); i++ {
		if recovery[i] != nil {
			decoded[i+len(orig)] = recovery[i]
		}
	}

	return
}

func max(x uint64, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}

func extractCounts(data [][]byte) (dataLen uint32, bufferBytes uint64, err error) {
	dataLen = uint32(len(data))
	if dataLen == 0 {
		err = errors.New("zero length data")
		return
	}
	for _, d := range data {
		if len(d) != 0 {
			bufferBytes = uint64(len(d))
			if bufferBytes%64 != 0 {
				err = ErrInvalidSize
			}
			return
		}
	}
	err = errAllBuffersEmpty
	return
}

// LeoInit function as declared in leopard/leopard.h:105
func LeoInit(version int32) int32 {
	cversion, _ := (C.int)(version), cgoAllocsUnknown
	__ret := C.leo_init_(cversion)
	__v := (int32)(__ret)
	return __v
}

// LeoResultString function as declared in leopard/leopard.h:127
func LeoResultString(result Leopardresult) string {
	cresult, _ := (C.LeopardResult)(result), cgoAllocsUnknown
	__ret := C.leo_result_string(cresult)
	__v := packPCharString(__ret)
	return __v
}

// LeoEncodeWorkCount function as declared in leopard/leopard.h:143
func LeoEncodeWorkCount(originalCount uint32, recoveryCount uint32) uint32 {
	coriginalCount, _ := (C.uint)(originalCount), cgoAllocsUnknown
	crecoveryCount, _ := (C.uint)(recoveryCount), cgoAllocsUnknown
	__ret := C.leo_encode_work_count(coriginalCount, crecoveryCount)
	__v := (uint32)(__ret)
	return __v
}

// LeoEncode function as declared in leopard/leopard.h:180
func LeoEncode(bufferBytes uint64, originalCount uint32, recoveryCount uint32, workCount uint32, originalData []unsafe.Pointer, workData []unsafe.Pointer) Leopardresult {
	cbufferBytes, _ := (C.uint64_t)(bufferBytes), cgoAllocsUnknown
	coriginalCount, _ := (C.uint)(originalCount), cgoAllocsUnknown
	crecoveryCount, _ := (C.uint)(recoveryCount), cgoAllocsUnknown
	cworkCount, _ := (C.uint)(workCount), cgoAllocsUnknown
	coriginalData, _ := (*unsafe.Pointer)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&originalData)).Data)), cgoAllocsUnknown
	cworkData, _ := (*unsafe.Pointer)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&workData)).Data)), cgoAllocsUnknown
	__ret := C.leo_encode(cbufferBytes, coriginalCount, crecoveryCount, cworkCount, coriginalData, cworkData)
	__v := (Leopardresult)(__ret)
	return __v
}

// LeoDecodeWorkCount function as declared in leopard/leopard.h:202
func LeoDecodeWorkCount(originalCount uint32, recoveryCount uint32) uint32 {
	coriginalCount, _ := (C.uint)(originalCount), cgoAllocsUnknown
	crecoveryCount, _ := (C.uint)(recoveryCount), cgoAllocsUnknown
	__ret := C.leo_decode_work_count(coriginalCount, crecoveryCount)
	__v := (uint32)(__ret)
	return __v
}

// LeoDecode function as declared in leopard/leopard.h:227
func LeoDecode(bufferBytes uint64, originalCount uint32, recoveryCount uint32, workCount uint32, originalData []unsafe.Pointer, recoveryData []unsafe.Pointer, workData []unsafe.Pointer) Leopardresult {
	cbufferBytes, _ := (C.uint64_t)(bufferBytes), cgoAllocsUnknown
	coriginalCount, _ := (C.uint)(originalCount), cgoAllocsUnknown
	crecoveryCount, _ := (C.uint)(recoveryCount), cgoAllocsUnknown
	cworkCount, _ := (C.uint)(workCount), cgoAllocsUnknown
	coriginalData, _ := (*unsafe.Pointer)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&originalData)).Data)), cgoAllocsUnknown
	crecoveryData, _ := (*unsafe.Pointer)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&recoveryData)).Data)), cgoAllocsUnknown
	cworkData, _ := (*unsafe.Pointer)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&workData)).Data)), cgoAllocsUnknown
	__ret := C.leo_decode(cbufferBytes, coriginalCount, crecoveryCount, cworkCount, coriginalData, crecoveryData, cworkData)
	__v := (Leopardresult)(__ret)
	return __v
}
