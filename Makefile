BINARY_SUFFIX=
BINARY=go-dnsmasq${BINARY_SUFFIX}
VERSION=1.2.0
B_LDFLAGS=-X main.Version=${VERSION}
R_LDFLAGS=-w -s -X main.Version=${VERSION}
TEST=$$(go list ./... | grep -v 'vendor')
OS_ARCH=darwin_arm64 # MacOS
# OS_ARCH=linux_amd64 % Linux
GOLANG_CI_LINT_VERSION=v1.62.2

.PHONY: lint build release clean

default: build

fmt:
# Run go fmt on all Go files
	find . -name "*.go" -exec go fmt {} \;

lint:
# Run linters in docker
	docker run -t --rm -v ${CURDIR}:/app -v /tmp/cache/golangci-lint/${GOLANG_CI_LINT_VERSION}:/root/.cache -w /app golangci/golangci-lint:${GOLANG_CI_LINT_VERSION} golangci-lint run

lint-ci:
# lint command to use inside CI, to avoid docker-in-docker
	if ! command -v golangci-lint; then go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANG_CI_LINT_VERSION}; fi
	golangci-lint run -v

build:
	go build --ldflags "${B_LDFLAGS}" -o ${BINARY}

release:
# Darwin
	GOOS=darwin GOARCH=amd64 go build --ldflags "${R_LDFLAGS}" -o ./binaries/${BINARY}_darwin_amd64
	GOOS=darwin GOARCH=arm64 go build --ldflags "${R_LDFLAGS}" -o ./binaries/${BINARY}_darwin_arm64
# Linux
	GOOS=linux GOARCH=amd64 go build --ldflags "${R_LDFLAGS}" -o ./binaries/${BINARY}_linux_amd64
	GOOS=linux GOARCH=arm64 go build --ldflags "${R_LDFLAGS}" -o ./binaries/${BINARY}_linux_arm64

test:
	go test ./...  -list=. -timeout=30s -parallel=1

clean:
	go clean
	rm -f ${BINARY}
	rm -Rf ./binaries
