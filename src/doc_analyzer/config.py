"""Configuration management with priority: CLI > config file > env > defaults."""

import os
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml
from dotenv import load_dotenv

# Load .env file if present
load_dotenv()


def is_frozen() -> bool:
    """Check if running as a PyInstaller frozen executable."""
    return getattr(sys, 'frozen', False) and hasattr(sys, '_MEIPASS')


def get_bundle_dir() -> Path:
    """Get the directory where bundled files are extracted."""
    if is_frozen():
        return Path(sys._MEIPASS)
    return Path(__file__).parent.parent.parent  # src/../.. = project root

DEFAULT_CONFIG_DIR = Path.home() / ".doc-analyzer"
DEFAULT_CONFIG_FILE = DEFAULT_CONFIG_DIR / "config.yaml"

DEFAULT_PROMPT_TEMPLATE = """Analyze these two statements for logical contradiction:

STATEMENT A (from {file_a}:{line_a}):
"{text_a}"

STATEMENT B (from {file_b}:{line_b}):
"{text_b}"

Consider:
- Direct contradictions (X vs not-X)
- Numerical conflicts (different values for same metric)
- Temporal conflicts (different timelines)
- Implicit conflicts (assumptions that can't both be true)

Respond in JSON:
{{
  "contradiction": true/false,
  "confidence": 0.0-1.0,
  "type": "direct|numerical|temporal|implicit|none",
  "severity": "critical|high|medium|low",
  "explanation": "brief explanation"
}}"""


@dataclass
class OpenRouterConfig:
    """OpenRouter API configuration."""
    api_key: str = ""
    embedding_model: str = "openai/text-embedding-3-small"
    base_url: str = "https://openrouter.ai/api/v1"


@dataclass
class AnomalyConfig:
    """Anomaly detection settings."""
    method: str = "ensemble"  # isolation_forest, lof, hdbscan, ensemble
    contamination: float = 0.05  # Expected proportion of anomalies (1-10%)
    ensemble_weights: list[float] = field(default_factory=lambda: [0.4, 0.4, 0.2])  # IF, LOF, HDBSCAN
    min_methods_agree: int = 2  # Minimum methods that must flag for ensemble
    lof_neighbors: int = 20  # Number of neighbors for LOF
    isolation_forest_estimators: int = 100  # Number of trees


@dataclass
class AnalysisConfig:
    """Analysis settings."""
    similarity_threshold: float = 0.75
    max_pairs_to_analyze: int = 100
    skip_same_file: bool = True
    min_statement_length: int = 50
    anomaly_percentile: float = 95.0  # Legacy, kept for backward compatibility


@dataclass
class ClaudeConfig:
    """Claude CLI settings."""
    command: str = "claude"
    args: list[str] = field(default_factory=lambda: ["--print", "-p"])
    timeout: int = 30


@dataclass
class OutputConfig:
    """Output settings."""
    format: str = "markdown"  # markdown | json | html
    group_by: str = "severity"  # severity | file | type
    include_non_contradictions: bool = False


@dataclass
class DocumentsConfig:
    """Documents to analyze."""
    path: str = "./docs"
    extensions: list[str] = field(default_factory=lambda: [".md", ".txt", ".json"])


@dataclass
class Config:
    """Main configuration container."""
    documents: DocumentsConfig = field(default_factory=DocumentsConfig)
    openrouter: OpenRouterConfig = field(default_factory=OpenRouterConfig)
    analysis: AnalysisConfig = field(default_factory=AnalysisConfig)
    anomaly: AnomalyConfig = field(default_factory=AnomalyConfig)
    claude: ClaudeConfig = field(default_factory=ClaudeConfig)
    output: OutputConfig = field(default_factory=OutputConfig)
    prompt_template: str = DEFAULT_PROMPT_TEMPLATE

    @classmethod
    def load(cls, config_path: Path | None = None) -> "Config":
        """Load config from file, env vars, and defaults.

        Priority: config file > env vars > defaults
        """
        config = cls()

        # 1. Load from config file if exists
        file_path = config_path or _get_config_file_path()
        if file_path and file_path.exists():
            config = _merge_from_file(config, file_path)

        # 2. Override with env vars
        config = _merge_from_env(config)

        return config

    def save(self, config_path: Path | None = None) -> None:
        """Save config to file."""
        file_path = config_path or DEFAULT_CONFIG_FILE
        file_path.parent.mkdir(parents=True, exist_ok=True)

        data = _config_to_dict(self)
        with open(file_path, "w") as f:
            yaml.dump(data, f, default_flow_style=False, sort_keys=False)

    def get(self, key: str) -> Any:
        """Get config value by dot-separated key."""
        parts = key.split(".")
        obj: Any = self
        for part in parts:
            if hasattr(obj, part):
                obj = getattr(obj, part)
            elif isinstance(obj, dict) and part in obj:
                obj = obj[part]
            else:
                raise KeyError(f"Config key not found: {key}")
        return obj

    def set(self, key: str, value: Any) -> None:
        """Set config value by dot-separated key."""
        parts = key.split(".")
        obj: Any = self
        for part in parts[:-1]:
            obj = getattr(obj, part)
        setattr(obj, parts[-1], value)


