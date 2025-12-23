"""Reporter: Generate analysis reports in various formats."""

import json
from datetime import datetime
from pathlib import Path

from .models import (
    AnalysisReport,
    Anomaly,
    ContradictionResult,
    Severity,
    Statistics,
)


def generate_report(
    report: AnalysisReport,
    output_format: str = "markdown",
    group_by: str = "severity",
) -> str:
    """Generate formatted report.

    Args:
        report: AnalysisReport object
        output_format: "markdown" or "json"
        group_by: "severity", "file", or "type"

    Returns:
        Formatted report string
    """
    if output_format == "json":
        return _generate_json_report(report)
    else:
        return _generate_markdown_report(report, group_by)


def _generate_markdown_report(report: AnalysisReport, group_by: str) -> str:
    """Generate markdown report."""
    lines: list[str] = []

    # Header
    lines.append("# Document Analysis Report")
    lines.append("")
    lines.append(f"**Scanned:** {report.statistics.total_files} files, {report.statistics.total_statements} statements")
    lines.append(f"**Topics detected:** {report.clusters.n_clusters} clusters")
    lines.append(f"**Generated:** {datetime.now().strftime('%Y-%m-%d %H:%M')}")
    lines.append("")
    lines.append("---")
    lines.append("")

    # Statistics section
    lines.extend(_format_statistics_section(report.statistics))
    lines.append("")

    # Anomalies section
    if report.anomalies:
        lines.extend(_format_anomalies_section(report.anomalies))
        lines.append("")

    # Contradictions section
    if report.contradictions:
        lines.extend(_format_contradictions_section(report.contradictions, group_by))
        lines.append("")

    # Summary
    lines.extend(_format_summary(report))

    return "\n".join(lines)


def _format_statistics_section(stats: Statistics) -> list[str]:
    """Format statistics section."""
    lines = [
        "## Statistics",
        "",
        "### Topic Distribution",
        "",
        "| Cluster | Topic | Statements | Files | Density |",
        "|---------|-------|------------|-------|---------|",
    ]

    for cluster_id in sorted(stats.per_cluster.keys()):
        if cluster_id == -1:
            continue
        cs = stats.per_cluster[cluster_id]
        lines.append(f"| {cluster_id} | {cs.name} | {cs.count} | {len(cs.files)} | {cs.density:.2f} |")

    # File coverage matrix
    lines.append("")
    lines.append("### File Coverage Matrix")
    lines.append("")

    # Get all cluster IDs (excluding noise)
    all_clusters = sorted([c for c in stats.per_cluster.keys() if c >= 0])
    if all_clusters:
        # Header row
        cluster_names = [stats.per_cluster[c].name[:12] for c in all_clusters]
        lines.append("| File | " + " | ".join(cluster_names) + " |")
        lines.append("|" + "------|" * (len(all_clusters) + 1))

        # Data rows
        for file_path, clusters in sorted(stats.coverage_matrix.items()):
            file_name = Path(file_path).name[:30]
            coverage = ["\u2713" if c in clusters else "-" for c in all_clusters]
            lines.append(f"| {file_name} | " + " | ".join(coverage) + " |")

    return lines


def _format_anomalies_section(anomalies: list[Anomaly]) -> list[str]:
    """Format anomalies section."""
    lines = [
        "---",
        "",
        f"## Anomalies ({len(anomalies)})",
        "",
    ]

    for i, anomaly in enumerate(anomalies, 1):
        lines.append(f"### {i}. Anomalous statement")
        lines.append(f"- **File:** {anomaly.statement.source_file.name}:{anomaly.statement.line_number}")
        lines.append(f"  > \"{anomaly.statement.text[:100]}{'...' if len(anomaly.statement.text) > 100 else ''}\"")
        lines.append(f"- **Distance:** {anomaly.distance:.3f}")
        lines.append(f"- **Cluster:** {anomaly.cluster_id}")
        lines.append(f"- **Reason:** {anomaly.reason}")
        lines.append("")

    return lines


def _format_contradictions_section(
    contradictions: list[ContradictionResult],
    group_by: str,
) -> list[str]:
    """Format contradictions section."""
    lines = [
        "---",
        "",
        f"## Contradictions ({len(contradictions)})",
        "",
    ]

    if group_by == "severity":
        lines.extend(_group_by_severity(contradictions))
    elif group_by == "file":
        lines.extend(_group_by_file(contradictions))
    else:
        lines.extend(_group_by_type(contradictions))

    return lines


