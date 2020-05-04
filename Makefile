generate:
	c-for-go --ccincl leopard.yml

clean:
	rm -f leopard/cgo_helpers.go leopard/cgo_helpers.h leopard/cgo_helpers.c
	rm -f leopard/const.go leopard/doc.go leopard/types.go
	rm -f leopard/leopard.go

build-cleo:
	git submodule update --init --recursive
	cmake -S leopard/leopard/ -B leopard/build
	cd leopard/build && make libleopard

test:
	cd leopard && go build
	