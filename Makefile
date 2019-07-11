.PHONY: build
build:
	@./scripts/ci build

.PHONY: nebula-%
nebula-%:
	@./scripts/ci build $@
