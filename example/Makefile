all: examples
gx-bins:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go

deps: gx-bins
	gx --verbose install --global
	gx-go rewrite
examples: deps example blockhandler/blockhandler
example:
	go build example.go
blockhandler/blockhandler:
	cd blockhandler && go build blockhandler.go
clean: gx-bins
	gx-go rewrite --undo
	rm -rf example blockhandler/blockhandler

.PHONY = all gx-bins deps examples clean
