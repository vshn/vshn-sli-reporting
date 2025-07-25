## These are some common variables for Make

PROJECT_ROOT_DIR = .
PROJECT_NAME ?= vshn-sli-reporting
PROJECT_OWNER ?= vshn

## BUILD:go
BIN_FILENAME ?= $(PROJECT_NAME)

## BUILD:docker
DOCKER_CMD ?= docker

IMG_TAG ?= latest
# Image URL to use all building/pushing image targets
CONTAINER_IMG ?= local.dev/$(PROJECT_OWNER)/$(PROJECT_NAME):$(IMG_TAG)

PROMETHEUS_VERSION ?= 2.40.7
PROMETHEUS_DIST ?= $(shell go env GOOS)
PROMETHEUS_ARCH ?= $(shell go env GOARCH)
PROMETHEUS_DOWNLOAD_LINK ?= https://github.com/prometheus/prometheus/releases/download/v$(PROMETHEUS_VERSION)/prometheus-$(PROMETHEUS_VERSION).$(PROMETHEUS_DIST)-$(PROMETHEUS_ARCH).tar.gz
