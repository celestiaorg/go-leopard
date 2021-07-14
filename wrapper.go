package leopard

//#include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	// to avoid repeatedly writing leopard.Leopard* below:
	. "github.com/celestiaorg/go-leopard/leopard"
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