def _get_config_file_path() -> Path | None:
    """Get config file path from env or default location.

    Priority:
    1. DOC_ANALYZER_CONFIG env var
    2. ~/.doc-analyzer/config.yaml (user config - highest priority for saved settings)
    3. Bundled config (for frozen executables - fallback defaults)
    """
    env_path = os.getenv("DOC_ANALYZER_CONFIG")
    if env_path:
        return Path(env_path)

    # User config takes priority (where API key gets saved)
    if DEFAULT_CONFIG_FILE.exists():
        return DEFAULT_CONFIG_FILE

    # Fall back to bundled config (PyInstaller frozen executable)
    if is_frozen():
        bundled_config = get_bundle_dir() / "config.yaml"
        if bundled_config.exists():
            return bundled_config

    return None


def _merge_from_file(config: Config, file_path: Path) -> Config:
    """Merge config from YAML file."""
    with open(file_path) as f:
        data = yaml.safe_load(f) or {}

    # Documents
    if "documents" in data:
        doc_data = data["documents"]
        if "path" in doc_data:
            config.documents.path = doc_data["path"]
        if "extensions" in doc_data:
            config.documents.extensions = doc_data["extensions"]

    # OpenRouter
    if "openrouter" in data:
        or_data = data["openrouter"]
        if "api_key" in or_data:
            # Handle ${VAR} syntax
            key = or_data["api_key"]
            if key.startswith("${") and key.endswith("}"):
                var_name = key[2:-1]
                key = os.getenv(var_name, "")
            config.openrouter.api_key = key
        if "embedding_model" in or_data:
            config.openrouter.embedding_model = or_data["embedding_model"]
        if "base_url" in or_data:
            config.openrouter.base_url = or_data["base_url"]

    # Analysis
    if "analysis" in data:
        an_data = data["analysis"]
        if "similarity_threshold" in an_data:
            config.analysis.similarity_threshold = float(an_data["similarity_threshold"])
        if "max_pairs_to_analyze" in an_data:
            config.analysis.max_pairs_to_analyze = int(an_data["max_pairs_to_analyze"])
        if "skip_same_file" in an_data:
            config.analysis.skip_same_file = bool(an_data["skip_same_file"])
        if "min_statement_length" in an_data:
            config.analysis.min_statement_length = int(an_data["min_statement_length"])
        if "anomaly_percentile" in an_data:
            config.analysis.anomaly_percentile = float(an_data["anomaly_percentile"])

    # Anomaly detection
    if "anomaly" in data:
        anom_data = data["anomaly"]
        if "method" in anom_data:
            config.anomaly.method = anom_data["method"]
        if "contamination" in anom_data:
            config.anomaly.contamination = float(anom_data["contamination"])
        if "ensemble_weights" in anom_data:
            config.anomaly.ensemble_weights = [float(w) for w in anom_data["ensemble_weights"]]
        if "min_methods_agree" in anom_data:
            config.anomaly.min_methods_agree = int(anom_data["min_methods_agree"])
        if "lof_neighbors" in anom_data:
            config.anomaly.lof_neighbors = int(anom_data["lof_neighbors"])
        if "isolation_forest_estimators" in anom_data:
            config.anomaly.isolation_forest_estimators = int(anom_data["isolation_forest_estimators"])

    # Claude
    if "claude" in data:
        cl_data = data["claude"]
        if "command" in cl_data:
            config.claude.command = cl_data["command"]
        if "args" in cl_data:
            config.claude.args = cl_data["args"]
        if "timeout" in cl_data:
            config.claude.timeout = int(cl_data["timeout"])

    # Output
    if "output" in data:
        out_data = data["output"]
        if "format" in out_data:
            config.output.format = out_data["format"]
        if "group_by" in out_data:
            config.output.group_by = out_data["group_by"]
        if "include_non_contradictions" in out_data:
            config.output.include_non_contradictions = bool(
                out_data["include_non_contradictions"]
            )

    # Prompt template
    if "prompt_template" in data:
        config.prompt_template = data["prompt_template"]

    return config


def _merge_from_env(config: Config) -> Config:
    """Override config from environment variables."""
    api_key = os.getenv("OPENROUTER_API_KEY")
    if api_key:
        config.openrouter.api_key = api_key

    return config


