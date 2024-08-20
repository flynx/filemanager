
GO_TESTS := $(wildcard *_test.go)

GO_FILES := $(filter-out $(GO_TESTS), $(wildcard *.go))


##%: %.go
##	GOOS=linux \
##		go build -o $@ $<
##
##%.exe: %.go
##	GOOS=windows \
##		go build -o $@ $<


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

