all: build
.PHONY: all

build:
	go build -o release-resource-diff
.PHONY: build
	
clean:
	rm -f release-resource-diff
.PHONY: clean
