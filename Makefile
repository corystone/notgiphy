all: bin/notgiphy

.PHONY: bin/notgiphy

bin/notgiphy:
	mkdir -p bin
	go build -o bin/notgiphy github.com/ston9665/notgiphy

