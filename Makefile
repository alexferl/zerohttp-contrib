.PHONY: dev audit cover cover-html fmt lint pre-commit test bench tidy update-deps

.DEFAULT: help
help:
	@echo "make dev"
	@echo "    setup development environment"
	@echo "make audit"
	@echo "    conduct quality checks"
	@echo "make cover"
	@echo "    generate coverage report"
	@echo "make cover-html"
	@echo "    generate coverage HTML report"
	@echo "make fmt"
	@echo "    fix code format issues"
	@echo "make lint"
	@echo "    run lint checks"
	@echo "make pre-commit"
	@echo "    run pre-commit hooks"
	@echo "make test"
	@echo "    run all tests"
	@echo "make bench"
	@echo "    run all benchmarks"
	@echo "make tidy"
	@echo "    clean and tidy dependencies"
	@echo "make update-deps"
	@echo "    update dependencies"

# gotestsum: pattern is passed per-module
GOTESTSUM := go run gotest.tools/gotestsum@latest -f testname -- -race -count=1
TESTFLAGS := -shuffle=on
COVERFLAGS := -covermode=atomic
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3

# All module directories (subdirs that contain go.mod), excluding the repo root and examples
MODULE_DIRS := $(shell find . -mindepth 2 -name go.mod -not -path './examples/*' -print0 | xargs -0 -n1 dirname)

define run_in_modules
	for m in $(MODULE_DIRS); do \
		echo "==> $$m"; \
		( cd $$m && $(1) ) || exit $$?; \
	done
endef

check-pre-commit:
ifeq (, $(shell which pre-commit))
	$(error "pre-commit not in $(PATH), pre-commit (https://pre-commit.com) is required")
endif

dev: check-pre-commit
	pre-commit install

audit:
	go mod verify
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

cover:
	$(call run_in_modules,$(GOTESTSUM) $(TESTFLAGS) $(COVERFLAGS) ./...)

cover-html:
	$(call run_in_modules,$(GOTESTSUM) $(TESTFLAGS) $(COVERFLAGS) -coverprofile=coverage.out ./...)
	# merge coverage.out files
	@echo "mode: atomic" > coverage.out
	@find . -name coverage.out -print0 | xargs -0 -I{} tail -n +2 {} >> coverage.out
	go tool cover -html=coverage.out

fmt:
	$(call run_in_modules, $(GOLANGCI_LINT) fmt)

lint:
	$(call run_in_modules,$(GOLANGCI_LINT) run)

pre-commit: check-pre-commit
	pre-commit run --all-files

test:
	$(call run_in_modules,$(GOTESTSUM) $(TESTFLAGS) ./...)

bench:
	$(call run_in_modules,go test -bench=. -benchmem -run=^$$ ./...)

tidy:
	$(call run_in_modules,go mod tidy -v)

update-deps: tidy
	$(call run_in_modules,go get -u ./...)
