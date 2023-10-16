# PROVIDER_DIR is used instead of PWD since docker volume commands can be dangerous to run in $HOME.
# This ensures docker volumes are mounted from within provider directory instead.
PROVIDER_DIR := $(abspath $(lastword $(dir $(MAKEFILE_LIST))))
TEST         := "$(PROVIDER_DIR)/kubernetes"
GOFMT_FILES  := $$(find $(PROVIDER_DIR) -name '*.go' |grep -v vendor)
WEBSITE_REPO := github.com/hashicorp/terraform-website
PKG_NAME     := kubernetes
OS_ARCH      := $(shell go env GOOS)_$(shell go env GOARCH)
TF_PROV_DOCS := $(PWD)/kubernetes/test-infra/tfproviderdocs

ifneq ($(PWD),$(PROVIDER_DIR))
$(error "Makefile must be run from the provider directory")
endif

# For changelog generation, default the last release to the last tag on
# any branch, and this release to just be the current branch we're on.
LAST_RELEASE?=$$(git describe --tags $$(git rev-list --tags --max-count=1))
THIS_RELEASE?=$$(git rev-parse --abbrev-ref HEAD)

# The maximum number of tests to run simultaneously.
PARALLEL_RUNS?=8

default: build

all: build depscheck fmtcheck test testacc test-compile tests-lint tests-lint-fix tools vet website-lint website-lint-fix

build: fmtcheck
	go install

# expected to be invoked by make changelog LAST_RELEASE=gitref THIS_RELEASE=gitref
changelog:
	@echo "Generating changelog for $(THIS_RELEASE) from $(LAST_RELEASE)..."
	@echo
	@changelog-build -last-release $(LAST_RELEASE) \
		-entries-dir .changelog/ \
		-changelog-template .changelog/changelog.tmpl \
		-note-template .changelog/note.tmpl \
		-this-release $(THIS_RELEASE)

changelog-entry:
	@changelog-entry -dir .changelog/

depscheck:
	@echo "==> Checking source code with 'git diff'..."
	@git diff --check || exit 1
	@echo "==> Checking source code with go mod tidy..."
	@go mod tidy
	@git diff --exit-code -- go.mod go.sum || \
		(echo; echo "Unexpected difference in go.mod/go.sum files. Run 'go mod tidy' command or revert any go.mod/go.sum changes and commit."; exit 1)
	@echo "==> Checking source code with go mod vendor..."
	@go mod vendor
	@git diff --exit-code -- vendor || \
		(echo; echo "Unexpected difference in vendor/ directory. Run 'go mod vendor' command or revert any go.mod/go.sum/vendor changes and commit."; exit 1)

examples-lint: tools
	@echo "==> Checking _examples dir formatting..."
	@./scripts/fmt-examples.sh || (echo; \
		echo "Terraform formatting errors found in _examples dir."; \
		echo "To see the full differences, run: ./scripts/fmt-examples.sh diff"; \
		echo "To automatically fix the formatting, run 'make examples-lint-fix' and commit the changes."; \
		exit 1)

examples-lint-fix: tools
	@echo "==> Fixing terraform formatting of _examples dir..."
	@./scripts/fmt-examples.sh fix

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@./scripts/gofmtcheck.sh

errcheck:
	@./scripts/errcheck.sh
	

test: fmtcheck
	go test $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc: fmtcheck vet
	TF_ACC=1 go test $(TEST) -v -vet=off $(TESTARGS) -parallel $(PARALLEL_RUNS) -timeout 3h

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

tests-lint: tools
	@echo "==> Checking acceptance test terraform blocks code with terrafmt..."
	@terrafmt diff -f ./kubernetes --check --pattern '*_test.go' --quiet || (echo; \
		echo "Unexpected differences in acceptance test HCL formatting."; \
		echo "To see the full differences, run: terrafmt diff ./kubernetes --pattern '*_test.go'"; \
		echo "To automatically fix the formatting, run 'make tests-lint-fix' and commit the changes."; \
		exit 1)

tests-lint-fix: tools
	@echo "==> Fixing acceptance test terraform blocks code with terrafmt..."
	@find ./kubernetes -name "*_test.go" -exec sed -i ':a;N;$$!ba;s/fmt.Sprintf(`\n/fmt.Sprintf(`/g' '{}' \; # remove newlines for terrafmt
	@terrafmt fmt -f ./kubernetes --pattern '*_test.go'

tools:
	go install github.com/client9/misspell/cmd/misspell@v0.3.4
	go install github.com/bflad/tfproviderlint/cmd/tfproviderlint@v0.28.1
	go install github.com/bflad/tfproviderdocs@v0.9.1
	go install github.com/katbyte/terrafmt@v0.5.2
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.0
	go install github.com/hashicorp/go-changelog/cmd/changelog-build@latest
	go install github.com/hashicorp/go-changelog/cmd/changelog-entry@latest

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

