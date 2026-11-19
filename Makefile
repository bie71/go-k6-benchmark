.PHONY: all build run test bench k6-smoke k6-full clean

all: build

build:
	go build -v ./...

run:
	go run .

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./...

k6-smoke:
	k6 run --vus 5 --duration 10s k6/load.js

k6-full:
	k6 run k6/load.js

clean:
	rm -f go-k6-benchmark hasil.json
