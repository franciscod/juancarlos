build:
	go fmt
	go mod tidy
	go build

run: build
	$(shell cat .env) ./juancarlos --server "127.0.0.1:64738" --insecure --username "juancarlos" --certificate cert/cert.pem --key cert/key.pem audio/

