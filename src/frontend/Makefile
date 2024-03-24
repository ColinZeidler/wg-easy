PWD=$(shell pwd)

.PHONY: install
install:
	docker run --rm -v ${PWD}:/app -w /app node:20-alpine npm ci

.PHONY: update
update:
	docker run --rm -v ${PWD}:/app -w /app node:20-alpine npm install