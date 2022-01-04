# Alertmanager-cli

`alertmanager-cli` is a cli writtin in golang to silence alerts in AlertManager

```
usage: alertmanager-cli [<flags>]

Flags:

--help                         Show context-sensitive help (also try --help-long and --help-man).
-m, --mode="show"              work mode: create/delete/show silence
--silence-period=2             default period for silenced alerts in hours
-l, --labels=LABELS            comma separated silence matching labels, eg. key1=value1,key2=value2
-c, --creator="auto-silencer"  creator of the silence
-C, --comment="auto-silencer"  comment attached to the silence. Recommended to add the jira ticket
-u, --URL="http://127.0.0.1"   Alertmanager URL
-t, --timeout=3                Alertmanager connection timeout
```
## Build the code

Using the makefile we can build, tag and push the cli to google registry

```
DOCKERFLAGS := --rm -v $(CURDIR):/app:rw -w /app
BUILD_DEV_IMAGE_PATH := golang:1.16.3-buster
IMAGE_PATH := snigdhasambit/travix-com/alertmanager-cli
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

```

## Create Alerts

```
./alertmanager-cli -u https://alertmanager.example.com -l alertname="DeprecatedAPIUsageWarning" -m create

Creating silence [creator: auto-silencer, comment: auto-silencer, start: 2022-01-03T17:10:50Z, end: 2022-01-03T19:10:50Z]
```
## Show Alerts

```
./alertmanager-cli -u https://alertmanager.example.com -l alertname="DeprecatedAPIUsageWarning" -m show

ID: bcba8dec-75ca-4546-8275-c6b2d0db70e6, creator: auto-silencer, comment: auto-silencer, start: 2022-01-03T17:10:50.678848412Z, end: 2022-01-03T19:10:50Z, labels: alertname=DeprecatedAPIUsageWarning
```

## Delete Alerts

```
./alertmanager-cli -u https://alertmanager.example.com -l alertname="DeprecatedAPIUsageWarning" -m delete

Deleting silence [creator: auto-silencer, comment: auto-silencer, start: 2022-01-03T17:10:50.678848412Z, end: 2022-01-03T19:10:50Z]
```