
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



windows: lines.exe

linux: lines



clean:
	rm -f lines lines.exe

