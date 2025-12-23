"""Analyzer: Check statement pairs for contradictions using Claude CLI."""

import json
import re
import subprocess
from typing import Any

from rich.progress import Progress, TaskID

from .config import Config
from .models import (
    ContradictionResult,
    ContradictionType,
    Severity,
    SimilarPair,
    Statement,
)


class AnalysisError(Exception):
    """Error during contradiction analysis."""
    pass


def analyze_pairs(
    pairs: list[SimilarPair],
    statements: list[Statement],
    config: Config,
    progress: Progress | None = None,
    task_id: TaskID | None = None,
) -> list[ContradictionResult]:
    """Analyze pairs for contradictions using Claude CLI.

    Args:
        pairs: List of similar pairs to check
        statements: All statements (for lookup)
        config: Configuration object
        progress: Optional rich Progress
        task_id: Optional task ID

    Returns:
        List of ContradictionResult objects
    """
    results: list[ContradictionResult] = []

    for pair in pairs:
        stmt_a = statements[pair.idx_a]
        stmt_b = statements[pair.idx_b]

        try:
            result = analyze_pair(stmt_a, stmt_b, pair.similarity, config)
            results.append(result)
        except AnalysisError:
            # Skip failed analyses
            pass

        if progress and task_id is not None:
            progress.update(task_id, advance=1)

    # Filter to only contradictions if configured
    if not config.output.include_non_contradictions:
        results = [r for r in results if r.is_contradiction]

    # Sort by severity and confidence
    severity_order = {
        Severity.CRITICAL: 0,
        Severity.HIGH: 1,
        Severity.MEDIUM: 2,
        Severity.LOW: 3,
    }
    results.sort(key=lambda r: (severity_order.get(r.severity, 4), -r.confidence))

    return results


def analyze_pair(
    stmt_a: Statement,
    stmt_b: Statement,
    similarity: float,
    config: Config,
) -> ContradictionResult:
    """Analyze a single pair for contradiction using Claude CLI.

    Args:
        stmt_a: First statement
        stmt_b: Second statement
        similarity: Similarity score
        config: Configuration object

    Returns:
        ContradictionResult
    """
    # Build prompt
    prompt = config.prompt_template.format(
        file_a=stmt_a.source_file.name,
        line_a=stmt_a.line_number,
        text_a=stmt_a.text,
        file_b=stmt_b.source_file.name,
        line_b=stmt_b.line_number,
        text_b=stmt_b.text,
    )

    # Call Claude CLI
    response = _call_claude(prompt, config)

    # Parse response
    parsed = _parse_response(response)

    return ContradictionResult(
        statement_a=stmt_a,
        statement_b=stmt_b,
        similarity=similarity,
        is_contradiction=parsed.get("contradiction", False),
        confidence=parsed.get("confidence", 0.5),
        contradiction_type=_parse_type(parsed.get("type", "none")),
        severity=_parse_severity(parsed.get("severity", "low")),
        explanation=parsed.get("explanation", ""),
    )


def _call_claude(prompt: str, config: Config) -> str:
    """Call Claude CLI and return response."""
    cmd = [config.claude.command] + config.claude.args + [prompt]

    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=config.claude.timeout,
        )

        if result.returncode != 0:
            raise AnalysisError(f"Claude CLI error: {result.stderr}")

        return result.stdout.strip()

    except subprocess.TimeoutExpired:
        raise AnalysisError(f"Claude CLI timeout after {config.claude.timeout}s")
    except FileNotFoundError:
        raise AnalysisError(
            f"Claude CLI not found: {config.claude.command}. "
            "Install it or configure correct path in config."
        )


def _parse_response(response: str) -> dict[str, Any]:
    """Parse Claude's JSON response."""
    # Try to extract JSON from response
    # Claude might wrap it in markdown code blocks

    # Remove markdown code blocks if present
    json_match = re.search(r"```(?:json)?\s*(\{[\s\S]*?\})\s*```", response)
    if json_match:
        json_str = json_match.group(1)
    else:
        # Try to find raw JSON
        json_match = re.search(r"\{[\s\S]*\}", response)
        if json_match:
            json_str = json_match.group(0)
        else:
            # Return defaults if no JSON found
            return {
                "contradiction": False,
                "confidence": 0.0,
                "type": "none",
                "severity": "low",
                "explanation": "Could not parse response",
            }

    try:
        return json.loads(json_str)
    except json.JSONDecodeError:
        return {
            "contradiction": False,
            "confidence": 0.0,
            "type": "none",
            "severity": "low",
            "explanation": "Invalid JSON in response",
        }


def _parse_type(type_str: str) -> ContradictionType:
    """Parse contradiction type string."""
    type_map = {
        "direct": ContradictionType.DIRECT,
        "numerical": ContradictionType.NUMERICAL,
        "temporal": ContradictionType.TEMPORAL,
        "implicit": ContradictionType.IMPLICIT,
        "none": ContradictionType.NONE,
    }
    return type_map.get(type_str.lower(), ContradictionType.NONE)


def _parse_severity(severity_str: str) -> Severity:
    """Parse severity string."""
    severity_map = {
        "critical": Severity.CRITICAL,
        "high": Severity.HIGH,
        "medium": Severity.MEDIUM,
        "low": Severity.LOW,
    }
    return severity_map.get(severity_str.lower(), Severity.LOW)


def test_claude_cli(config: Config) -> bool:
    """Test if Claude CLI is available and working."""
    try:
        result = subprocess.run(
            [config.claude.command, "--version"],
            capture_output=True,
            text=True,
            timeout=5,
        )
        return result.returncode == 0
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return False


def get_contradiction_summary(results: list[ContradictionResult]) -> dict:
    """Get summary of contradiction analysis."""
    contradictions = [r for r in results if r.is_contradiction]

    by_severity: dict[str, int] = {}
    by_type: dict[str, int] = {}

    for c in contradictions:
        sev = c.severity.value
        by_severity[sev] = by_severity.get(sev, 0) + 1

        typ = c.contradiction_type.value
        by_type[typ] = by_type.get(typ, 0) + 1

    return {
        "total_analyzed": len(results),
        "total_contradictions": len(contradictions),
        "by_severity": by_severity,
        "by_type": by_type,
        "avg_confidence": (
            sum(c.confidence for c in contradictions) / len(contradictions)
            if contradictions
            else 0.0
        ),
    }
