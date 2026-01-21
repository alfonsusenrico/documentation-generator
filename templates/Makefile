SHELL := /bin/bash

.PHONY: install lint type test test-integration migrate-check build scan ci

install:
	{{INSTALL_CMD}}

lint:
	{{LINT_CMD}}

type:
	{{TYPE_CMD}}

test:
	{{TEST_CMD}}

test-integration:
	{{INTEGRATION_TEST_CMD}}

migrate-check:
	{{MIGRATE_CHECK_CMD}}

build:
	{{BUILD_CMD}}

scan:
	{{DEP_AUDIT_CMD}}

ci: lint type test build scan
