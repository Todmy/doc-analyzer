"""Similarity: Find semantically similar statement pairs."""

import numpy as np
from sklearn.metrics.pairwise import cosine_similarity

from .models import Statement, SimilarPair


def find_similar_pairs(
    embeddings: np.ndarray,
    statements: list[Statement],
    threshold: float = 0.75,
    skip_same_file: bool = True,
    max_pairs: int | None = None,
) -> list[SimilarPair]:
    """Find pairs of similar statements above threshold.

    Args:
        embeddings: Embedding vectors (n_statements, embedding_dim)
        statements: List of statements (for filtering)
        threshold: Minimum similarity score
        skip_same_file: Skip pairs from same file
        max_pairs: Maximum pairs to return

    Returns:
        List of SimilarPair sorted by similarity (descending)
    """
    n = len(embeddings)
    if n < 2:
        return []

    # Calculate full similarity matrix
    sim_matrix = cosine_similarity(embeddings)

    # Find pairs above threshold
    pairs: list[SimilarPair] = []

    for i in range(n):
        for j in range(i + 1, n):
            sim = sim_matrix[i, j]

            if sim < threshold:
                continue

            # Skip same file if requested
            if skip_same_file:
                if statements[i].source_file == statements[j].source_file:
                    continue

            pairs.append(SimilarPair(idx_a=i, idx_b=j, similarity=float(sim)))

    # Sort by similarity descending
    pairs.sort(key=lambda p: p.similarity, reverse=True)

    # Limit if requested
    if max_pairs is not None:
        pairs = pairs[:max_pairs]

    return pairs


def get_similarity_matrix(embeddings: np.ndarray) -> np.ndarray:
    """Calculate full cosine similarity matrix."""
    return cosine_similarity(embeddings)


def get_nearest_neighbors(
    embeddings: np.ndarray,
    query_idx: int,
    k: int = 5,
) -> list[tuple[int, float]]:
    """Get k nearest neighbors for a statement.

    Args:
        embeddings: Embedding vectors
        query_idx: Index of query statement
        k: Number of neighbors

    Returns:
        List of (index, similarity) tuples
    """
    n = len(embeddings)
    if n < 2:
        return []

    # Calculate similarities to query
    query = embeddings[query_idx].reshape(1, -1)
    similarities = cosine_similarity(query, embeddings)[0]

    # Get top k (excluding self)
    indices = np.argsort(similarities)[::-1]
    neighbors: list[tuple[int, float]] = []

    for idx in indices:
        if idx == query_idx:
            continue
        neighbors.append((int(idx), float(similarities[idx])))
        if len(neighbors) >= k:
            break

    return neighbors


def get_similarity_distribution(
    embeddings: np.ndarray,
    bins: int = 10,
) -> dict[str, int]:
    """Get histogram of pairwise similarities.

    Returns:
        Dict mapping bin labels to counts
    """
    n = len(embeddings)
    if n < 2:
        return {}

    sim_matrix = cosine_similarity(embeddings)

    # Extract upper triangle (excluding diagonal)
    similarities: list[float] = []
    for i in range(n):
        for j in range(i + 1, n):
            similarities.append(sim_matrix[i, j])

    # Create histogram
    counts, edges = np.histogram(similarities, bins=bins, range=(0, 1))

    distribution: dict[str, int] = {}
    for i, count in enumerate(counts):
        label = f"{edges[i]:.1f}-{edges[i+1]:.1f}"
        distribution[label] = int(count)

    return distribution


def average_similarity(embeddings: np.ndarray) -> float:
    """Calculate average pairwise similarity."""
    n = len(embeddings)
    if n < 2:
        return 1.0

    sim_matrix = cosine_similarity(embeddings)

    # Average of upper triangle
    total = 0.0
    count = 0
    for i in range(n):
        for j in range(i + 1, n):
            total += sim_matrix[i, j]
            count += 1

    return total / count if count > 0 else 0.0
