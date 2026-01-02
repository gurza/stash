TAG=$(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
BRANCH=$(if $(TAG),$(TAG),$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null))
HASH=$(shell git rev-parse --short=7 HEAD 2>/dev/null)
TIMESTAMP=$(shell git log -1 --format=%ct HEAD 2>/dev/null | xargs -I{} date -u -r {} +%Y%m%dT%H%M%S)
GIT_REV=$(shell printf "%s-%s-%s" "$(BRANCH)" "$(HASH)" "$(TIMESTAMP)")
REV=$(if $(filter --,$(GIT_REV)),latest,$(GIT_REV))

STASH_JAVA_HOME ?= /Users/umputun/Library/Java/JavaVirtualMachines/corretto-17.0.14/Contents/Home

build:
	go build -o stash -ldflags "-X main.revision=$(REV) -s -w" ./app

test:
	go test -race -coverprofile=coverage.out -coverpkg=$$(go list ./... | grep -v /enum | tr '\n' ',' | sed 's/,$$//') ./...

lint:
	golangci-lint run

docker:
	docker build -t stash .

run:
	go run ./app --dbg server

prep_site:
	cp -fv README.md site/docs/index.md
	sed -i 's|^# Stash \[!\[.*$$|# Stash|' site/docs/index.md
	cd site && pip install -r requirements.txt && mkdocs build

e2e-setup:
	go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium

e2e:
	go test -v -failfast -count=1 -timeout=5m -tags=e2e ./e2e/...

e2e-ui:
	E2E_HEADLESS=false go test -v -failfast -count=1 -timeout=10m -tags=e2e ./e2e/...

test-python-sdk:
	cd lib/stash-python && uv sync --all-extras && uv run pytest

test-js-sdk:
	cd lib/stash-js && npm ci && npm test

test-java-sdk:
	cd lib/stash-java && JAVA_HOME=$(STASH_JAVA_HOME) ./gradlew test --no-daemon --console=plain

test-all-sdks: test-python-sdk test-js-sdk test-java-sdk

.PHONY: build test lint docker run prep_site e2e-setup e2e e2e-ui test-python-sdk test-js-sdk test-java-sdk test-all-sdks
