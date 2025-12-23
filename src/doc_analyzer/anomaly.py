"""Hybrid anomaly detection: Ensemble of Isolation Forest, LOF, and HDBSCAN."""

import numpy as np
from sklearn.ensemble import IsolationForest
from sklearn.neighbors import LocalOutlierFactor
from sklearn.preprocessing import MinMaxScaler

from .config import AnomalyConfig
from .models import Anomaly, AnomalyMethod, AnomalyScores, ClusterResult, Statement


def detect_anomalies(
    embeddings: np.ndarray,
    statements: list[Statement],
    cluster_result: ClusterResult,
    config: AnomalyConfig | None = None,
) -> list[Anomaly]:
    """Detect anomalous statements using hybrid ensemble approach.

    Methods:
    1. Isolation Forest - catches global outliers
    2. LOF (Local Outlier Factor) - catches local density anomalies
    3. HDBSCAN noise - statements that don't fit any cluster
    4. Ensemble - combines all methods with weighted voting

    Args:
        embeddings: Embedding vectors (n_statements, embedding_dim)
        statements: List of Statement objects
        cluster_result: Clustering result from HDBSCAN/KMeans
        config: Anomaly detection configuration

    Returns:
        List of Anomaly objects sorted by score (most anomalous first)
    """
    if config is None:
        config = AnomalyConfig()

    method = AnomalyMethod(config.method)

    # Calculate scores from each detector
    if_scores = _isolation_forest_scores(embeddings, config)
    lof_scores = _lof_scores(embeddings, config)
    hdbscan_scores = _hdbscan_scores(cluster_result)
    centroid_scores = _centroid_distance_scores(embeddings, cluster_result)

    # Combine scores based on method
    if method == AnomalyMethod.ISOLATION_FOREST:
        combined_scores = if_scores
        threshold = np.percentile(combined_scores, (1 - config.contamination) * 100)
    elif method == AnomalyMethod.LOF:
        combined_scores = lof_scores
        threshold = np.percentile(combined_scores, (1 - config.contamination) * 100)
    elif method == AnomalyMethod.HDBSCAN:
        combined_scores = hdbscan_scores
        threshold = 0.5  # Binary: noise or not
    else:  # ENSEMBLE
        combined_scores, threshold = _ensemble_scores(
            if_scores, lof_scores, hdbscan_scores,
            config.ensemble_weights,
            config.contamination,
            config.min_methods_agree,
        )

    # Build anomaly objects
    anomalies: list[Anomaly] = []

    for i, (stmt, score, label) in enumerate(
        zip(statements, combined_scores, cluster_result.labels)
    ):
        if score < threshold:
            continue

        # Determine which methods flagged this
        methods_flagged = []
        reasons = []

        if_threshold = np.percentile(if_scores, (1 - config.contamination) * 100)
        lof_threshold = np.percentile(lof_scores, (1 - config.contamination) * 100)

        if if_scores[i] >= if_threshold:
            methods_flagged.append("isolation_forest")
            reasons.append(f"Isolation Forest: {if_scores[i]:.3f}")

        if lof_scores[i] >= lof_threshold:
            methods_flagged.append("lof")
            reasons.append(f"LOF: {lof_scores[i]:.3f}")

        if hdbscan_scores[i] > 0.5:
            methods_flagged.append("hdbscan")
            reasons.append("HDBSCAN noise")

        if not methods_flagged:
            methods_flagged.append(config.method)
            reasons.append(f"Score: {score:.3f}")

        anomalies.append(
            Anomaly(
                statement=stmt,
                statement_idx=i,
                score=float(score),
                cluster_id=label,
                reason="; ".join(reasons),
                scores=AnomalyScores(
                    isolation_forest=float(if_scores[i]),
                    lof=float(lof_scores[i]),
                    hdbscan=float(hdbscan_scores[i]),
                    centroid_distance=float(centroid_scores[i]),
                ),
                methods_flagged=methods_flagged,
            )
        )

    # Sort by score descending (most anomalous first)
    anomalies.sort(key=lambda a: a.score, reverse=True)

    return anomalies


def _isolation_forest_scores(
    embeddings: np.ndarray,
    config: AnomalyConfig,
) -> np.ndarray:
    """Calculate anomaly scores using Isolation Forest.

    Isolation Forest isolates observations by randomly selecting a feature
    and then randomly selecting a split value. Anomalies require fewer splits.

    Returns scores in [0, 1] where higher = more anomalous.
    """
    iso = IsolationForest(
        n_estimators=config.isolation_forest_estimators,
        contamination=config.contamination,
        random_state=42,
        n_jobs=-1,
    )

    # fit_predict returns -1 for anomalies, 1 for normal
    # decision_function returns negative for anomalies
    iso.fit(embeddings)
    raw_scores = -iso.decision_function(embeddings)  # Negate so higher = anomaly

    # Normalize to [0, 1]
    return _normalize_scores(raw_scores)


