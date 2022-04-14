//go:build ignore

//go:generate go run asm.go -out asm.s -pkg leopard

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func main() {
	Package("github.com/celestiaorg/go-leopard")

	// Size of multiply256LUT_t: 2*4*32 bytes
	const size_multiply256LUT_t = uint64(256)

	////////////////////////////////////////////////////////////////////////////
	// func xor_mem
	////////////////////////////////////////////////////////////////////////////
	{
		const unroll = 4
		Implement("asm_xor_mem")

		x32 := Mem{Base: Load(Param("vx").Base(), GP64())}
		y32 := Mem{Base: Load(Param("vy").Base(), GP64())}
		bytes := Load(Param("bytes"), GP64())

		rx := YMM()
		ry := YMM()

		Label("loop1")
		Comment("for bytes >= 128")
		CMPQ(bytes, Imm(128))
		JLT(LabelRef("done1"))

		for i := 0; i < unroll; i++ {
			VMOVDQU(rx, x32.Offset(i*32))
			VMOVDQU(ry, y32.Offset(i*32))
			VPXOR(rx, rx, ry)
			VMOVDQU(x32.Offset(i*32), rx)
		}
		Comment("x32 += 4")
		ADDQ(Imm(unroll*32), x32)
		Comment("y32 += 4")
		ADDQ(Imm(unroll*32), y32)
		Comment("bytes -= 128")
		SUBQ(Imm(unroll*32), bytes)
		Label("done1")

		Label("loop2")
		Comment("if bytes > 0")
		CMPQ(bytes, Imm(0))
		JLE(LabelRef("done2"))
		for i := 0; i < 2; i++ {
			VMOVDQU(rx, x32.Offset(i*32))
			VMOVDQU(ry, y32.Offset(i*32))
			VPXOR(rx, rx, ry)
			VMOVDQU(x32.Offset(i*32), rx)
		}

		Label("done2")

		RET()
	}

	////////////////////////////////////////////////////////////////////////////
	// func xor_mem
	////////////////////////////////////////////////////////////////////////////
	{
		const unroll = 2
		const numTables = 1
		Implement("asm_mul_mem")

		// _multiply256LUT := Mem{Base: Load(Param("_multiply256LUT").Base(), GP64())}
		x32 := Mem{Base: Load(Param("x").Base(), GP64())}
		y32 := Mem{Base: Load(Param("y").Base(), GP64())}
		// log_m := Load(Param("log_m"), GP64())
		bytes := Load(Param("bytes"), GP64())

		// LEO_MUL_TABLES_256

		T_lo := make([][]VecVirtual, numTables)
		T_hi := make([][]VecVirtual, numTables)
		for i := 0; i < numTables; i++ {
			T_lo[i] = make([]VecVirtual, 4)
			T_hi[i] = make([]VecVirtual, 4)

			for j := 0; j < 4; j++ {
				T_lo[i][j] = YMM()
				VXORPS(T_lo[i][j], T_lo[i][j], T_lo[i][j])
				T_hi[i][j] = YMM()
				VXORPS(T_hi[i][j], T_hi[i][j], T_hi[i][j])
			}

			r_multiply256LUT := Mem{Base: Load(Param("_multiply256LUT").Base(), GP64())}
			r_log_m := Load(Param("log_m"), GP64())
			r_sizeLUT := GP64()

			// MOVQ(Imm(size_multiply256LUT_t), r_sizeLUT)
			MOVQ(r_log_m, RAX)
			MULQ(r_sizeLUT)
			ADDQ(r_multiply256LUT, RAX)
			for j := 0; j < 4; j++ {
				VMOVDQU(T_lo[i][j], r_multiply256LUT)
				ADDQ(Imm(32), r_multiply256LUT)
			}
		}

		// TODO
		// const LEO_M256 clr_mask = _mm256_set1_epi8(0x0f);

		Label("loop1")
		Comment("do")

		// TODO
		// #define LEO_MUL_256_LS(x_ptr, y_ptr) { \
		//     const LEO_M256 data_lo = _mm256_loadu_si256(y_ptr); \
		//     const LEO_M256 data_hi = _mm256_loadu_si256(y_ptr + 1); \
		//     LEO_M256 prod_lo, prod_hi; \
		//     LEO_MUL_256(data_lo, data_hi, 0); \
		//     _mm256_storeu_si256(x_ptr, prod_lo); \
		//     _mm256_storeu_si256(x_ptr + 1, prod_hi); }
		//     LEO_MUL_256_LS(x32, y32);

		Comment("y32 += 2")
		ADDQ(Imm(unroll*32), y32)
		Comment("x32 += 2")
		ADDQ(Imm(unroll*32), x32)

		Comment("bytes -= 64")
		SUBQ(Imm(64), bytes)

		Comment("while (bytes > 0)")
		CMPQ(bytes, Imm(0))
		JGT(LabelRef("loop1"))
		Label("done1")

		RET()
	}

	////////////////////////////////////////////////////////////////////////////
	// Generate asm
	////////////////////////////////////////////////////////////////////////////
	Generate()
}
