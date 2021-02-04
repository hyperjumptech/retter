GO111MODULE=on

.PHONY: all test test-short fix-antlr4-bug build

build-linux:
	mkdir -p build/linux
	env GOOS=linux GOARCH=amd64 go build -o build/linux ./...

build-windows:
	mkdir -p build/windows
	env GOOS=windows GOARCH=amd64 go build -o build/windows ./...

lint: build-linux build-windows
	go get -u golang.org/x/lint/golint
	golint -set_exit_status .

test: lint
	go test ./... -covermode=count -coverprofile=coverage.out

test-coverage: test
	go tool cover -html=coverage.out

clean:
	rm -f build/retter.*