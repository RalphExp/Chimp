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
	@echo testing ast ... && (cd ast && go test *);
	@echo testing lexer ... && (cd lexer && go test *);
	@echo testing parser ... && (cd parser && go test *);
	@echo testing object ... && (cd object && go test *);
	@echo testing interpreter ... && (cd evaluator && go test *);
	@echo testing code ... && (cd code && go test *);
	@echo testing vm ... && (cd vm && go test *);
	@echo testing compiler ... && (cd compiler && go test *);

benchmark:
	go run benchmark/main.go

clean:
	rm -rf chimp 
