.PHONY: build-all test

build-all: test ./bin/hkswitch-darwin-amd64 ./bin/hkswitch-linux-armhf

test:
	go test -v -race -count 1 -timeout 10s ./...

./bin/hkswitch-darwin-amd64: export GOOS=darwin
./bin/hkswitch-darwin-amd64: export GOARCH=amd64

./bin/hkswitch-linux-armhf: export GOOS=linux
./bin/hkswitch-linux-armhf: export GOARCH=arm
./bin/hkswitch-linux-armhf: export GOARM=6

./bin/%: test Makefile $(shell find . -type f -name "*.go" -not -path './vendor/*' -not -path './bin/*' -not -path './.*')
	@echo Build $@ with GOOS=$(GOOS), GOARCH=$(GOARCH)...
	go build -ldflags="-s -w" -o $@ main.go
	# upx --brute $@
	file $@
