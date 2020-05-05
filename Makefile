test: build-cleo
	cd leopard && GODEBUG=cgocheck=2 go test

generate:
	c-for-go --ccincl leopard.yml

build-cleo:
	git submodule update --init --recursive
	mkdir -p leopard/build && cd leopard/build && cmake ../leopard
	cd leopard/build && make libleopard

clean:
	rm -f leopard/cgo_helpers.go leopard/cgo_helpers.h leopard/cgo_helpers.c
	rm -f leopard/const.go leopard/doc.go leopard/types.go
	rm -f leopard/leopard.go
	rm -rf leopard/build
