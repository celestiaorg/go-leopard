// +build ignore
//go:generate go run asm.go -out asm.s -pkg leopard -stubs asmstub.go

package main

import . "github.com/mmcloughlin/avo/build"

func main() {
	Generate()
}
