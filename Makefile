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

.PHONY: build
build: $(VENDOR_DIR)
	GOOS=linux CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o minigun .

.PHONY: local-build
local-build: $(VENDOR_DIR)
	CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o minigun .

.PHONY: clean
clean:
	rm -f minigun
	rm -f *.tgz

.PHONY: test
test: $(VENDOR_DIR)
	go test -v -timeout 30s github.com/wayfair-incubator/minigun

.PHONY: release
release: $(VENDOR_DIR)
	mkdir -p release
	GOOS=linux CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o minigun .
	tar czf release/minigun-linux64.tgz minigun
	rm -f minigun
	GOOS=darwin CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o minigun .
	tar czf release/minigun-darwin64.tgz minigun
	rm -f minigun
