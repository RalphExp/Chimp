all: chimp

chimp:
	go build -o $@ main.go

install:
	go install

clean:
	rm -rf chimp
