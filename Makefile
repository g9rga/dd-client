GOARCH = amd64
GOOS = linux

lint:
	golangci-lint run ./... --fix -E bodyclose,errorlint,gofmt,makezero,maligned,misspell,nilerr,prealloc,promlinter,rowserrcheck,scopelint,unconvert,wastedassign
build:
	CGO_ENABLED=0 go build -o bin/main main.go
	docker build -t g9rga/dd-client:latest --build-arg DD_API_URL=$(DD_API_URL) -f Dockerfile .
	docker push g9rga/dd-client:latest
build-dev:
	CGO_ENABLED=0 go build -o bin/main main.go
	docker build -t g9rga/dd-client-dev:latest --build-arg DD_API_URL=$(DD_API_URL) -f Dockerfile .
run:
	docker run --pull=always -it g9rga/dd-client
run-dev:
	docker run -it g9rga/dd-client-dev:latest