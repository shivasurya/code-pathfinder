# Define version and commit variables
VERSION ?= $(shell cat VERSION)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)

.PHONY: all build run test lint clean version bump-version release tag-exists tag

# Default target
all: build

# Build the Go application with version information
build:
	go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT)" -o pathfinder .

# Run the Go application
run: build
	./pathfinder --project /Users/shiva/src/code-pathfinder/test-src/android --stdin true

# Run tests
test:
	go test ./...

# Lint the code
lint:
	golangci-lint run

# Clean the build output
clean:
	rm -f pathfinder

# Print version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Bump the version number
bump-version:
	@echo "Current version: $(VERSION)"
	@read -p "Enter new version: " new_version; \
	echo $$new_version > VERSION; \
	echo "Version bumped to $$new_version"

# Release a new version
release: bump-version
	@git add VERSION
	@git commit -m "Bump version to $(shell cat VERSION)"
	@git tag v$(shell cat VERSION)
	@git push origin main --tags
	@echo "New version finalized: $(shell cat VERSION)"

# Check if a Git tag exists
tag-exists:
	@if git rev-parse v$(VERSION) >/dev/null 2>&1; then \
		echo "Tag v$(VERSION) already exists."; \
		exit 1; \
	else \
		exit 0; \
	fi

# Release a new version
tag: tag-exists
	@git tag v$(VERSION)
	@git push origin tag v$(VERSION)
	@echo "New version tag published: v$(VERSION)"