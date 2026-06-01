SERVICES := auth planet fleet combat espionage research nebula alliance chat event notification quest radar friend ranking gateway
MIGRATE_CMD := migrate -path ./migrations -database

.PHONY: all build test vet lint clean migrate-up migrate-down $(addprefix build-, $(SERVICES)) $(addprefix test-, $(SERVICES)) $(addprefix vet-, $(SERVICES)) $(addprefix migrate-up-, $(SERVICES)) $(addprefix migrate-down-, $(SERVICES))

all: vet build

# Build all services
build: $(addprefix build-, $(SERVICES))

$(addprefix build-, $(SERVICES)):
	@echo "Building $@..."
	@go build -o /dev/null ./cmd/$(subst build-,,$@)

# Test all services
test: $(addprefix test-, $(SERVICES))

$(addprefix test-, $(SERVICES)):
	@echo "Testing $@..."
	@go test ./cmd/$(subst test-,,$@)/...

# Vet all services
vet: $(addprefix vet-, $(SERVICES))

$(addprefix vet-, $(SERVICES)):
	@echo "Vetting $@..."
	@go vet ./cmd/$(subst vet-,,$@)/...

# Lint all services (requires golangci-lint)
lint:
	@golangci-lint run ./cmd/...

# Run all migrations up
migrate-up: $(addprefix migrate-up-, $(filter-out gateway, $(SERVICES)))

# Run all migrations down
migrate-down: $(addprefix migrate-down-, $(filter-out gateway, $(SERVICES)))

$(addprefix migrate-up-, $(SERVICES)):
	@echo "Running migrations up for $(subst migrate-up-,,$@)..."
	@cd cmd/$(subst migrate-up-,,$@) && $(MIGRATE_CMD) "postgres://galaxy:galaxy_dev@localhost:5432/galaxy_empire?sslmode=disable" up

$(addprefix migrate-down-, $(SERVICES)):
	@echo "Running migrations down for $(subst migrate-down-,,$@)..."
	@cd cmd/$(subst migrate-down-,,$@) && $(MIGRATE_CMD) "postgres://galaxy:galaxy_dev@localhost:5432/galaxy_empire?sslmode=disable" down

# Clean build artifacts
clean:
	@rm -f cmd/*/gateway cmd/*/auth cmd/*/planet cmd/*/fleet cmd/*/combat cmd/*/espionage cmd/*/research cmd/*/nebula cmd/*/alliance cmd/*/chat cmd/*/event cmd/*/notification cmd/*/quest cmd/*/radar cmd/*/friend cmd/*/ranking
	@echo "Cleaned build artifacts"

# Tidy go modules
tidy:
	@for dir in cmd/*/; do \
		(cd "$$dir" && go mod tidy); \
	done

# Docker compose helpers
up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f
