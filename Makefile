.PHONY: build
build: test
	go build ./...

###


.PHONY: fmt
fmt:
	go mod tidy
	go fmt ./...

.PHONY: mock
mock:
	go generate ./... # mock

.PHONY: test
test: fmt mock
	ginkgo -r .

