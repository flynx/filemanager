
GO_FILES := $(wildcard *.go)



%: %.go
	GOOS=linux \
		go build -o $@ $<

%.exe: %.go
	GOOS=windows \
		go build -o $@ $<



windows: lines.exe $(GO_FILES)

linux: lines $(GO_FILES)



clean:
	rm -f lines lines.exe

