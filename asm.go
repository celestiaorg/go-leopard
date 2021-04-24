// +build ignore
//go:generate go run asm.go -out asm.s -pkg leopard

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	// . "github.com/mmcloughlin/avo/reg"
)

func main() {
	Package("github.com/lazyledger/go-leopard")

	////////////////////////////////////////////////////////////////////////////
	// func xor_mem
	////////////////////////////////////////////////////////////////////////////
	const unroll = 4
	Implement("asm_xor_mem")
	x32 := Mem{Base: Load(Param("vx").Base(), GP64())}
	y32 := Mem{Base: Load(Param("vy").Base(), GP64())}
	bytes := Load(Param("bytes"), GP64())

	rx := YMM()
	ry := YMM()

	Label("L1")
	Comment("for bytes >= 128")
	CMPQ(bytes, Imm(128))
	JLT(LabelRef("Z1"))

	for i := 0; i < unroll; i++ {
		VMOVDQU(rx, x32.Offset(i))
		VMOVDQU(ry, y32.Offset(i))
		VPXOR(rx, rx, ry)
		VMOVDQU(x32.Offset(i), rx)
	}
	// TODO sizeof
	Comment("x32 += 4")
	ADDQ(Imm(unroll), x32)
	Comment("y32 += 4")
	ADDQ(Imm(unroll), y32)
	Comment("bytes -= 128")
	SUBQ(Imm(unroll*32), bytes)

	Label("Z1")
	Comment("if bytes > 0")
	CMPQ(bytes, Imm(0))
	JLE(LabelRef("Z2"))
	for i := 0; i < 2; i++ {
		VMOVDQU(rx, x32.Offset(i))
		VMOVDQU(ry, y32.Offset(i))
		VPXOR(rx, rx, ry)
		VMOVDQU(x32.Offset(i), rx)
	}

	Label("Z2")
	RET()

	////////////////////////////////////////////////////////////////////////////
	// Generate asm
	////////////////////////////////////////////////////////////////////////////
	Generate()
}
