
GO_TESTS := $(wildcard *_test.go)

GO_FILES := $(filter-out $(GO_TESTS), $(wildcard *.go))



lines: $(GO_FILES)
	GOOS=linux \
		go build -o $@ $?
	strip $@

lines.exe: $(GO_FILES)
	GOOS=windows \
		go build -o $@ $?
	strip $@



.PHONY: windows
windows: lines.exe

.PHONY: linux
linux: lines

.PHONY: test
test:
	go test



.PHONY: clean
clean:
	rm -f lines lines.exe

