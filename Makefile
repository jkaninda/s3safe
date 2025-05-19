BINARY_NAME=s3safe
IMAGE_NAME=jkaninda/${BINARY_NAME}:latest
export

help:
	@echo "Makefile for s3safe"
	@echo "Available commands:"
	@echo "  run       - Run the application"
	@echo "  lint      - Run linters"
	@echo "  build     - Build the application"
	@echo "  compile   - Compile for different OS/Arch"
	@echo "  backup    - Backup files to S3"
	@echo "  restore   - Restore files from S3"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-backup - Backup files to S3 using Docker"
	@echo "  docker-restore - Restore files from S3 using Docker"
	@echo "  validate  - Validate the application"
	@echo "  help      - Show this help message"

.PHONY: help run lint build compile backup restore docker-build docker-backup docker-restore
run:
	go run .
lint:
	golangci-lint run
build:
	go build -ldflags="-X 'github.com/jkaninda/s3safe/utils.Version=1.0'" -o bin/${BINARY_NAME} .

compile:
	GOOS=darwin GOARCH=arm64 go build -o bin/${BINARY_NAME}-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY_NAME}-darwin-amd64 .
	GOOS=linux GOARCH=arm64 go build -o bin/${BINARY_NAME}-linux-arm64 .
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME}-linux-amd64 .

backup: build
	./bin/${BINARY_NAME} backup --path backups -d /s3path/backups --exclude readme.md -r

restore: build
	./bin/${BINARY_NAME} restore -d backups -p /s3path/backups  --exclude s3safe.txt -r --force

validate: build
	./bin/${BINARY_NAME} validate

docker-build:
	docker build --build-arg appVersion=latest -t ${IMAGE_NAME} .
docker-backup:
	docker run --rm --env-file .env --name s3safe -v "./backups:/backups" ${IMAGE_NAME} backup --path backups -d /s3path --compress #-t
docker-restore: docker-build
	docker run --rm --env-file .env --name s3safe -v "./backups:/backups" ${IMAGE_NAME} restore -d ./backups -p /s3path --decompress


