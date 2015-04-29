all:
	go build

clean:
	rm -f philrs232

indent:
	gofmt -w main.go
