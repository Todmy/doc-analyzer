# doc-analyzer

Semantic document analysis CLI tool. Finds contradictions, detects anomalies, and provides topic statistics.

## Features

- **Contradictions** — Finds logically contradictory statements using embeddings + LLM analysis
- **Anomalies** — Detects outlier statements via clustering and distance metrics
- **Statistics** — Topic distribution, file coverage matrix, similarity histograms

## Installation

```bash
pip install -e .
# or
make install
```

## Setup

Set your OpenRouter API key:

```bash
export OPENROUTER_API_KEY="sk-or-..."
```

Or use the config command:

```bash
doc-analyzer config init
doc-analyzer config set openrouter.api_key "sk-or-..."
```

## Usage

### Full Analysis

```bash
doc-analyzer analyze ./docs
```

Options:
- `--threshold` / `-t` — Similarity threshold (default: 0.75)
- `--max-pairs` / `-m` — Max pairs to analyze (default: 100)
- `--output` / `-o` — Output file path
- `--format` / `-f` — Output format: markdown or json
- `--dry-run` — Skip LLM analysis
- `--no-contradictions` — Skip contradiction check
- `--no-anomalies` — Skip anomaly detection

### Individual Commands

```bash
# Find contradictions only
doc-analyzer contradictions ./docs

# Detect anomalies only
doc-analyzer anomalies ./docs

# Show statistics
doc-analyzer stats ./docs

# Show topic clusters
doc-analyzer clusters ./docs --samples
```

### Configuration

```bash
doc-analyzer config init    # Create config file
doc-analyzer config show    # View current config
doc-analyzer config test    # Test API connections
doc-analyzer config set KEY VALUE
```

### Cache Management

```bash
doc-analyzer cache-stats    # Show cache info
doc-analyzer cache-clear    # Clear embedding cache
```

## How It Works

```
Documents → Statements → Embeddings → Analysis → Report
                                         ↓
                           ┌─────────────┼─────────────┐
                           │             │             │
                     Contradictions  Anomalies   Statistics
```

1. **Parse** — Extract statements from .md and .json files
2. **Embed** — Generate vector embeddings via OpenRouter API
3. **Cluster** — Group statements by topic (K-means/HDBSCAN)
4. **Analyze**:
   - Find similar pairs → check for contradictions via Claude CLI
   - Calculate distance from centroids → detect anomalies
   - Compute statistics and coverage matrix

## Configuration File

Located at `~/.doc-analyzer/config.yaml`:

```yaml
openrouter:
  api_key: ${OPENROUTER_API_KEY}
  embedding_model: "openai/text-embedding-3-small"

analysis:
  similarity_threshold: 0.75
  max_pairs_to_analyze: 100
  skip_same_file: true
  min_statement_length: 50

claude:
  command: "claude"
  args: ["--print", "-p"]
  timeout: 30

output:
  format: "markdown"
  group_by: "severity"
```

## Example Output

```markdown
# Document Analysis Report

**Scanned:** 12 files, 847 statements
**Topics detected:** 8 clusters

## Statistics

| Cluster | Topic | Statements | Density |
|---------|-------|------------|---------|
| 1 | Growth & Metrics | 124 | 0.82 |
| 2 | Content Strategy | 189 | 0.76 |

## Anomalies (5)

### 1. Off-topic statement
- **File:** docs/config.md:342
- **Reason:** Far from cluster centroid

## Contradictions (3)

### Critical

#### 1. Numerical contradiction
- **File A:** overview.md:124
  > "0 → 1K followers in 3 months"
- **File B:** config.md:460
  > "0 → 2,500-3,500 followers in 90 days"
- **Explanation:** Same metric, different targets
```

## Development

```bash
make dev      # Install with dev deps
make test     # Run tests
make lint     # Check syntax
make clean    # Remove artifacts
```

## Requirements

- Python 3.11+
- OpenRouter API key (for embeddings)
- Claude CLI (for contradiction analysis)
