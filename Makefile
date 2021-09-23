# Copyright 2012 tsuru authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

GOCMD=go
BUILD_DIR = build
C8Y_PKGS = $$(go list ./... | grep -v /vendor/)
GOMOD=$(GOCMD) mod

ENV_FILE ?= c8y.env
-include $(ENV_FILE)
export $(shell sed 's/=.*//' $(ENV_FILE) 2>/dev/null)

.PHONY: all check-path test race docs install tsurud

all: check-path test

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
      $(error Undefined $1$(if $2, ($2))))

# It does not support GOPATH with multiple paths.
check-path:
	ifndef GOPATH
		@echo "FATAL: you must declare GOPATH environment variable, for more"
		@echo "       details, please check"
		@echo "       http://golang.org/doc/code.html#GOPATH"
		@exit 1
	endif
	@exit 0

check-integration-variables:
	$(call check_defined, C8Y_HOST, Cumulocity host url. i.e. https://cumulocity.com)
	$(call check_defined, C8Y_TENANT , Cumulocity tenant)
	$(call check_defined, C8Y_USER, Cumulocity username)
	$(call check_defined, C8Y_PASSWORD, Cumulocity password)
	@exit 0

_go_test:
	$(MAKE) check-integration-variables
	$(MAKE) _go_integration_tests

install_test_deps:
	go mod download github.com/reubenmiller/go-c8y/test/c8y_test
	go mod download github.com/reubenmiller/go-c8y/test/microservice_test

# check_integration_configuration:
# 	$(call check_defined, C8Y_PASSWORD, applicaiton.properties file path)

_go_integration_tests:
	GO111MODULE=on go test -v -timeout 30m github.com/reubenmiller/go-c8y/test/c8y_test
	GO111MODULE=on go test -v -timeout 30m github.com/reubenmiller/go-c8y/test/microservice_test

test: _go_test

lint: metalint

install:
	go mod download

metalint:
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	golangci-lint run -c .golangci.yml

race:
	go test $(GO_EXTRAFLAGS) -race -i $(C8Y_PKGS)
	go test $(GO_EXTRAFLAGS) -race $(C8Y_PKGS)


docs:
	godoc -http=":6060"

update-vendor:
	GO111MODULE=on $(GOMOD) download
	GO111MODULE=on $(GOMOD) vendor
