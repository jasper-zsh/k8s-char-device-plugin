.PHONY: build
build: proto go

.PHONY: go
go:
	@echo Compiling golang
	go build -o k8s-char-device-plugin main.go

.PHONY: proto
proto:
	@echo Compiling protos
	protoc --go-grpc_out=. --go_out=. proto/*.proto

.PHONY: docker
docker:
	@echo Building docker image
	docker build -t k8s-char-device-plugin .
