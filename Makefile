#
# Makefile
#
# Simple makefile to build binary.
#
# @authors Minigun Maintainers
# @copyright 2019 Wayfair, LLC. -- All rights reserved.

VENDOR_DIR = vendor

.PHONY: get-deps
get-deps: $(VENDOR_DIR)

$(VENDOR_DIR):
	GO111MODULE=on go mod vendor

$(OUTPUT_DIR):
	mkdir output

.PHONY: build
build: $(VENDOR_DIR) $(OUTPUT_DIR)
	GOOS=linux CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o output/minigun .

.PHONY: local-build
local-build: $(VENDOR_DIR) $(OUTPUT_DIR)
	CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o output/minigun .

.PHONY: local-build-wo-cgo
local-build: $(VENDOR_DIR) $(OUTPUT_DIR)
	CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o output/minigun .

.PHONY: clean
clean:
	rm -f minigun
	rm -f output/*

.PHONY: test
test: $(VENDOR_DIR)
	go test -v -timeout 30s github.com/wayfair-incubator/minigun

# Some Github targets

.PHONY: build-ubuntu-latest
build-ubuntu-latest: $(VENDOR_DIR) $(OUTPUT_DIR)
	CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o output/minigun .
	tar czf output/minigun-linux64.tar.gz output/minigun
	rm -f output/minigun

.PHONY: build-macos-latest
build-macos-latest: $(VENDOR_DIR) $(OUTPUT_DIR)
	CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o output/minigun .
	tar czf output/minigun-darwin64.tar.gz output/minigun
	rm -f output/minigun

.PHONY: build-windows-latest
build-windows-latest: $(VENDOR_DIR) $(OUTPUT_DIR)
	CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o output/minigun.exe .
	tar czf output/minigun-win64.tar.gz output/minigun.exe
	rm -f output/minigun.exe
