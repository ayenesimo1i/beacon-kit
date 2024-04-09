#!/usr/bin/make -f

export VERSION := $(shell echo $(shell git describe --tags --always --match "v*") | sed 's/^v//')
export COMMIT := $(shell git log -1 --format='%H')
CURRENT_DIR = $(shell pwd)
OUT_DIR ?= $(CURDIR)/build/bin
BINDIR ?= $(GOPATH)/build/bin
TESTAPP_DIR = beacond
TESTAPP_CMD_DIR = $(TESTAPP_DIR)/cmd
PROJECT_NAME = $(shell git remote get-url origin | xargs basename -s .git)

# process build tags
build_tags = netgo

ifeq (legacy,$(findstring legacy,$(COSMOS_BUILD_OPTIONS)))
  build_tags += app_v1
endif

# DB backend selection
ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += gcc
endif
ifeq (badgerdb,$(findstring badgerdb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += badgerdb
endif
# handle rocksdb
ifeq (rocksdb,$(findstring rocksdb,$(COSMOS_BUILD_OPTIONS)))
  CGO_ENABLED=1
  build_tags += rocksdb grocksdb_clean_link
endif
# handle boltdb
ifeq (boltdb,$(findstring boltdb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += boltdb
endif

# always include pebble
build_tags += pebbledb

# always include blst
build_tags += blst

# always include ckzg
build_tags += ckzg
build_tags += cgo

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=sim \
		-X github.com/cosmos/cosmos-sdk/version.AppName=simd \
		-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		-X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)"

ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

build_tags += $(BUILD_TAGS)
# build_tags := $(strip $(build_tags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

# Check for debug option
ifeq (debug,$(findstring debug,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -gcflags "all=-N -l"
endif

# This allows us to reuse the build target steps for both go build and go install
BUILD_TARGETS := build install

## Build: 
build: BUILD_ARGS=-o $(OUT_DIR)/beacond ## build `beacond`

$(BUILD_TARGETS): $(OUT_DIR)/
	@echo "Building ${TESTAPP_CMD_DIR}"
	@cd ${CURRENT_DIR}/$(TESTAPP_CMD_DIR) && go $@ -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./.

$(OUT_DIR)/:
	mkdir -p $(OUT_DIR)/

	# Variables
ARCH ?= $(shell uname -m)
ifeq ($(ARCH),)
	ARCH = arm64
endif
GO_VERSION ?= 1.22.1
IMAGE_NAME ?= beacond

# Docker Paths
DOCKERFILE = ./Dockerfile

build-docker: ## build a docker image containing `beacond`
	@echo "Build a release docker image for the Cosmos SDK chain..."
	docker build \
	--build-arg GO_VERSION=$(GO_VERSION) \
	--platform linux/$(ARCH) \
	--build-arg GIT_COMMIT=$(shell git rev-parse HEAD) \
	--build-arg GIT_VERSION=$(shell git describe --tags --always --dirty) \
	--build-arg GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD) \
	--build-arg GOOS=linux \
	--build-arg GOARCH=$(ARCH) \
	-f ${DOCKERFILE} \
	-t $(IMAGE_NAME):$(VERSION) \
	.