def _config_to_dict(config: Config) -> dict:
    """Convert config to dict for YAML serialization."""
    # Save actual API key if set, otherwise use env var placeholder
    api_key = config.openrouter.api_key if config.openrouter.api_key else "${OPENROUTER_API_KEY}"

    return {
        "documents": {
            "path": config.documents.path,
            "extensions": config.documents.extensions,
        },
        "openrouter": {
            "api_key": api_key,
            "embedding_model": config.openrouter.embedding_model,
            "base_url": config.openrouter.base_url,
        },
        "analysis": {
            "similarity_threshold": config.analysis.similarity_threshold,
            "max_pairs_to_analyze": config.analysis.max_pairs_to_analyze,
            "skip_same_file": config.analysis.skip_same_file,
            "min_statement_length": config.analysis.min_statement_length,
        },
        "anomaly": {
            "method": config.anomaly.method,
            "contamination": config.anomaly.contamination,
            "ensemble_weights": config.anomaly.ensemble_weights,
            "min_methods_agree": config.anomaly.min_methods_agree,
            "lof_neighbors": config.anomaly.lof_neighbors,
            "isolation_forest_estimators": config.anomaly.isolation_forest_estimators,
        },
        "claude": {
            "command": config.claude.command,
            "args": config.claude.args,
            "timeout": config.claude.timeout,
        },
        "output": {
            "format": config.output.format,
            "group_by": config.output.group_by,
            "include_non_contradictions": config.output.include_non_contradictions,
        },
        "prompt_template": config.prompt_template,
    }


def init_config() -> Path:
    """Initialize default config file."""
    config = Config()
    config.save()
    return DEFAULT_CONFIG_FILE


def ensure_api_key(config: Config) -> Config:
    """Ensure API key is set, prompting user if needed.

    If no API key is configured, prompts the user to enter one
    and saves it to ~/.doc-analyzer/config.yaml for future use.

    Returns:
        Config with API key set (or raises SystemExit if user cancels)
    """
    if config.openrouter.api_key:
        return config

    # Prompt user for API key
    print("\n" + "=" * 60)
    print("OpenRouter API Key Required")
    print("=" * 60)
    print("\nNo API key found. Get one at: https://openrouter.ai/keys")
    print("The key will be saved to ~/.doc-analyzer/config.yaml\n")

    try:
        api_key = input("Enter your OpenRouter API key: ").strip()
    except (KeyboardInterrupt, EOFError):
        print("\nCancelled.")
        raise SystemExit(1)

    if not api_key:
        print("\nNo API key provided. Exiting.")
        raise SystemExit(1)

    # Validate key format (basic check)
    if not api_key.startswith("sk-"):
        print("\nWarning: API key doesn't start with 'sk-'. Proceeding anyway...")

    # Save to config
    config.openrouter.api_key = api_key
    config.save(DEFAULT_CONFIG_FILE)
    print(f"\nAPI key saved to {DEFAULT_CONFIG_FILE}")
    print("You won't need to enter it again.\n")

    return config


def show_config(config: Config) -> str:
    """Format config for display."""
    lines = [
        "# Current Configuration",
        "",
        f"Config file: {_get_config_file_path() or '(none)'}",
        "",
        "## Documents",
        f"  path: {config.documents.path}",
        f"  extensions: {config.documents.extensions}",
        "",
        "## OpenRouter",
        f"  api_key: {'***' if config.openrouter.api_key else '(not set)'}",
        f"  embedding_model: {config.openrouter.embedding_model}",
        f"  base_url: {config.openrouter.base_url}",
        "",
        "## Analysis",
        f"  similarity_threshold: {config.analysis.similarity_threshold}",
        f"  max_pairs_to_analyze: {config.analysis.max_pairs_to_analyze}",
        f"  skip_same_file: {config.analysis.skip_same_file}",
        f"  min_statement_length: {config.analysis.min_statement_length}",
        "",
        "## Anomaly Detection",
        f"  method: {config.anomaly.method}",
        f"  contamination: {config.anomaly.contamination}",
        f"  ensemble_weights: {config.anomaly.ensemble_weights}",
        f"  min_methods_agree: {config.anomaly.min_methods_agree}",
        f"  lof_neighbors: {config.anomaly.lof_neighbors}",
        f"  isolation_forest_estimators: {config.anomaly.isolation_forest_estimators}",
        "",
        "## Claude CLI",
        f"  command: {config.claude.command}",
        f"  args: {config.claude.args}",
        f"  timeout: {config.claude.timeout}s",
        "",
        "## Output",
        f"  format: {config.output.format}",
        f"  group_by: {config.output.group_by}",
        f"  include_non_contradictions: {config.output.include_non_contradictions}",
    ]
    return "\n".join(lines)
