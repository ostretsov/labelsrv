VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
CREATED  ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
IMAGE    ?= labelsrv
LDFLAGS  := -s -w -X github.com/ostretsov/labelsrv/internal/version.Version=$(VERSION)

DOCKER_LINT := docker run --rm -t -v $(shell pwd):/app -w /app \
	-v $(shell go env GOCACHE):/tmp/.cache/go-build -e GOCACHE=/tmp/.cache/go-build \
	-v $(shell go env GOMODCACHE):/tmp/.cache/mod -e GOMODCACHE=/tmp/.cache/mod \
	-v ~/.cache/golangci-lint:/tmp/.cache/golangci-lint -e GOLANGCI_LINT_CACHE=/tmp/.cache/golangci-lint \
	golangci/golangci-lint:v2.6.1

.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o labelsrv ./cmd/labelsrv

.PHONY: docker-build
docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg CREATED=$(CREATED) \
		-t $(IMAGE):$(VERSION) \
		-t $(IMAGE):latest \
		.

.PHONY: docker-push
docker-push:
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest

.PHONY: retag
retag:
	git tag -d $(VERSION) 2>/dev/null || true
	git push origin :refs/tags/$(VERSION) 2>/dev/null || true
	git tag $(VERSION)
	git push origin $(VERSION)

.PHONY: lint
lint:
	$(DOCKER_LINT) golangci-lint run --build-tags=test
