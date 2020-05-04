generate:
	c-for-go --ccincl leopard.yml

clean:
	rm -f leopard/cgo_helpers.go leopard/cgo_helpers.h leopard/cgo_helpers.c
	rm -f leopard/const.go leopard/doc.go leopard/types.go
	rm -f leopard/leopard.go
	rm -rf build

build-cleo:
	git submodule update --init --recursive
	mkdir -p leopard/build && cd leopard/build && cmake ../leopard
	cd leopard/build && make libleopard

test:
	cd leopard && go test
	