package leopard

func asm_xor_mem(vx []byte, vy []byte, bytes uint64)

func asm_mul_mem(_multiply256LUT []multiply256LUT_t, x []byte, y []byte, log_m ffe_t, bytes uint64)

func asm_ifft_DIT2(x []byte, y []byte, log_m ffe_t, bytes uint64)

func asm_ifft_DIT4(bytes uint64, work [][]byte, dist uint32, log_m01 ffe_t, log_m23 ffe_t, log_m02 ffe_t)

func asm_fft_DIT2(x []byte, y []byte, log_m ffe_t, bytes uint64)

func asm_fft_DIT4(bytes uint64, work [][]byte, dist uint32, log_m01 ffe_t, log_m23 ffe_t, log_m02 ffe_t)
