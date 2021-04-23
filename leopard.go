package leopard

import (
	"errors"
	"fmt"

	"golang.org/x/sys/cpu"
)

var (
	ErrCpuNoSSE3       = errors.New("CPU does not support SSE3")
	ErrCpuNoAVX2       = errors.New("CPU does not support AVX2")
	ErrNeedMoreData    = errors.New("not enough recovery data received")
	ErrTooMuchData     = errors.New("buffer counts are too high")
	ErrInvalidSize     = errors.New("buffer size must be a multiple of 64 bytes")
	ErrInvalidCounts   = errors.New("invalid counts provided")
	ErrInvalidInput    = errors.New("a function parameter was invalid")
	ErrPlatform        = errors.New("platform is unsupported")
	ErrCallInitialize  = errors.New("call Init() first")
	errAllBuffersEmpty = errors.New("all buffers are empty")
)

// Tracks whether Leopard is initialized
var isInitialized = false

// Init initializes the codec. Must be called before any other functions.
func Init() error {
	if isInitialized {
		return nil
	}

	if !cpu.X86.HasSSE3 {
		return ErrCpuNoSSE3
	}
	if !cpu.X86.HasAVX2 {
		return ErrCpuNoAVX2
	}

	// TODO init FF16

	isInitialized = true
	return nil
}

// Encode takes an slice of equally sized byte slices and computes len(data) parity shares.
// This means you can lose half of (data || encodeWork) and still recover the data.
func Encode(data [][]byte) ([][]byte, error) {
	origCount, bufferBytes, err := extractCounts(data)
	if err != nil {
		return nil, err
	}
	recoveryCount := origCount
	workCount := getEncodeWorkCount(origCount, recoveryCount)

	encodeWork := make([][]byte, workCount)
	for i := uint(0); i < uint(workCount); i++ {
		encodeWork[i] = make([]byte, bufferBytes)
	}

	err = LeoEncode(
		bufferBytes,
		origCount,
		recoveryCount,
		workCount,
		data,
		encodeWork)
	if err != nil {
		return nil, err
	}
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
	decodeWorkCount := getDecodeWorkCount(origCount, recoveryCount)

	decodeWork = make([][]byte, decodeWorkCount)
	for i := uint(0); i < uint(decodeWorkCount); i++ {
		decodeWork[i] = make([]byte, bufferBytes)
	}

	err = LeoDecode(
		bufferBytes,
		origCount,
		recoveryCount,
		decodeWorkCount,
		orig,
		recovery,
		decodeWork)
	if err != nil {
		return
	}

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

// nextPowerOf2 returns the next lowest power of 2 unless the input is a power
// of two, in which case it returns the input
func nextPowerOf2(v uint32) uint32 {
	if v == 1 {
		return 1
	}
	// keep track of the input
	i := v

	// find the next highest power using bit mashing
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++

	// check if the input was the next highest power
	if i == v {
		return v
	}

	// return the next lowest power
	return v / 2
}

func getEncodeWorkCount(originalCount uint32, recoveryCount uint32) uint32 {
	if originalCount == 1 {
		return recoveryCount
	}
	if recoveryCount == 1 {
		return 1
	}

	return nextPowerOf2(recoveryCount) * 2
}

func getDecodeWorkCount(originalCount uint32, recoveryCount uint32) uint32 {
	if originalCount == 1 || recoveryCount == 1 {
		return originalCount
	}

	m := nextPowerOf2(recoveryCount)
	n := nextPowerOf2(m + originalCount)
	return n
}
