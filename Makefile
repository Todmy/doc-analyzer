.PHONY: install dev test lint clean analyze help build build-clean

# Load .env file (ignore if doesn't exist for build-only scenarios)
-include .env
export

# Config file path
CONFIG = --config ./config.yaml

# Default target
help:
	@echo "doc-analyzer - Semantic document analysis CLI"
	@echo ""
	@echo "Setup:"
	@echo "  make install     Install package"
	@echo "  make dev         Install with dev dependencies"
	@echo ""
	@echo "Development:"
	@echo "  make test        Run tests"
	@echo "  make lint        Run linter"
	@echo "  make clean       Remove build artifacts"
	@echo ""
	@echo "Build:"
	@echo "  make build       Build portable executable (current platform)"
	@echo "  make build-clean Clean build artifacts"
	@echo ""
	@echo "Usage:"
	@echo "  make analyze     Run full analysis on ../../docs"
	@echo "  make stats       Show statistics only"
	@echo "  make clusters    Show topic clusters"
	@echo "  make anomalies   Detect anomalies"
	@echo "  make config      Show current config"

# Setup
install:
	pip install -e .

dev:
	pip install -e ".[dev]"

# Development
test:
	pytest tests/ -v

lint:
	python -m py_compile src/doc_analyzer/*.py

clean:
	rm -rf build/ dist/ *.egg-info src/*.egg-info
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete

# Analysis commands (default: analyze ../../docs)
DOCS_PATH ?= ../../docs

analyze:
	doc-analyzer analyze $(DOCS_PATH) $(CONFIG) --verbose

stats:
	doc-analyzer stats $(DOCS_PATH) $(CONFIG)

clusters:
	doc-analyzer clusters $(DOCS_PATH) $(CONFIG) --samples

anomalies:
	doc-analyzer anomalies $(DOCS_PATH) $(CONFIG)

contradictions:
	doc-analyzer contradictions $(DOCS_PATH) $(CONFIG)

# Config
config:
	doc-analyzer config show $(CONFIG)

config-test:
	doc-analyzer config test $(CONFIG)

# Cache
cache-stats:
	doc-analyzer cache-stats

cache-clear:
	doc-analyzer cache-clear

# Build portable executable
# Note: Builds for current platform only. For cross-platform, build on each OS.
build:
	@echo "Building portable executable..."
	@echo "Config will be bundled from: ./config.yaml"
	@echo ""
	@if [ ! -f config.yaml ]; then echo "ERROR: config.yaml not found"; exit 1; fi
	pyinstaller doc_analyzer.spec --clean --noconfirm
	@echo ""
	@echo "Build complete! Executable at: dist/doc-analyzer"
	@echo ""
	@echo "Usage: ./dist/doc-analyzer /path/to/documents"
	@ls -lh dist/doc-analyzer* 2>/dev/null || true

build-clean:
	rm -rf build/ dist/ *.spec.bak
	find . -name "*.pyc" -delete
	find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
