.PHONY: build test run clean initdb dropdb gh_home_page gh_daily_page

BINARY = hndigest

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY) home

gh_home_page: build
	rm -rf output/static
	./$(BINARY) home

gh_daily_page: build
	./$(BINARY) daily

test:
	go test ./...

clean:
	rm -f $(BINARY)
	rm -rf output/static

initdb:
	go run . home 2>&1 || true

dropdb:
	rm -f hackernews.db

format:
	go fmt ./...

vet:
	go vet ./...
