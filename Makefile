build:
	go build -o stash -ldflags "-X main.revision=$(shell git describe --tags --always)" ./app

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

STASH_JAVA_HOME ?= /Users/umputun/Library/Java/JavaVirtualMachines/corretto-17.0.14/Contents/Home

test-java-sdk:
	cd lib/stash-java && JAVA_HOME=$(STASH_JAVA_HOME) ./gradlew test --no-daemon --console=plain

test-all-sdks: test-python-sdk test-js-sdk test-java-sdk

.PHONY: build test lint docker run prep_site e2e-setup e2e e2e-ui test-python-sdk test-js-sdk test-java-sdk test-all-sdks
