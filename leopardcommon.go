package leopard

type XORSummer struct{}

func (x *XORSummer) initialize(recoveryData [][]byte) {}
func (x *XORSummer) add(originalData []byte)          {}
func (x *XORSummer) finalize()                        {}
