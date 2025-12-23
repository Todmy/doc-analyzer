"""Tests for anomaly detection."""

from pathlib import Path

import numpy as np
import pytest

from doc_analyzer.anomaly import (
    detect_anomalies,
    _isolation_forest_scores,
    _lof_scores,
    _hdbscan_scores,
    _normalize_scores,
    classify_anomaly_severity,
)
from doc_analyzer.config import AnomalyConfig
from doc_analyzer.models import Statement, ClusterResult, Anomaly, AnomalyScores


@pytest.fixture
def sample_statements():
    """Create sample statements for testing."""
    return [
        Statement(text=f"Statement {i}", source_file=Path("test.md"), line_number=i)
        for i in range(100)
    ]


@pytest.fixture
def sample_embeddings():
    """Create sample embeddings with one clear outlier."""
    np.random.seed(42)
    # Normal cluster
    embeddings = np.random.randn(100, 128) * 0.1
    # Make one point an outlier
    embeddings[50] = np.random.randn(128) * 5
    return embeddings


@pytest.fixture
def sample_cluster_result():
    """Create sample cluster result."""
    labels = [0] * 90 + [-1] * 10  # 10 noise points
    centroids = [np.zeros(128).tolist()]
    return ClusterResult(
        labels=labels,
        centroids=centroids,
        n_clusters=1,
    )


class TestNormalizeScores:
    def test_normalizes_to_zero_one(self):
        scores = np.array([1.0, 5.0, 10.0, 2.0])
        normalized = _normalize_scores(scores)
        assert normalized.min() >= 0.0
        assert normalized.max() <= 1.0

    def test_handles_empty_array(self):
        scores = np.array([])
        normalized = _normalize_scores(scores)
        assert len(normalized) == 0


class TestIsolationForestScores:
    def test_returns_correct_shape(self, sample_embeddings):
        config = AnomalyConfig(contamination=0.05)
        scores = _isolation_forest_scores(sample_embeddings, config)
        assert len(scores) == len(sample_embeddings)

    def test_scores_in_range(self, sample_embeddings):
        config = AnomalyConfig(contamination=0.05)
        scores = _isolation_forest_scores(sample_embeddings, config)
        assert all(0 <= s <= 1 for s in scores)

    def test_outlier_has_high_score(self, sample_embeddings):
        config = AnomalyConfig(contamination=0.05)
        scores = _isolation_forest_scores(sample_embeddings, config)
        # Point 50 is the outlier
        assert scores[50] > np.median(scores)


class TestLofScores:
    def test_returns_correct_shape(self, sample_embeddings):
        config = AnomalyConfig(lof_neighbors=10)
        scores = _lof_scores(sample_embeddings, config)
        assert len(scores) == len(sample_embeddings)

    def test_scores_in_range(self, sample_embeddings):
        config = AnomalyConfig(lof_neighbors=10)
        scores = _lof_scores(sample_embeddings, config)
        assert all(0 <= s <= 1 for s in scores)


class TestHdbscanScores:
    def test_noise_points_get_high_score(self, sample_cluster_result):
        scores = _hdbscan_scores(sample_cluster_result)
        # Last 10 points are noise (label=-1)
        assert all(scores[i] == 1.0 for i in range(90, 100))

    def test_cluster_points_get_low_score(self, sample_cluster_result):
        scores = _hdbscan_scores(sample_cluster_result)
        # First 90 points are in cluster 0
        assert all(scores[i] == 0.0 for i in range(90))


class TestDetectAnomalies:
    def test_returns_anomaly_list(
        self, sample_statements, sample_embeddings, sample_cluster_result
    ):
        config = AnomalyConfig(method="ensemble", contamination=0.1)
        anomalies = detect_anomalies(
            sample_embeddings, sample_statements, sample_cluster_result, config
        )
        assert isinstance(anomalies, list)
        assert all(isinstance(a, Anomaly) for a in anomalies)

    def test_anomalies_sorted_by_score(
        self, sample_statements, sample_embeddings, sample_cluster_result
    ):
        config = AnomalyConfig(method="ensemble", contamination=0.1)
        anomalies = detect_anomalies(
            sample_embeddings, sample_statements, sample_cluster_result, config
        )
        if len(anomalies) > 1:
            scores = [a.score for a in anomalies]
            assert scores == sorted(scores, reverse=True)

    def test_different_methods(
        self, sample_statements, sample_embeddings, sample_cluster_result
    ):
        for method in ["isolation_forest", "lof", "hdbscan", "ensemble"]:
            config = AnomalyConfig(method=method, contamination=0.1)
            anomalies = detect_anomalies(
                sample_embeddings, sample_statements, sample_cluster_result, config
            )
            assert isinstance(anomalies, list)


class TestClassifyAnomalySeverity:
    def test_high_severity(self):
        anomaly = Anomaly(
            statement=Statement(text="test", source_file=Path("t.md"), line_number=1),
            statement_idx=0,
            score=0.9,
            cluster_id=0,
            reason="test",
            methods_flagged=["if", "lof", "hdbscan"],
        )
        assert classify_anomaly_severity(anomaly) == "high"

    def test_medium_severity(self):
        anomaly = Anomaly(
            statement=Statement(text="test", source_file=Path("t.md"), line_number=1),
            statement_idx=0,
            score=0.7,
            cluster_id=0,
            reason="test",
            methods_flagged=["if", "lof"],
        )
        assert classify_anomaly_severity(anomaly) == "medium"

    def test_low_severity(self):
        anomaly = Anomaly(
            statement=Statement(text="test", source_file=Path("t.md"), line_number=1),
            statement_idx=0,
            score=0.5,
            cluster_id=0,
            reason="test",
            methods_flagged=["if"],
        )
        assert classify_anomaly_severity(anomaly) == "low"
