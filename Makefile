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
local-build-wo-cgo: $(VENDOR_DIR) $(OUTPUT_DIR)
	CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o output/minigun .

.PHONY: clean
clean:
	rm -f output/*

.PHONY: clean-all
clean-all:
	rm -rf output/* vendor

.PHONY: test
test: $(VENDOR_DIR)
	go test -v -timeout 30s github.com/wayfair-incubator/minigun

# Some Github targets

.PHONY: build-ubuntu-latest-amd64
build-ubuntu-latest: $(VENDOR_DIR) $(OUTPUT_DIR)
	CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o output/minigun .
	cd output && tar czf minigun-linux-amd64.tar.gz minigun
	rm -f output/minigun

.PHONY: build-macos-latest-amd64
build-macos-latest: $(VENDOR_DIR) $(OUTPUT_DIR)
	CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o output/minigun .
	cd output && tar czf minigun-darwin-amd64.tar.gz minigun
	rm -f output/minigun

.PHONY: build-ubuntu-latest-arm64
build-ubuntu-latest: $(VENDOR_DIR) $(OUTPUT_DIR)
	GOARCH=arm64 CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o output/minigun .
	cd output && tar czf minigun-linux-arm64.tar.gz minigun
	rm -f output/minigun

.PHONY: build-macos-latest-arm64
build-macos-latest: $(VENDOR_DIR) $(OUTPUT_DIR)
	GOARCH=arm64 CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o output/minigun .
	cd output && tar czf minigun-darwin-arm64.tar.gz minigun
	rm -f output/minigun

.PHONY: build-windows-latest
build-windows-latest: $(VENDOR_DIR) $(OUTPUT_DIR)
	CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o output/minigun.exe .
	cd output && tar czf minigun-win64.tar.gz minigun.exe
	rm -f output/minigun.exe
