


.PHONY: all
all: container


.PHONY: container
container: *.go
	mkdir -p ./bin
	go fmt ./... && go vet ./...
	go build -o ./bin/sctr .

.PHONY: test
test: container
	./bin/sctr container run docker.io/openanolis/anolisos:23 -- /bin/bash -c "ls /"
