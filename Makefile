.PHONY: all cert

all: cert
	go build -trimpath -ldflags "-s -w" -o main main.go

cert:
	rm -rf certs && mkdir -p certs && touch certs/key.pem certs/cert.pem && go test -run ^TestGenerateCerts$