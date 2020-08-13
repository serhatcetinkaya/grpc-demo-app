GO := go
DOCKER := docker

SERVER_IMAGE := grpc-rate-limiting-example-server
CLIENT_IMAGE := grpc-rate-limiting-example-client

build: dependencies protoc server client

protoc:
	@echo "** Generating Go files from proto"
	@cd proto/math && protoc --go_out=plugins=grpc:. *.proto
	@echo "** Done"
.PHONY: protoc

dependencies:
	@echo "** Getting dependencies"
	@$(GO) get google.golang.org/grpc
.PHONY: dependencies

server:
	@echo "** Building local server"
	@$(GO) build -o bin/server github.com/serhatcetinkaya/grpc-demo-app/app/server
	@echo "** Done"
.PHONY: server

client:
	@echo "** Building local client"
	@$(GO) build -o bin/client github.com/serhatcetinkaya/grpc-demo-app/app/client
	@echo "** Done"
.PHONY: client

client-docker:
	@echo "** Cross compiling client"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build \
		-o bin/client-linux github.com/serhatcetinkaya/grpc-demo-app/app/client
	@echo "** Building a docker container for client"
	@$(DOCKER) build -f app/client/Dockerfile -t $(CLIENT_IMAGE) .
	@echo "** Done"
.PHONY: client-docker

server-docker:
	@echo "** Cross compiling server"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build \
		-o bin/server-linux github.com/serhatcetinkaya/grpc-demo-app/app/server
	@echo "** Building a docker container for server"
	@$(DOCKER) build -f app/server/Dockerfile -t $(SERVER_IMAGE) .
	@echo "** Done"
.PHONY: server-docker

clean:
	@echo "** Cleaning binaries"
	@rm -rf bin/*
	@echo "** Cleaning docker images"
	@$(DOCKER) rmi -f $(SERVER_IMAGE) $(CLIENT_IMAGE) 2>/dev/null 1>&2
	@echo "** Done"
.PHONY: clean
