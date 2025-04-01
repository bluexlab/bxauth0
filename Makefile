TOPDIR := $(patsubst %/,%,$(strip $(dir $(realpath $(lastword $(MAKEFILE_LIST))))))

CGO_ENABLED ?= 0
ifneq (,$(wildcard $(TOPDIR)/.env))
	include $(TOPDIR)/.env
	export
endif
ifneq (,$(wildcard $(TOPDIR)/.${ENVIRONMENT}.env))
	include $(TOPDIR)/.${ENVIRONMENT}.env
	export
endif
ifneq (,$(wildcard $(TOPDIR)/.local.env))
	include $(TOPDIR)/.local.env
	export
endif

comma:= ,
empty:=
space:= $(empty) $(empty)

bold := $(shell tput bold)
green := $(shell tput setaf 2)
sgr0 := $(shell tput sgr0)

GOLANGCI := $(shell command -v golangci-lint 2> /dev/null)
REVIVE := $(shell command -v revive 2> /dev/null)
GOTESTSUM := $(shell command -v gotestsum 2> /dev/null)
GOSEC := $(shell command -v gosec 2> /dev/null)

APP := bxauth0
IMAGE := local/$(APP)
MODULE_NAME := $(shell go list -m)

PLATFORM ?= $(platform)
ifneq ($(PLATFORM),)
	GOOS := $(or $(word 1, $(subst /, ,$(PLATFORM))),$(shell go env GOOS))
	GOARCH := $(or $(word 2, $(subst /, ,$(PLATFORM))),$(shell go env GOARCH))
endif

BIN_SUFFIX :=
ifneq ($(or $(GOOS),$(GOARCH)),)
	GOOS ?= $(shell go env GOOS)
	GOARCH ?= $(shell go env GOARCH)
	BIN_SUFFIX := $(BIN_SUFFIX)-$(GOOS)-$(GOARCH)
endif
ifeq ($(GOOS),windows)
	BIN_SUFFIX := $(BIN_SUFFIX).exe
endif

