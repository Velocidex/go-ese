all:
	go build -o eseparser bin/*.go


windows:
	GOOS=windows GOARCH=amd64 \
            go build \
	    -o eseparser.exe ./bin/*.go

generate:
	cd parser/ && binparsegen conversion.spec.yaml > ese_gen.go


test:
	go test ./...
