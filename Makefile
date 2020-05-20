# default target: build if necessary and run tests
test: build-cleo
	GODEBUG=cgocheck=2 go test -v ./...

# only necessary if you need to re-generate the c-go bindings
# Note: deletes all previously generated c-go bindings and
# any build
re-generate: clean
	c-for-go --ccincl leopard.yml

# init leopard submodule and build C library
build-cleo:
	git submodule update --init --recursive
	mkdir -p leopard/build && cd leopard/build && cmake ../leopard
	cd leopard/build && make libleopard

# clean generated files and build artifacts
clean:
	rm -f leopard/cgo_helpers.go leopard/cgo_helpers.h leopard/cgo_helpers.c
	rm -f leopard/const.go leopard/doc.go leopard/types.go
	rm -f leopard/leopard.go
	rm -rf leopard/build
