DOCKERFLAGS := --rm -v $(CURDIR):/app:rw -w /app
BUILD_DEV_IMAGE_PATH := golang:1.16.3-buster
IMAGE_PATH := snigdhasambit/alertmanager-cli
REF_NAME ?= $(shell git rev-parse --abbrev-ref HEAD)
export IMAGE_VERSION ?= ${REF_NAME}-$(shell git rev-parse HEAD)

.PHONY: clean
clean:
	rm -rf ./bin/*

.PHONY: build
build: clean
	GOOS=linux go build -a -o ./bin/alertmanager-cli

.PHONY: build-image
build-image:
	docker run $(DOCKERFLAGS) $(BUILD_DEV_IMAGE_PATH) make build
	docker build -t $(IMAGE_PATH):$(IMAGE_VERSION) .

.PHONY: tag-image
tag-image:
	docker tag $(IMAGE_PATH):$(IMAGE_VERSION) $(IMAGE_PATH):$(IMAGE_VERSION)

.PHONY: push-image
push-image:
	docker push $(IMAGE_PATH):$(IMAGE_VERSION)

.PHONY: deploy
deploy:
	export IMAGE_VERSION=$(IMAGE_VERSION)
	envsubst < deploy/deploy.yml | kubectl apply -f -

.PHONY: docker-lint
docker-lint:
	docker run $(DOCKERFLAGS) golangci/golangci-lint:v1.39.0 golangci-lint run -v

.PHONY: docker-test
docker-test:
	docker run $(DOCKERFLAGS) $(BUILD_DEV_IMAGE_PATH) make test
