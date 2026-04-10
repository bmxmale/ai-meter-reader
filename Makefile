.PHONY: build clean

export PATH := $(PATH):/usr/local/go/bin:$(HOME)/go/bin

build:
	go build -o bin/ocr ./src/

clean:
	rm -f bin/ocr
