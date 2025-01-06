.PHONY: ci gomod init build test dev lint build-docker fly-sqlite3 clean sec cross-compile migratetest

GOTOOLCHAIN=auto

init: gomod custom-gcl
	@echo "Dependencies installed"

gomod:
	@echo "Installing Go dependencies..."
	go version
	go mod download
	go mod verify

custom-gcl:
	@echo "Installing golangci-lint and building custom version for nilaway plugin to ./custom-gcl"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.63.4
	bin/golangci-lint custom

sec:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck -show verbose ./...

ci: init build lint test sec

build:
	@echo "Building..."
	go build -o bin/sheerluck github.com/myrjola/sheerluck/cmd/web
	go build -o bin/smoketest github.com/myrjola/sheerluck/cmd/smoketest
	go build -o bin/migratetest github.com/myrjola/sheerluck/cmd/migratetest

test:
	@echo "Running tests..."
	go test --race ./...

lint:
	@echo "Running linter..."
	./custom-gcl run

dev:
	@echo "Running dev server with debug build..."
	go build -gcflags="all=-N -l" -o bin/sheerluck github.com/myrjola/sheerluck/cmd/web
	./bin/sheerluck

cross-compile:
	@echo "Cross-compiling..."
	docker build --tag sheerluck-bin --file cross-compile.Dockerfile .
	docker create --name sheerluck-bin-extract sheerluck-bin
	docker cp sheerluck-bin-extract:/dist/sheerluck.linux_amd64 ./bin/sheerluck.linux_amd64
	docker rm sheerluck-bin-extract

build-docker:
	@echo "Building Docker image..."
	docker build --tag sheerluck .

fly-sqlite3:
	@echo "Connecting to sqlite3 database on deployed Fly machine"
	fly ssh console --pty --user sheerluck -C "/usr/bin/sqlite3 /data/sheerluck.sqlite3"

clean:
	@echo "Cleaning up..."
	rm -rf bin
	rm -rf custom-gcl

migratetest: build
	@echo "Deleting previous restored backup..."
	rm -rf restored.sqlite3* .restored.sqlite3-litestream/
	@echo "Restoring database from backup..."
	litestream restore --config litestream.yml restored.sqlite3
	@echo "Running migration test..."
	bin/migratetest