BINS := $(patsubst cmd/%/,%,$(sort $(dir $(wildcard cmd/*/main.go))))
GOFILES := $(shell find . -type f -name '*.go' -not -path '*/\.*' -not -path './cmd/*')
$(foreach bin,$(BINS),\
	$(eval GOFILES_$(subst /,-,$(bin)) := $(shell find ./cmd/$(bin) -type f -name '*.go' -not -path '*/\.*')))

PROTO_DIR := proto
PROTO_SOURCES := $(wildcard $(PROTO_DIR)/**/*.proto)
PROTO_FILES := $(patsubst $(PROTO_DIR)/%,%,$(PROTO_SOURCES))
PROTO_GRPC_SOURCES := $(shell grep -slrw $(PROTO_DIR) -e '^service')
PROTO_GRPC_FILES := $(patsubst $(PROTO_DIR)/%,%,$(PROTO_GRPC_SOURCES))

PROTO_GO_DIR := pkg/proto
PROTO_GO_FILES := $(patsubst %.proto,$(PROTO_GO_DIR)/%.pb.go,$(PROTO_FILES))
PROTO_GO_GRPC_FILES := $(patsubst %.proto,$(PROTO_GO_DIR)/%_grpc.pb.go,$(PROTO_GRPC_FILES))

PROTO_GO_OPTS := $(patsubst %,M%,$(join $(PROTO_FILES),$(patsubst %/,=$(MODULE_NAME)/%,$(dir $(PROTO_GO_FILES)))))
PROTO_GO_OUT_OPT := $(subst $(space),$(comma),$(PROTO_GO_OPTS))

PROTO_MOCK_DIR := test/mock/proto
PROTO_MOCK_SOURCES :=
#PROTO_MOCK_SOURCES += $(PROTO_GO_DIR)/hello_world/greeting_grpc.pb.go
PROTO_MOCK_FILES := $(patsubst $(PROTO_GO_DIR)/%,$(PROTO_MOCK_DIR)/%,$(PROTO_MOCK_SOURCES))

# gRPC mock stuff
# $(PROTO_MOCK_DIR)/hello_proto/hello_grpc.pb.go: INTERFACES=HelloServiceServer
# $(PROTO_MOCK_DIR)/world_proto/world_grpc.pb.go: INTERFACES=WorldServiceServer,WorldServiceClient

MOCK_DIR := test/mock
MOCK_SOURCES :=
#MOCK_SOURCES += pkg/database/interface.go

MOCK_FILES := $(patsubst pkg/%,$(MOCK_DIR)/%,$(MOCK_SOURCES))

.DEFAULT_GOAL: all
.DEFAULT: all

.PHONY: all
all: $(BINS)

.PHONY: $(BINS)
$(BINS): %: bin/%$(BIN_SUFFIX)

.PHONY: pb-sync
pb-sync: ## Download protocol buffer schema files
	@$(MAKE) -C $(PROTO_DIR) sync

.PHONY: pb-fmt
pb-fmt:
	@$(MAKE) -C $(PROTO_DIR) fmt

.PHONY: pb
pb: $(PROTO_GO_FILES) $(PROTO_GO_GRPC_FILES)

$(PROTO_GO_DIR)/%.pb.go: $(PROTO_DIR)/%.proto
	@mkdir -p $(@D)
	@printf "Generating $(bold)$@$(sgr0) ... "
	@protoc -I $(PROTO_DIR) \
		--go_out=$(PROTO_GO_OUT_OPT):$(PROTO_GO_DIR) \
		--go_opt=paths=source_relative \
		$<
	@printf "$(green)done$(sgr0)\n"

$(PROTO_GO_DIR)/%_grpc.pb.go: $(PROTO_DIR)/%.proto
	@mkdir -p $(@D)
	@printf "Generating $(bold)$@$(sgr0) ... "
	@protoc -I $(PROTO_DIR) \
		--go-grpc_out=$(PROTO_GO_OUT_OPT):$(PROTO_GO_DIR) \
		--go-grpc_opt=paths=source_relative \
		$<
	@printf "$(green)done$(sgr0)\n"

.PHONY: mock
mock: pb $(PROTO_MOCK_FILES) $(MOCK_FILES)

.PHONY: mock-clean
mock-clean:
	@rm -f $(PROTO_MOCK_FILES)
	@rm -f $(MOCK_FILES)
	@rm -rf test/mock

$(PROTO_MOCK_FILES): test/mock/% : pkg/%
	@mkdir -p $(@D)
	@printf "Generating $(bold)$@$(sgr0) ... "
	@mockgen $(patsubst %/,%,$(dir $(addprefix $(MODULE_NAME)/,$<))) $(INTERFACES) > $@
	@printf "$(green)done$(sgr0)\n"

$(MOCK_FILES): test/mock/% : pkg/%
	@mkdir -p $(@D)
	@printf "Generating $(bold)$@$(sgr0) ... "
	@mockgen -source $< -destination $@
	@printf "$(green)done$(sgr0)\n"

.SECONDEXPANSION:
bin/%: $$(GOFILES) $$(GOFILES_$$(subst /,-,%)) $$(PROTO_GO_FILES) $$(PROTO_GO_GRPC_FILES)
	@printf "Building $(bold)$@$(sgr0) ... "
	@go build -o $@ $(@:bin/%$(BIN_SUFFIX)=./cmd/%)
	@printf "$(green)done$(sgr0)\n"

.PHONY: vet
vet: ## Run the vet static analysis tool
	@go vet ./...

.PHONY: lint
lint: ## Run the linter
ifdef GOLANGCI
	@golangci-lint run --skip-dirs "(^|/)(pkg/proto|test/mock|vendor)($$|/)" --tests=false
else
	$(info "golangci-lint is not available, running 'golint' instead...")
	@golint $(shell go list ./...)
endif

.PHONY: revive
revive: ## Run revive the linter
ifdef REVIVE
	@revive -formatter friendly -exclude ./pkg/proto/... -exclude ./test/mock/... -exclude ./vendor/... ./...
else
	$(info "revive is not available, running 'golint' instead...")
	@golint $(shell go list ./...)
endif

.PHONY: fmt
fmt: ## Reformat source codes
	@go fmt $$(go list ./... | grep -v -E "/pkg/proto/|/test/mock/")
	@-gogroup -order std,prefix=$(MODULE_NAME),other -rewrite $$(find . -type f -name '*.go' -not -path '*/\.*' -not -path './pkg/proto/*' -not -path './test/mock/*')
	@-goimports -w -local $(MODULE_NAME) $$(find . -type f -name '*.go' -not -path '*/\.*' -not -path './pkg/proto/*' -not -path './test/mock/*')
	@-goreturns -w $$(find . -type f -name '*.go' -not -path '*/\.*' -not -path './pkg/proto/*' -not -path './test/mock/*')

.PHONY: test
test: pb mock ## Run unit test
	@go clean -testcache
ifdef GOTESTSUM
	@gotestsum --format standard-verbose -- -p 1 -cover -covermode=count -coverprofile=coverage.out ./...
else
	$(info "gotestsum is not available, running 'go test' instead...")
	@go test -p 1 -v -cover -covermode=count -coverprofile=coverage.out ./...
endif
	@go tool cover -html coverage.out -o coverage.html
	@go tool cover -func coverage.out | tail -n 1

.PHONY: gosec
gosec: ## Run the golang security checker
ifdef GOSEC
	@gosec \
		-exclude-dir pkg/proto \
		-exclude-dir test/mock \
		-exclude-dir vendor \
		./...
else
	$(error "gosec is not available, please install gosec")
endif

.PHONY: docker
docker: ## Build docker image
	@docker build -t $(IMAGE) .

.PHONY: db-create
db-create: ## Create database
	@soda -c config/database.yml create

.PHONY: db-drop
db-drop: ## Drop database
	@soda -c config/database.yml drop

.PHONY: db-migrate
db-migrate: ## Apply the 'up' migrations on the database
	@soda -c config/database.yml migrate up

.PHONY: db-rollback
db-rollback: ## Apply the 'down' migrations on the database
	@soda -c config/database.yml migrate down

.PHONY: platforms
platforms: ## Show available platforms
	@go tool dist list

.PHONY: clean
clean: ## Remove generated binary files
	@$(RM) -f $(BINS:%=bin/%$(BIN_SUFFIX))

.PHONY: distclean
distclean: clean ## Remove all generated files
	@$(RM) -f $(PROTO_GO_FILES)
	@$(RM) -f $(PROTO_GO_GRPC_FILES)
	@$(RM) -f $(PROTO_MOCK_FILES)
	@$(RM) -f $(MOCK_FILES)
	@$(RM) -rf $(PROTO_GO_DIR)
	@$(RM) -rf $(PROTO_MOCK_DIR)
	@$(RM) -rf $(MOCK_DIR)
	@$(RM) -rf bin

.PHONY: help
help: ## Show this help
	@egrep -h '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

