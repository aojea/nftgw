REPO_ROOT:=${CURDIR}
OUT_DIR=$(REPO_ROOT)/bin
NFTW_BINARY_NAME?=nftgw

# go1.9+ can autodetect GOROOT, but if some other tool sets it ...
GOROOT:=
# enable modules
GO111MODULE=on
# disable CGO by default for static binaries
CGO_ENABLED=0
export GOROOT GO111MODULE CGO_ENABLED


build:
	go build -v -o "$(OUT_DIR)/$(NFTW_BINARY_NAME)" $(NFTW_BUILD_FLAGS) cmd/main.go

clean:
	rm -rf "$(OUT_DIR)/"

test:
	hack/test-all.sh

# code linters
lint:
	hack/lint.sh

# run linters, ensure generated code, etc.
verify:
	hack/verify-all.sh
