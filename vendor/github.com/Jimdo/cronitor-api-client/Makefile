
swagger:
	go get -u github.com/go-swagger/go-swagger/cmd/swagger

clean:
	rm -rf client models

generate: swagger clean
	swagger generate client -f swagger.yaml 

install:
	go get -u -f ./...
	go get -u github.com/stretchr/testify/assert
	go get -u github.com/stretchr/testify/require

test:
	go test ./...
