UISOURCES := $(shell find . -type f -not \( -path ./build/\* -o -path ./node_modules/\* -o -path ./.cache/\* -o -name Makefile -prune \) )

all: build

node_modules:
	npm i

 e2e: node_modules
	npm run test:junit

build: node_modules $(UISOURCES)
	npm run build

lint: node_modules $(UISOURCES)
	npm run lint

test:
	npm test

clean:
	npm run clean

run:
	npm start

.PHONY: all clean test run lint junit
