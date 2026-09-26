.PHONY: help
help:
	@cat Makefile | grep '# `' | grep -v '@cat Makefile'

# `make build`
.PHONY: build
build:
	bash build.sh

# `make test`
.PHONY: test
test:
	bash unit.test.coverage.sh

# `make clean`
.PHONY: clean
clean:
	rm -rf build/
