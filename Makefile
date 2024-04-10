


%: %.go
	GOOS=linux \
	     go build -o $@ $<

%.exe: %.go
	GOOS=windows \
		go build -o $@ $<



windows: fm.exe

linux: fm



clean:
	rm -f fm fm.exe

