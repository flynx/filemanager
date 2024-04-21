


%: %.go
	GOOS=linux \
	     go build -o $@ $<

%.exe: %.go
	GOOS=windows \
		go build -o $@ $<



windows: lines.exe

linux: lines



clean:
	rm -f lines lines.exe

