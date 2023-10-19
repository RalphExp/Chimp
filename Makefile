all: chimp

chimp:
	go build -o $@ main.go

clean:
	rm -rf chimp
