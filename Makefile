BIN_OUT = ./cmd/shelf


build:
	go build -o $(BIN_OUT) ./internal/domain/book/...

test:
	go test ./internal/...