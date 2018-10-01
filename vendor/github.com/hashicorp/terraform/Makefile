VERSION?="0.3.32"
TEST?=./...
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)
WEBSITE_REPO=github.com/hashicorp/terraform-website

default: test

tools:
	go get -u github.com/kardianos/govendor
	go get -u golang.org/x/tools/cmd/stringer
	go get -u golang.org/x/tools/cmd/cover

# bin generates the releaseable binaries for Terraform
bin: fmtcheck generate
	@TF_RELEASE=1 sh -c "'$(CURDIR)/scripts/build.sh'"

# dev creates binaries for testing Terraform locally. These are put
# into ./bin/ as well as $GOPATH/bin
dev: fmtcheck generate
	@TF_DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

quickdev: generate
	@TF_DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

# Shorthand for building and installing just one plugin for local testing.
# Run as (for example): make plugin-dev PLUGIN=provider-aws
plugin-dev: generate
	go install github.com/hashicorp/terraform/builtin/bins/$(PLUGIN)
	mv $(GOPATH)/bin/$(PLUGIN) $(GOPATH)/bin/terraform-$(PLUGIN)

# test runs the unit tests
# we run this one package at a time here because running the entire suite in
# one command creates memory usage issues when running in Travis-CI.
test: fmtcheck generate
	go list $(TEST) | xargs -t -n4 go test $(TESTARGS) -timeout=60s -parallel=4

# testacc runs acceptance tests
testacc: fmtcheck generate
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make testacc TEST=./builtin/providers/aws"; \
		exit 1; \
	fi
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m

# e2etest runs the end-to-end tests against a generated Terraform binary
# The TF_ACC here allows network access, but does not require any special
# credentials since the e2etests use local-only providers such as "null".
e2etest: generate
	TF_ACC=1 go test -v ./command/e2etest

test-compile: fmtcheck generate
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./builtin/providers/aws"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

# testrace runs the race checker
testrace: fmtcheck generate
	TF_ACC= go test -race $(TEST) $(TESTARGS)

cover:
	@go tool cover 2>/dev/null; if [ $$? -eq 3 ]; then \
		go get -u golang.org/x/tools/cmd/cover; \
	fi
	go test $(TEST) -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

# generate runs `go generate` to build the dynamically generated
# source files.
generate:
	@which stringer > /dev/null; if [ $$? -ne 0 ]; then \
	  go get -u golang.org/x/tools/cmd/stringer; \
	fi
	go generate ./...
	@go fmt command/internal_plugin_list.go > /dev/null

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

vendor-status:
	@govendor status

website:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), get-ting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	$(eval WEBSITE_PATH := $(GOPATH)/src/$(WEBSITE_REPO))
	@echo "==> Starting core website in Docker..."
	@docker run \
		--interactive \
		--rm \
		--tty \
		--publish "4567:4567" \
		--publish "35729:35729" \
		--volume "$(shell pwd)/website:/website" \
		--volume "$(shell pwd):/ext/terraform" \
		--volume "$(WEBSITE_PATH)/content:/terraform-website" \
		--volume "$(WEBSITE_PATH)/content/source/assets:/website/docs/assets" \
		--volume "$(WEBSITE_PATH)/content/source/layouts:/website/docs/layouts" \
		--workdir /terraform-website \
		hashicorp/middleman-hashicorp:${VERSION}

website-test:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), get-ting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	$(eval WEBSITE_PATH := $(GOPATH)/src/$(WEBSITE_REPO))
	@echo "==> Testing core website in Docker..."
	-@docker stop "tf-website-core-temp"
	@docker run \
		--detach \
		--rm \
		--name "tf-website-core-temp" \
		--publish "4567:4567" \
		--volume "$(shell pwd)/website:/website" \
		--volume "$(shell pwd):/ext/terraform" \
		--volume "$(WEBSITE_PATH)/content:/terraform-website" \
		--volume "$(WEBSITE_PATH)/content/source/assets:/website/docs/assets" \
		--volume "$(WEBSITE_PATH)/content/source/layouts:/website/docs/layouts" \
		--workdir /terraform-website \
		hashicorp/middleman-hashicorp:${VERSION}
	$(WEBSITE_PATH)/content/scripts/check-links.sh "http://127.0.0.1:4567" "/" "/docs/providers/*"
	@docker stop "tf-website-core-temp"

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:

.PHONY: bin cover default dev e2etest fmt fmtcheck generate plugin-dev quickdev test-compile test testacc testrace tools vendor-status website website-test
