.PHONY: build clean

build:
	go build -o bin/ocr ./src/

clean:
	rm -f bin/ocr