def _lof_scores(
    embeddings: np.ndarray,
    config: AnomalyConfig,
) -> np.ndarray:
    """Calculate anomaly scores using Local Outlier Factor.

    LOF measures local density deviation. Points in sparse regions
    relative to their neighbors are considered anomalies.

    NOTE: For high-dimensional embeddings (1000D+), brute force is actually
    faster than ball_tree/kd_tree due to curse of dimensionality.
    Using algorithm='auto' lets sklearn choose optimally.

    Returns scores in [0, 1] where higher = more anomalous.
    """
    n_samples = len(embeddings)

    # Adjust neighbors based on sample size
    n_neighbors = min(config.lof_neighbors, n_samples - 1)
    n_neighbors = max(2, n_neighbors)

    # Use cosine metric directly - for high-dim embeddings, brute force is optimal
    # and cosine is semantically correct for text embeddings
    lof = LocalOutlierFactor(
        n_neighbors=n_neighbors,
        metric="cosine",
        algorithm="auto",  # Let sklearn choose (will pick brute for high-dim)
        contamination=config.contamination,
        novelty=False,
        n_jobs=-1,
    )

    # fit_predict returns -1 for anomalies, 1 for normal
    lof.fit_predict(embeddings)

    # negative_outlier_factor_ is negative, more negative = more anomalous
    raw_scores = -lof.negative_outlier_factor_

    # Normalize to [0, 1]
    return _normalize_scores(raw_scores)


def _hdbscan_scores(cluster_result: ClusterResult) -> np.ndarray:
    """Calculate anomaly scores from HDBSCAN clustering.

    Points labeled as noise (-1) by HDBSCAN get score 1.0,
    normal cluster members get 0.0.

    Returns scores in [0, 1] where higher = more anomalous.
    """
    labels = np.array(cluster_result.labels)
    scores = np.where(labels == -1, 1.0, 0.0)
    return scores


def _centroid_distance_scores(
    embeddings: np.ndarray,
    cluster_result: ClusterResult,
) -> np.ndarray:
    """Calculate normalized distance from cluster centroids.

    Returns scores in [0, 1] where higher = farther from centroid.
    """
    distances = np.zeros(len(embeddings))
    centroids = np.array(cluster_result.centroids)

    if len(centroids) == 0:
        return distances

    for i, (embedding, label) in enumerate(zip(embeddings, cluster_result.labels)):
        if label == -1:
            # Noise: use distance to nearest centroid
            dists = np.linalg.norm(centroids - embedding, axis=1)
            distances[i] = dists.min()
        else:
            # Distance to assigned centroid
            if label < len(centroids):
                distances[i] = np.linalg.norm(embedding - centroids[label])

    return _normalize_scores(distances)


def _ensemble_scores(
    if_scores: np.ndarray,
    lof_scores: np.ndarray,
    hdbscan_scores: np.ndarray,
    weights: list[float],
    contamination: float,
    min_methods_agree: int,
) -> tuple[np.ndarray, float]:
    """Combine scores from multiple detectors.

    Two strategies:
    1. Weighted average of normalized scores
    2. Voting: require min_methods_agree to flag as anomaly

    Args:
        if_scores: Isolation Forest scores [0-1]
        lof_scores: LOF scores [0-1]
        hdbscan_scores: HDBSCAN scores [0-1]
        weights: Weights for [IF, LOF, HDBSCAN]
        contamination: Expected anomaly proportion
        min_methods_agree: Minimum methods that must flag

    Returns:
        Combined scores and threshold
    """
    # Normalize weights
    weights = np.array(weights)
    weights = weights / weights.sum()

    # Weighted average
    combined = (
        weights[0] * if_scores +
        weights[1] * lof_scores +
        weights[2] * hdbscan_scores
    )

    # Calculate individual thresholds
    if_threshold = np.percentile(if_scores, (1 - contamination) * 100)
    lof_threshold = np.percentile(lof_scores, (1 - contamination) * 100)
    hdbscan_threshold = 0.5

    # Count how many methods flag each point
    votes = (
        (if_scores >= if_threshold).astype(int) +
        (lof_scores >= lof_threshold).astype(int) +
        (hdbscan_scores >= hdbscan_threshold).astype(int)
    )

    # Boost score for points flagged by multiple methods
    agreement_bonus = np.where(votes >= min_methods_agree, 0.2, 0.0)
    combined = np.clip(combined + agreement_bonus, 0, 1)

    # Dynamic threshold based on voting
    # Points flagged by min_methods_agree+ get priority
    threshold = np.percentile(combined, (1 - contamination) * 100)

    return combined, threshold


def _normalize_scores(scores: np.ndarray) -> np.ndarray:
    """Normalize scores to [0, 1] range using MinMax scaling."""
    if len(scores) == 0:
        return scores

    scaler = MinMaxScaler()
    normalized = scaler.fit_transform(scores.reshape(-1, 1)).flatten()
    return normalized


# Legacy function for backward compatibility
def get_anomaly_summary(anomalies: list[Anomaly]) -> dict:
    """Get summary statistics for anomalies."""
    if not anomalies:
        return {
            "total": 0,
            "by_method": {},
            "avg_score": 0.0,
            "max_score": 0.0,
        }

    # Count by method
    method_counts: dict[str, int] = {}
    for a in anomalies:
        for method in a.methods_flagged:
            method_counts[method] = method_counts.get(method, 0) + 1

    scores = [a.score for a in anomalies]

    return {
        "total": len(anomalies),
        "by_method": method_counts,
        "avg_score": sum(scores) / len(scores),
        "max_score": max(scores),
        "multi_method_count": sum(1 for a in anomalies if len(a.methods_flagged) > 1),
    }


def classify_anomaly_severity(anomaly: Anomaly) -> str:
    """Classify anomaly severity based on score and method agreement."""
    # High: score > 0.8 or 3 methods agree
    if anomaly.score > 0.8 or len(anomaly.methods_flagged) >= 3:
        return "high"
    # Medium: score > 0.6 or 2 methods agree
    elif anomaly.score > 0.6 or len(anomaly.methods_flagged) >= 2:
        return "medium"
    else:
        return "low"
