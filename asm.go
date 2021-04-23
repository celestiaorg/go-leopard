// +build ignore
//go:generate go run asm.go -out asm.s -pkg leopard -stubs asmstub.go

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func main() {
	TEXT("xor_mem", NOSPLIT, "func(vx []byte, vy []byte, bytes uint64)")
	x32 := Mem{Base: Load(Param("vx").Base(), GP64())}
	y32 := Mem{Base: Load(Param("vy").Base(), GP64())}
	bytes := Load(Param("bytes"), GP64())

	x0 := YMM()
	x1 := YMM()
	x2 := YMM()
	x3 := YMM()
	for bytes >= 128 {
		VPXOR(x0, x32, y32)
		x32 += 4
		y32 += 4
		bytes -= 128
	}
	if bytes > 0 {
	}
	RET()

	Generate()
}
