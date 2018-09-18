all: golint vet serv pd

GLOCK := glock

$(GLOCK): 
	go get github.com/robfig/glock

build: $(GLOCK)
	$(GLOCK) sync github.com/taorenhai/ancestor/

golint:
	golint client/
	golint cmd/...
	golint config/

	@echo "golint meta"
	@find ./meta/ -name '*.go' ! -name '*.pb.go' | xargs golint

	golint storage/

	@echo "golint storage/engine"
	@find ./storage/engine/ -name '*.go' ! -name '*.pb.go' | xargs golint

	golint pd/
	golint util/...

clean:
	@rm -rf bin

fmt:
	gofmt -s -l -w .
	goimports -l -w .

vet:
	go tool vet . 2>&1
	go tool vet --shadow . 2>&1

serv:
	go build -o bin/nodeserver ./cmd/node/main.go

pd:
	go build -o bin/pdserver ./cmd/pd/main.go

tool:
	go build -o bin/tool ./cmd/tool/main.go

rocksdbtool:
	go build -o bin/rocksdbtool ./cmd/rocksdbtool/main.go

.PHONY: pd serv tool

test:
	go test -v ./client
	go test -v ./storage/engine
	go test -v ./storage/
	go test -v ./pd
