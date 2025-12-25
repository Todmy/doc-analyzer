package contradiction

// ContradictionType represents the type of contradiction
type ContradictionType string

const (
	TypeDirect    ContradictionType = "direct"
	TypeNumerical ContradictionType = "numerical"
	TypeTemporal  ContradictionType = "temporal"
	TypeImplicit  ContradictionType = "implicit"
)

// Severity represents contradiction severity
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// StatementPair represents two statements to analyze
type StatementPair struct {
	Statement1   string
	Statement2   string
	Statement1ID string
	Statement2ID string
	File1        string
	File2        string
	Similarity   float64
}

// ContradictionResult represents a detected contradiction
type ContradictionResult struct {
	Statement1   string            `json:"statement1"`
	Statement2   string            `json:"statement2"`
	Statement1ID string            `json:"statement1_id"`
	Statement2ID string            `json:"statement2_id"`
	File1        string            `json:"file1"`
	File2        string            `json:"file2"`
	Type         ContradictionType `json:"type"`
	Severity     Severity          `json:"severity"`
	Explanation  string            `json:"explanation"`
	Confidence   float64           `json:"confidence"`
}

// AnalysisRequest represents a request to analyze contradictions
type AnalysisRequest struct {
	Pairs     []StatementPair
	MaxPairs  int
	Threshold float64
}
