package models

import (
	"time"
)

// User represents a registered user
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Project represents an analysis project
type Project struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Document represents an uploaded document
type Document struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Filename  string    `json:"filename"`
	Content   string    `json:"-"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

// Statement represents an extracted statement from a document
type Statement struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	Text       string    `json:"text"`
	Position   int       `json:"position"`
	Embedding  []float32 `json:"-"`
}

// Cluster represents a group of related statements
type Cluster struct {
	ID        string   `json:"id"`
	ProjectID string   `json:"project_id"`
	Label     int      `json:"label"`
	Keywords  []string `json:"keywords"`
	Size      int      `json:"size"`
	Density   float64  `json:"density"`
}

// VisualizationPoint represents a point in the visualization
type VisualizationPoint struct {
	ID           string  `json:"id"`
	StatementID  string  `json:"statement_id"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Z            float64 `json:"z,omitempty"`
	ClusterID    string  `json:"cluster_id"`
	AnomalyScore float64 `json:"anomaly_score"`
	Preview      string  `json:"preview"`
	SourceFile   string  `json:"source_file"`
}

// SimilarPair represents two similar statements
type SimilarPair struct {
	Statement1ID string  `json:"statement1_id"`
	Statement2ID string  `json:"statement2_id"`
	Text1        string  `json:"text1"`
	Text2        string  `json:"text2"`
	Similarity   float64 `json:"similarity"`
	File1        string  `json:"file1"`
	File2        string  `json:"file2"`
}

// Anomaly represents an outlier statement
type Anomaly struct {
	StatementID string  `json:"statement_id"`
	Text        string  `json:"text"`
	Score       float64 `json:"score"`
	SourceFile  string  `json:"source_file"`
}

// Contradiction represents a detected contradiction
type Contradiction struct {
	ID           string  `json:"id"`
	Statement1ID string  `json:"statement1_id"`
	Statement2ID string  `json:"statement2_id"`
	Text1        string  `json:"text1"`
	Text2        string  `json:"text2"`
	Type         string  `json:"type"` // direct, numerical, temporal, implicit
	Severity     string  `json:"severity"`
	Explanation  string  `json:"explanation"`
}

// SemanticAxis represents a user-defined dimension
type SemanticAxis struct {
	Word      string `json:"word"`
	Dimension int    `json:"dimension"`
}
