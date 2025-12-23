"""Statistics: Calculate document analysis statistics."""

import numpy as np

from .clusterer import get_cluster_keywords
from .models import ClusterResult, ClusterStats, Statement, Statistics
from .similarity import average_similarity, get_similarity_distribution


def calculate_stats(
    statements: list[Statement],
    embeddings: np.ndarray,
    cluster_result: ClusterResult,
) -> Statistics:
    """Calculate comprehensive statistics for document analysis.

    Args:
        statements: List of statements
        embeddings: Embedding vectors
        cluster_result: Clustering result

    Returns:
        Statistics object
    """
    # Per-file counts
    per_file: dict[str, int] = {}
    for stmt in statements:
        key = str(stmt.source_file)
        per_file[key] = per_file.get(key, 0) + 1

    # Per-cluster stats
    per_cluster: dict[int, ClusterStats] = {}
    cluster_sizes = cluster_result.get_cluster_sizes()

    for cluster_id in sorted(set(cluster_result.labels)):
        indices = cluster_result.get_cluster_indices(cluster_id)

        # Files in this cluster
        files = {str(statements[i].source_file) for i in indices}

        # Calculate density (average similarity within cluster)
        if len(indices) > 1:
            cluster_embeddings = embeddings[indices]
            density = average_similarity(cluster_embeddings)
        else:
            density = 1.0

        # Get cluster name from keywords or default
        name = cluster_result.cluster_names.get(cluster_id)
        if not name:
            keywords = get_cluster_keywords(statements, cluster_result, cluster_id, top_n=3)
            name = ", ".join(keywords) if keywords else f"Cluster {cluster_id}"

        per_cluster[cluster_id] = ClusterStats(
            cluster_id=cluster_id,
            name=name,
            count=cluster_sizes.get(cluster_id, 0),
            density=density,
            files=files,
        )

    # Coverage matrix: which files cover which clusters
    coverage_matrix: dict[str, set[int]] = {}
    for stmt, label in zip(statements, cluster_result.labels):
        key = str(stmt.source_file)
        if key not in coverage_matrix:
            coverage_matrix[key] = set()
        if label >= 0:  # Exclude noise
            coverage_matrix[key].add(label)

    # Similarity distribution
    similarity_distribution = get_similarity_distribution(embeddings)

    # Cluster balance (Gini coefficient)
    cluster_balance = _calculate_gini(list(cluster_sizes.values()))

    return Statistics(
        total_statements=len(statements),
        total_files=len(per_file),
        per_file=per_file,
        per_cluster=per_cluster,
        coverage_matrix=coverage_matrix,
        similarity_distribution=similarity_distribution,
        cluster_balance=cluster_balance,
    )


def _calculate_gini(values: list[int]) -> float:
    """Calculate Gini coefficient (0=equal, 1=unequal)."""
    if not values or sum(values) == 0:
        return 0.0

    values = sorted(values)
    n = len(values)
    total = sum(values)

    # Calculate Gini coefficient
    cumsum = 0
    for i, v in enumerate(values):
        cumsum += (2 * (i + 1) - n - 1) * v

    return cumsum / (n * total)


def get_coverage_report(stats: Statistics) -> list[dict]:
    """Generate coverage report showing which files cover which topics."""
    all_clusters = set()
    for clusters in stats.coverage_matrix.values():
        all_clusters.update(clusters)

    report: list[dict] = []
    for file_path, clusters in stats.coverage_matrix.items():
        covered = len(clusters)
        total = len(all_clusters)
        coverage_pct = (covered / total * 100) if total > 0 else 0

        # Get topic names
        topics = [
            stats.per_cluster[c].name
            for c in clusters
            if c in stats.per_cluster
        ]

        report.append({
            "file": file_path,
            "topics_covered": covered,
            "total_topics": total,
            "coverage_pct": coverage_pct,
            "topics": topics,
        })

    # Sort by coverage
    report.sort(key=lambda r: r["coverage_pct"], reverse=True)
    return report


def get_sparse_topics(stats: Statistics, threshold: int = 3) -> list[ClusterStats]:
    """Find topics with few statements (potential gaps)."""
    sparse: list[ClusterStats] = []
    for cluster_stats in stats.per_cluster.values():
        if cluster_stats.cluster_id == -1:  # Skip noise
            continue
        if cluster_stats.count < threshold:
            sparse.append(cluster_stats)

    sparse.sort(key=lambda c: c.count)
    return sparse


def get_dense_topics(stats: Statistics, threshold: float = 0.8) -> list[ClusterStats]:
    """Find topics with high internal similarity (well-defined topics)."""
    dense: list[ClusterStats] = []
    for cluster_stats in stats.per_cluster.values():
        if cluster_stats.cluster_id == -1:  # Skip noise
            continue
        if cluster_stats.density >= threshold:
            dense.append(cluster_stats)

    dense.sort(key=lambda c: c.density, reverse=True)
    return dense


def format_stats_summary(stats: Statistics) -> str:
    """Format statistics as human-readable summary."""
    lines = [
        "## Statistics Summary",
        "",
        f"**Documents:** {stats.total_files} files, {stats.total_statements} statements",
        f"**Topics:** {len([c for c in stats.per_cluster if c >= 0])} clusters",
        f"**Balance:** {stats.cluster_balance:.2f} (0=equal, 1=unequal)",
        "",
        "### Topic Distribution",
        "",
        "| Topic | Statements | Files | Density |",
        "|-------|------------|-------|---------|",
    ]

    for cluster_id in sorted(stats.per_cluster.keys()):
        if cluster_id == -1:
            continue
        cs = stats.per_cluster[cluster_id]
        lines.append(f"| {cs.name} | {cs.count} | {len(cs.files)} | {cs.density:.2f} |")

    lines.append("")
    lines.append("### Similarity Distribution")
    lines.append("")

    for bucket, count in stats.similarity_distribution.items():
        bar = "█" * min(count // 5, 20)
        lines.append(f"  {bucket}: {count:4d} {bar}")

    return "\n".join(lines)
