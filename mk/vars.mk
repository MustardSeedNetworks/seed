# vars.mk - Shared variables (single source of truth)
#
# Version comes from git tags: git tag v1.2.3
# All other version references derive from this.

# =============================================================================
# Version (from git tag - single source of truth)
# =============================================================================
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# =============================================================================
# Platform Detection
# =============================================================================
UNAME := $(shell uname -s)
ifeq ($(UNAME),Darwin)
    PLATFORM := darwin
else ifeq ($(UNAME),Linux)
    PLATFORM := linux
else
    PLATFORM := unknown
endif

ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
    GOARCH := amd64
else ifeq ($(ARCH),arm64)
    GOARCH := arm64
else ifeq ($(ARCH),aarch64)
    GOARCH := arm64
endif

# =============================================================================
# Project Info
# =============================================================================
PROJECT_NAME := seed
MODULE_PATH := github.com/krisarmstrong/seed
VERSION_PKG := $(MODULE_PATH)/internal/version

# =============================================================================
# Directories
# =============================================================================
BIN_DIR := bin
DIST_DIR := dist
UI_DIR := ui

# Output binary
BINARY := $(BIN_DIR)/$(PROJECT_NAME)

# =============================================================================
# ANSI Colors
# =============================================================================
BOLD := \033[1m
RESET := \033[0m
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
CYAN := \033[36m
