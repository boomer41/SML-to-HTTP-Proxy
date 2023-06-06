all: bin native linux-arm64 linux-amd64

.PHONY: clean native linux-arm64 linux-amd64

clean:
	-rm -Rf ./bin

bin:
	mkdir -p bin

native:
	CGO_ENABLED=0 go build -o bin/smlToHttp-native

linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/smlToHttp-linux-arm64

linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/smlToHttp-linux-amd64
