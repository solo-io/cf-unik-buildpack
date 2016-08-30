.PHONY: all

all:
	GOOS=linux go build -v -o bin/cf-unik-controller
