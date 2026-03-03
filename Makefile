swag:
	swag init

run:
	go run main.go

web-build:
	cd web && npm run build

build:
	go build -o bin/k8s-manage main.go

build-all: web-build build

docker:
	docker buildx build -t k8s-manage .

migrate-up:
	go run main.go migrate up

migrate-status:
	go run main.go migrate status

migrate-down:
	go run main.go migrate down --steps=1

# Testing commands
test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-check:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$coverage%"; \
	if [ $$(echo "$$coverage < 40" | bc -l) -eq 1 ]; then \
		echo "Error: Coverage $$coverage% is below threshold 40%"; \
		exit 1; \
	fi

web-test:
	cd web && npm run test:run

web-test-coverage:
	cd web && npm run test:run -- --coverage

test-all: test web-test
	@echo "All tests passed!"

# Performance testing commands
perf-test:
	@echo "Running k6 performance tests..."
	k6 run e2e/performance/api_benchmark_test.js

perf-test-smoke:
	@echo "Running k6 smoke test..."
	k6 run --scenario smoke e2e/performance/cluster_import_test.js

perf-test-load:
	@echo "Running k6 load test..."
	k6 run --scenario load e2e/performance/cluster_import_test.js

perf-test-stress:
	@echo "Running k6 stress test..."
	k6 run --scenario stress e2e/performance/cluster_import_test.js

# Run specific package tests
test-cluster:
	go test ./internal/service/cluster/... -v

test-deployment:
	go test ./internal/service/deployment/... -v

test-ai:
	go test ./internal/service/ai/... -v

test-notification:
	go test ./internal/service/notification/... -v

