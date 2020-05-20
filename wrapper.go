package leopard

//#include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	. "github.com/liamsi/go-leopard/leopard"
)

var (
	ErrNeedMoreData  = errors.New("not enough recovery data received")
	ErrTooMuchData   = errors.New("buffer counts are too high")
	ErrInvalidSize   = errors.New("buffer size must be a multiple of 64 bytes")
	ErrInvalidCounts = errors.New("invalid counts provided")
	ErrInvalidInput  = errors.New("a function parameter was invalid")
	ErrPlatform      = errors.New("platform is unsupported")

	ErrCallInitialize = errors.New("call Init() first")
)

const version = 2

func init() {
	if err := Init(); err != nil {
		panic(fmt.Sprintf("Unexpected error while initializing leopard %v", err))
	}
}

func leopardResultToErr(errCode Leopardresult) error {
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
	return leopardResultToErr(Leopardresult(LeoInit(version)))
}

// Encode takes an slice of equally sized byte slices and computes len(data) parity shares.
// This means you can lose half of (data || encodeWork) and still recover the data.
func Encode(data [][]byte) (encodeWork [][]byte, err error) {
	origCount, bufferBytes, err := extract(data)
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
		res[i] = copyByteBuffer(d)
	}
	return res
}

func copyByteBuffer(d []byte) unsafe.Pointer {
	if len(d) > 0 {
		return C.CBytes(d)
	} else {
		// keep this nil as Leopard uses this internally
		return nil
	}
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

// wrapper around C.freeAll (can also be used in tests)
func freeAndNil(p unsafe.Pointer) {
	if p != nil {
		C.free(p)
	}
}

func freeAll(ps []unsafe.Pointer) {
	for _, p := range ps {
		freeAndNil(p)
	}
}