def _group_by_severity(contradictions: list[ContradictionResult]) -> list[str]:
    """Group contradictions by severity."""
    lines: list[str] = []

    severity_order = [Severity.CRITICAL, Severity.HIGH, Severity.MEDIUM, Severity.LOW]

    for severity in severity_order:
        group = [c for c in contradictions if c.severity == severity]
        if not group:
            continue

        lines.append(f"### {severity.value.title()} ({len(group)})")
        lines.append("")

        for i, c in enumerate(group, 1):
            lines.extend(_format_contradiction(c, i))

    return lines


def _group_by_file(contradictions: list[ContradictionResult]) -> list[str]:
    """Group contradictions by file."""
    lines: list[str] = []

    # Group by file pairs
    by_files: dict[str, list[ContradictionResult]] = {}
    for c in contradictions:
        key = f"{c.statement_a.source_file.name} <-> {c.statement_b.source_file.name}"
        if key not in by_files:
            by_files[key] = []
        by_files[key].append(c)

    for files, group in sorted(by_files.items()):
        lines.append(f"### {files} ({len(group)})")
        lines.append("")

        for i, c in enumerate(group, 1):
            lines.extend(_format_contradiction(c, i))

    return lines


def _group_by_type(contradictions: list[ContradictionResult]) -> list[str]:
    """Group contradictions by type."""
    lines: list[str] = []

    by_type: dict[str, list[ContradictionResult]] = {}
    for c in contradictions:
        key = c.contradiction_type.value
        if key not in by_type:
            by_type[key] = []
        by_type[key].append(c)

    for type_name, group in sorted(by_type.items()):
        lines.append(f"### {type_name.title()} ({len(group)})")
        lines.append("")

        for i, c in enumerate(group, 1):
            lines.extend(_format_contradiction(c, i))

    return lines


def _format_contradiction(c: ContradictionResult, num: int) -> list[str]:
    """Format a single contradiction."""
    return [
        f"#### {num}. {c.contradiction_type.value.title()} contradiction",
        f"- **File A:** {c.statement_a.source_file.name}:{c.statement_a.line_number}",
        f"  > \"{c.statement_a.text[:100]}{'...' if len(c.statement_a.text) > 100 else ''}\"",
        f"- **File B:** {c.statement_b.source_file.name}:{c.statement_b.line_number}",
        f"  > \"{c.statement_b.text[:100]}{'...' if len(c.statement_b.text) > 100 else ''}\"",
        f"- **Similarity:** {c.similarity:.2f}",
        f"- **Confidence:** {c.confidence:.2f}",
        f"- **Explanation:** {c.explanation}",
        "",
    ]


def _format_summary(report: AnalysisReport) -> list[str]:
    """Format summary section."""
    summary = report.summary

    return [
        "---",
        "",
        "## Summary",
        "",
        "| Category | Count | Action |",
        "|----------|-------|--------|",
        f"| Contradictions | {summary['total_contradictions']} | Review & resolve |",
        f"| Critical | {summary['critical_contradictions']} | Immediate attention |",
        f"| Anomalies | {summary['total_anomalies']} | Check if intentional |",
        f"| Topics | {summary['total_clusters']} | - |",
        "",
    ]


def _generate_json_report(report: AnalysisReport) -> str:
    """Generate JSON report."""
    data = {
        "generated": datetime.now().isoformat(),
        "summary": report.summary,
        "statistics": {
            "total_statements": report.statistics.total_statements,
            "total_files": report.statistics.total_files,
            "per_file": report.statistics.per_file,
            "cluster_balance": report.statistics.cluster_balance,
            "similarity_distribution": report.statistics.similarity_distribution,
        },
        "clusters": {
            "n_clusters": report.clusters.n_clusters,
            "sizes": report.clusters.get_cluster_sizes(),
        },
        "anomalies": [
            {
                "file": str(a.statement.source_file),
                "line": a.statement.line_number,
                "text": a.statement.text,
                "distance": a.distance,
                "cluster_id": a.cluster_id,
                "reason": a.reason,
            }
            for a in report.anomalies
        ],
        "contradictions": [
            {
                "statement_a": {
                    "file": str(c.statement_a.source_file),
                    "line": c.statement_a.line_number,
                    "text": c.statement_a.text,
                },
                "statement_b": {
                    "file": str(c.statement_b.source_file),
                    "line": c.statement_b.line_number,
                    "text": c.statement_b.text,
                },
                "similarity": c.similarity,
                "is_contradiction": c.is_contradiction,
                "confidence": c.confidence,
                "type": c.contradiction_type.value,
                "severity": c.severity.value,
                "explanation": c.explanation,
            }
            for c in report.contradictions
        ],
    }

    return json.dumps(data, indent=2, ensure_ascii=False)


def save_report(content: str, output_path: Path) -> None:
    """Save report to file."""
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as f:
        f.write(content)
