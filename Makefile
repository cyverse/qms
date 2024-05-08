all: qms

install-swagger:
	which swagger || go install github.com/go-swagger/go-swagger/cmd/swagger@latest

swagger.json: install-swagger
	swagger generate spec -o ./swagger.json --scan-models

qms: swagger.json
	go build --buildvcs=false .

clean:
	rm -rf qms swagger.json

lint:
	golangci-lint run

test:
	go test ./...

.PHONY: install-swagger clean all lint test
