"""Data models for document analysis."""

from dataclasses import dataclass, field
from enum import Enum
from pathlib import Path


class ContradictionType(Enum):
    """Types of contradictions."""
    DIRECT = "direct"           # X vs not-X
    NUMERICAL = "numerical"     # Different values for same metric
    TEMPORAL = "temporal"       # Different timelines
    IMPLICIT = "implicit"       # Assumptions that can't both be true
    NONE = "none"


class Severity(Enum):
    """Severity levels for findings."""
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"


class AnomalyMethod(Enum):
    """Anomaly detection methods."""
    ISOLATION_FOREST = "isolation_forest"
    LOF = "lof"
    HDBSCAN = "hdbscan"
    ENSEMBLE = "ensemble"


@dataclass
class Statement:
    """A single statement extracted from a document."""
    text: str
    source_file: Path
    line_number: int
    context: str = ""  # Surrounding text for context

    @property
    def location(self) -> str:
        """Return file:line format."""
        return f"{self.source_file}:{self.line_number}"


@dataclass
class SimilarPair:
    """A pair of similar statements."""
    idx_a: int
    idx_b: int
    similarity: float

    def __post_init__(self):
        # Ensure idx_a < idx_b for consistency
        if self.idx_a > self.idx_b:
            self.idx_a, self.idx_b = self.idx_b, self.idx_a


@dataclass
class ContradictionResult:
    """Result of contradiction analysis between two statements."""
    statement_a: Statement
    statement_b: Statement
    similarity: float
    is_contradiction: bool
    confidence: float
    contradiction_type: ContradictionType
    severity: Severity
    explanation: str


@dataclass
class ClusterResult:
    """Result of clustering statements."""
    labels: list[int]  # Cluster ID for each statement (-1 = noise)
    centroids: list[list[float]]  # Centroid vectors
    n_clusters: int
    cluster_names: dict[int, str] = field(default_factory=dict)  # Optional names

    def get_cluster_indices(self, cluster_id: int) -> list[int]:
        """Get indices of statements in a cluster."""
        return [i for i, label in enumerate(self.labels) if label == cluster_id]

    def get_cluster_sizes(self) -> dict[int, int]:
        """Get size of each cluster."""
        sizes: dict[int, int] = {}
        for label in self.labels:
            sizes[label] = sizes.get(label, 0) + 1
        return sizes


@dataclass
class AnomalyScores:
    """Scores from individual anomaly detectors."""
    isolation_forest: float = 0.0  # 0-1, higher = more anomalous
    lof: float = 0.0  # 0-1, higher = more anomalous
    hdbscan: float = 0.0  # 0 or 1 (noise or not)
    centroid_distance: float = 0.0  # Normalized distance from centroid


@dataclass
class Anomaly:
    """An anomalous statement (outlier)."""
    statement: Statement
    statement_idx: int
    score: float  # Combined anomaly score (0-1)
    cluster_id: int  # Which cluster it was assigned to
    reason: str  # Why it's considered anomalous
    scores: AnomalyScores = field(default_factory=AnomalyScores)  # Individual detector scores
    methods_flagged: list[str] = field(default_factory=list)  # Which methods flagged this


@dataclass
class ClusterStats:
    """Statistics for a single cluster."""
    cluster_id: int
    name: str
    count: int
    density: float  # Average similarity within cluster
    files: set[str]  # Files that contain statements from this cluster


@dataclass
class Statistics:
    """Overall statistics for the document analysis."""
    total_statements: int
    total_files: int
    per_file: dict[str, int]  # Statements per file
    per_cluster: dict[int, ClusterStats]
    coverage_matrix: dict[str, set[int]]  # file -> cluster IDs covered
    similarity_distribution: dict[str, int]  # Histogram buckets
    cluster_balance: float  # Gini coefficient (0=equal, 1=unequal)


@dataclass
class AnalysisReport:
    """Complete analysis report."""
    statements: list[Statement]
    clusters: ClusterResult
    contradictions: list[ContradictionResult]
    anomalies: list[Anomaly]
    statistics: Statistics

    @property
    def summary(self) -> dict:
        """Return summary counts."""
        return {
            "total_statements": len(self.statements),
            "total_clusters": self.clusters.n_clusters,
            "total_contradictions": len(self.contradictions),
            "critical_contradictions": sum(
                1 for c in self.contradictions if c.severity == Severity.CRITICAL
            ),
            "total_anomalies": len(self.anomalies),
        }
