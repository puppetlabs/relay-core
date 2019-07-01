
.PHONY: nebula-slack-notification run-nebula-slack-notification


nebula-slack-notification:
	@cmd/nebula-slack-notification/build

run-nebula-slack-notification:
	@cmd/nebula-slack-notification/run
