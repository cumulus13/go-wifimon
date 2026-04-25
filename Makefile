APP_NAME := wifimon
MAIN_PACKAGE := ./cmd/wifimon
OUTPUT := $(APP_NAME)

.PHONY: build build-windows run test tidy clean release-snapshot release-check

build:
	go build -buildvcs=false -ldflags="-s -w" -o $(OUTPUT) $(MAIN_PACKAGE)

build-windows:
	go build -buildvcs=false -ldflags="-s -w" -o $(OUTPUT).exe $(MAIN_PACKAGE)

run:
	go run $(MAIN_PACKAGE)

test:
	go test ./...

tidy:
	go mod tidy

clean:
	-del /q $(OUTPUT).exe 2>nul
	-del /q $(OUTPUT) 2>nul

release-snapshot:
	goreleaser release --snapshot --clean

release-check:
	goreleaser check
