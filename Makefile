.PHONY: all check test lint build clean

all: build test

check: lint test

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f custom-gcl

custom-gcl: .custom-gcl.example.yml
	cp .custom-gcl.example.yml .custom-gcl.yml
	golangci-lint custom

.PHONY: testdata
testdata:
	@echo "Checking testdata module is buildable..."
	cd testdata && go build ./...
