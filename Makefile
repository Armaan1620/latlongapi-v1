.PHONY: run test build

run:
	go run .

test:
	@echo "Killing existing server if running..."
	@pkill -f "go run" || true
	@pkill -f "latlongapi-v1" || true
	@echo "Building..."
	@go build -o latlongapi-v1 .
	@echo "Starting server..."
	@./latlongapi-v1 &

build:
	go build -o latlongapi-v1 .

