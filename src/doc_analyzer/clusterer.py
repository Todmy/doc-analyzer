"""Clusterer: Group statements into topic clusters."""

import numpy as np
from sklearn.cluster import KMeans, HDBSCAN
from sklearn.metrics import silhouette_score

from .models import ClusterResult, Statement


def cluster_statements(
    embeddings: np.ndarray,
    n_clusters: int | None = None,
    method: str = "auto",
    min_cluster_size: int = 5,
) -> ClusterResult:
    """Cluster statements by semantic similarity.

    Args:
        embeddings: Embedding vectors (n_statements, embedding_dim)
        n_clusters: Number of clusters (None = auto-detect)
        method: "kmeans", "hdbscan", or "auto"
        min_cluster_size: Minimum cluster size for HDBSCAN

    Returns:
        ClusterResult with labels, centroids, and cluster count
    """
    n_samples = len(embeddings)

    if n_samples < 3:
        # Not enough samples to cluster
        return ClusterResult(
            labels=[0] * n_samples,
            centroids=[embeddings.mean(axis=0).tolist()],
            n_clusters=1,
        )

    # Determine method
    if method == "auto":
        # Use HDBSCAN for larger datasets, KMeans for smaller
        method = "hdbscan" if n_samples > 50 else "kmeans"

    if method == "hdbscan":
        return _cluster_hdbscan(embeddings, min_cluster_size)
    else:
        return _cluster_kmeans(embeddings, n_clusters)


def _cluster_kmeans(
    embeddings: np.ndarray,
    n_clusters: int | None = None,
) -> ClusterResult:
    """Cluster using K-means."""
    n_samples = len(embeddings)

    if n_clusters is None:
        # Auto-detect optimal k using elbow method
        n_clusters = _find_optimal_k(embeddings)

    # Ensure k is reasonable
    n_clusters = max(2, min(n_clusters, n_samples // 2))

    kmeans = KMeans(n_clusters=n_clusters, random_state=42, n_init=10)
    labels = kmeans.fit_predict(embeddings)

    return ClusterResult(
        labels=labels.tolist(),
        centroids=[c.tolist() for c in kmeans.cluster_centers_],
        n_clusters=n_clusters,
    )


def _cluster_hdbscan(
    embeddings: np.ndarray,
    min_cluster_size: int = 5,
) -> ClusterResult:
    """Cluster using HDBSCAN (density-based)."""
    # Adjust min_cluster_size based on dataset size
    n_samples = len(embeddings)
    min_size = max(2, min(min_cluster_size, n_samples // 10))

    hdbscan = HDBSCAN(
        min_cluster_size=min_size,
        min_samples=1,
        metric="euclidean",
    )
    labels = hdbscan.fit_predict(embeddings)

    # Calculate centroids for each cluster
    unique_labels = set(labels)
    centroids: list[list[float]] = []

    for label in sorted(unique_labels):
        if label == -1:
            # Noise points - use mean of all noise
            mask = labels == label
            if mask.any():
                centroids.append(embeddings[mask].mean(axis=0).tolist())
        else:
            mask = labels == label
            centroids.append(embeddings[mask].mean(axis=0).tolist())

    # Count actual clusters (excluding noise)
    n_clusters = len([l for l in unique_labels if l >= 0])

    return ClusterResult(
        labels=labels.tolist(),
        centroids=centroids,
        n_clusters=n_clusters,
    )


def _find_optimal_k(embeddings: np.ndarray, max_k: int = 15) -> int:
    """Find optimal number of clusters using silhouette score."""
    n_samples = len(embeddings)
    max_k = min(max_k, n_samples - 1)

    if max_k < 2:
        return 2

    best_k = 2
    best_score = -1

    for k in range(2, max_k + 1):
        kmeans = KMeans(n_clusters=k, random_state=42, n_init=10)
        labels = kmeans.fit_predict(embeddings)

        # Calculate silhouette score
        try:
            score = silhouette_score(embeddings, labels)
            if score > best_score:
                best_score = score
                best_k = k
        except ValueError:
            continue

    return best_k


def get_cluster_samples(
    statements: list[Statement],
    cluster_result: ClusterResult,
    cluster_id: int,
    max_samples: int = 5,
) -> list[Statement]:
    """Get sample statements from a cluster."""
    indices = cluster_result.get_cluster_indices(cluster_id)
    return [statements[i] for i in indices[:max_samples]]


def get_cluster_keywords(
    statements: list[Statement],
    cluster_result: ClusterResult,
    cluster_id: int,
    top_n: int = 5,
) -> list[str]:
    """Extract top keywords from cluster statements.

    Simple approach: word frequency analysis.
    """
    from collections import Counter
    import re

    indices = cluster_result.get_cluster_indices(cluster_id)
    texts = [statements[i].text for i in indices]

    # Tokenize and count
    words: list[str] = []
    for text in texts:
        # Simple tokenization
        tokens = re.findall(r"\b[a-zA-Z]{3,}\b", text.lower())
        words.extend(tokens)

    # Filter common words
    stopwords = {
        "the", "and", "for", "are", "but", "not", "you", "all", "can",
        "her", "was", "one", "our", "out", "has", "have", "been", "this",
        "that", "with", "will", "your", "from", "they", "more", "when",
        "there", "what", "about", "which", "their", "than", "into", "also",
    }

    filtered = [w for w in words if w not in stopwords]
    counter = Counter(filtered)

    return [word for word, _ in counter.most_common(top_n)]
