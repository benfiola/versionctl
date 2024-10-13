.PHONY: unit-test
unit-test:
	go test -v -count=1 ./internal/versionctl
