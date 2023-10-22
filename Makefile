.PHONY: all install test benchmark clean

all: chimp 

chimp:
	@echo building chimp ...
	@go build -o $@ main.go
	@echo done

install:
	@echo installing chimp ...
	@go install
	@echo done

test:
	@echo testing ast ... && go test ast/*
	@echo testing lexer ... && go test lexer/*
	@echo testing parser ... && go test parser/*
	@echo testing object ... && go test object/*
	@echo testing interpreter ... && go test evaluator/*
	@echo testing code ... && go test code/*
	@echo testing compiler ... && go test compiler/*
	@echo testing vm ... && go test vm/*

benchmark:
	go run benchmark/main.go

clean:
	rm -rf chimp 
