
test:
	@go test -race -covermode=atomic -coverprofile=coverage.txt ./...
	@go tool cover -html coverage.txt -o cover.html

