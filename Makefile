.PHONY: all

all:
	go build -trimpath -ldflags "-s -w" -o main main.go