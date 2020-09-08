GIT_EXISTS := $(shell git status > /dev/null 2>&1 ; echo $$?)

ifeq ($(GIT_EXISTS), 0)
	ifeq ($(shell git tag --points-at HEAD),)
		# Not currently on a tag
		VERSION := $(shell git describe --tags | sed 's/^v//; s/-.*/-next/') # 1.2.3-next
	else
		# On a tag
		VERSION := $(shell git tag --points-at HEAD)
	endif

	COMMIT := $(shell git rev-parse --verify HEAD)
endif

INSTALL := install -o root -g 0
INSTALL_DIR := /usr/local/bin

.PHONY: all build install clean uninstall fmt

all: build

build:
ifneq ($(GIT_EXISTS), 0)
	# No Git repo
	$(error No Git repo was found, which is needed to compile the commit and version)
endif
	@echo "Downloading dependencies"
	@go env -w GO111MODULE=on ; go mod download
	@echo "Building binary"
	@go env -w GO111MODULE=on CGO_ENABLED=0 ; go build -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.builtBy=Makefile"

install:
	@echo "Installing gemget to $(INSTALL_DIR)"
	@$(INSTALL) -m 755 gemget $(INSTALL_DIR)

clean:
	@echo "Removing gemget binary in local directory"
	@$(RM) gemget

uninstall:
	@echo "Removing gemget from $(INSTALL_DIR)"
	@$(RM) $(INSTALL_DIR)/gemget

fmt:
	go fmt ./...
