generate:
	c-for-go --ccincl leopard.yml

clean:
	rm -f leopard/cgo_helpers.go leopard/cgo_helpers.h leopard/cgo_helpers.c
	rm -f leopard/const.go leopard/doc.go leopard/types.go
	rm -f leopard/leopard.go

test:
	cd leopard && go build
	