# The docker command and run options may be overridden using env variables DOCKER and DOCKER_RUN_OPTS.
# Example:
#   DOCKER="podman --cgroup-manager=cgroupfs" make website-lint
#   DOCKER_RUN_OPTS="--userns=keep-id" make website-lint
#   This option is needed for systems using SELinux and rootless containers.
#   DOCKER_VOLUME_OPTS="rw,Z"
# For more info, see https://docs.docker.com/storage/bind-mounts/#configure-the-selinux-label
DOCKER?=$(shell which docker)
ifeq ($(strip $(DOCKER)),)
$(error "Docker binary could not be found in PATH. Please install docker, or specify an alternative by setting DOCKER=/path/to/binary")
endif
DOCKER_VOLUME_OPTS?="rw"
DOCKER_SELINUX := $(shell which setenforce)
ifeq ($(.SHELLSTATUS),0)
DOCKER_VOLUME_OPTS="rw,Z"
endif

website-lint: tools
	@echo "==> Checking website against linters..."
	@misspell -error -source=text ./website || (echo; \
		echo "Unexpected mispelling found in website files."; \
		echo "To automatically fix the misspelling, run 'make website-lint-fix' and commit the changes."; \
		exit 1)
	@echo "==> Running markdownlint-cli using DOCKER='$(DOCKER)', DOCKER_RUN_OPTS='$(DOCKER_RUN_OPTS)' and DOCKER_VOLUME_OPTS='$(DOCKER_VOLUME_OPTS)'"
	@$(DOCKER) run $(DOCKER_RUN_OPTS) -v $(PROVIDER_DIR):/workspace:$(DOCKER_VOLUME_OPTS) -w /workspace 06kellyjac/markdownlint-cli ./website || (echo; \
		echo "Unexpected issues found in website Markdown files."; \
		echo "To apply any automatic fixes, run 'make website-lint-fix' and commit the changes."; \
		exit 1)
	@echo "==> Running terrafmt diff..."
	@terrafmt diff ./website --check --pattern '*.markdown' --quiet || (echo; \
		echo "Unexpected differences in website HCL formatting."; \
		echo "To see the full differences, run: terrafmt diff ./website --pattern '*.markdown'"; \
		echo "To automatically fix the formatting, run 'make website-lint-fix' and commit the changes."; \
		exit 1)
	@echo "==> Statically compiling provider for tfproviderdocs..."
	@env CGO_ENABLED=0 GOOS=$$(go env GOOS) GOARCH=$$(go env GOARCH) go build -a -o $(TF_PROV_DOCS)/terraform-provider-kubernetes
	@echo "==> Getting provider schema for tfproviderdocs..."
		@$(DOCKER) run $(DOCKER_RUN_OPTS) -v $(TF_PROV_DOCS):/workspace:$(DOCKER_VOLUME_OPTS) -w /workspace hashicorp/terraform:0.12.29 init
		@$(DOCKER) run $(DOCKER_RUN_OPTS) -v $(TF_PROV_DOCS):/workspace:$(DOCKER_VOLUME_OPTS) -w /workspace hashicorp/terraform:0.12.29 providers schema -json > $(TF_PROV_DOCS)/schema.json
	@echo "==> Running tfproviderdocs..."
	@tfproviderdocs check -providers-schema-json $(TF_PROV_DOCS)/schema.json -provider-name kubernetes
	@rm -f $(TF_PROV_DOCS)/schema.json $(TF_PROV_DOCS)/terraform-provider-kubernetes
	@echo "==> Checking for broken links..."
	@scripts/markdown-link-check.sh "$(DOCKER)" "$(DOCKER_RUN_OPTS)" "$(DOCKER_VOLUME_OPTS)" "$(PROVIDER_DIR)"

website-lint-fix: tools
	@echo "==> Applying automatic website linter fixes..."
	@misspell -w -source=text ./website
	@echo "==> Running markdownlint-cli --fix using DOCKER='$(DOCKER)', DOCKER_RUN_OPTS='$(DOCKER_RUN_OPTS)' and DOCKER_VOLUME_OPTS='$(DOCKER_VOLUME_OPTS)'"
	@$(DOCKER) run $(DOCKER_RUN_OPTS) -v $(PROVIDER_DIR):/workspace:$(DOCKER_VOLUME_OPTS) -w /workspace 06kellyjac/markdownlint-cli --fix ./website
	@echo "==> Fixing website terraform blocks code with terrafmt..."
	@terrafmt fmt ./website --pattern '*.markdown'

.PHONY: build test testacc tools vet fmt fmtcheck terrafmt test-compile depscheck tests-lint tests-lint-fix website-lint website-lint-fix changelog changelog-entry
