package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// Statement represents a statement extracted from a document
type Statement struct {
	ID         uuid.UUID
	DocumentID uuid.UUID
	Text       string
	Position   int
	Line       int
	Embedding  pgvector.Vector
	CreatedAt  time.Time
}

// StatementRepository defines the interface for statement storage operations
type StatementRepository interface {
	Create(ctx context.Context, statement *Statement) error
	CreateBatch(ctx context.Context, statements []*Statement) error
	GetByID(ctx context.Context, id uuid.UUID) (*Statement, error)
	GetByDocumentID(ctx context.Context, documentID uuid.UUID) ([]*Statement, error)
	GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]*Statement, error)
	FindSimilar(ctx context.Context, embedding pgvector.Vector, limit int, threshold float64) ([]*StatementWithSimilarity, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error
}

// StatementWithSimilarity represents a statement with its similarity score
type StatementWithSimilarity struct {
	Statement  *Statement
	Similarity float64
}

// PostgresStatementRepository implements StatementRepository using PostgreSQL with pgvector
type PostgresStatementRepository struct {
	db *sql.DB
}

// NewPostgresStatementRepository creates a new PostgresStatementRepository
func NewPostgresStatementRepository(db *sql.DB) *PostgresStatementRepository {
	return &PostgresStatementRepository{db: db}
}

// Create inserts a new statement into the database
func (r *PostgresStatementRepository) Create(ctx context.Context, statement *Statement) error {
	if statement.ID == uuid.Nil {
		statement.ID = uuid.New()
	}

	if statement.CreatedAt.IsZero() {
		statement.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO statements (id, document_id, text, position, line, embedding, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		statement.ID,
		statement.DocumentID,
		statement.Text,
		statement.Position,
		statement.Line,
		statement.Embedding,
		statement.CreatedAt,
	)

	return err
}

// CreateBatch inserts multiple statements in a single transaction
func (r *PostgresStatementRepository) CreateBatch(ctx context.Context, statements []*Statement) error {
	if len(statements) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO statements (id, document_id, text, position, line, embedding, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, s := range statements {
		if s.ID == uuid.Nil {
			s.ID = uuid.New()
		}
		if s.CreatedAt.IsZero() {
			s.CreatedAt = now
		}

		_, err := stmt.ExecContext(ctx,
			s.ID,
			s.DocumentID,
			s.Text,
			s.Position,
			s.Line,
			s.Embedding,
			s.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByID retrieves a statement by its ID
func (r *PostgresStatementRepository) GetByID(ctx context.Context, id uuid.UUID) (*Statement, error) {
	query := `
		SELECT id, document_id, text, position, line, embedding, created_at
		FROM statements
		WHERE id = $1
	`

	statement := &Statement{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&statement.ID,
		&statement.DocumentID,
		&statement.Text,
		&statement.Position,
		&statement.Line,
		&statement.Embedding,
		&statement.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return statement, nil
}

// GetByDocumentID retrieves all statements for a specific document
func (r *PostgresStatementRepository) GetByDocumentID(ctx context.Context, documentID uuid.UUID) ([]*Statement, error) {
	query := `
		SELECT id, document_id, text, position, line, embedding, created_at
		FROM statements
		WHERE document_id = $1
		ORDER BY position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statements []*Statement
	for rows.Next() {
		statement := &Statement{}
		err := rows.Scan(
			&statement.ID,
			&statement.DocumentID,
			&statement.Text,
			&statement.Position,
			&statement.Line,
			&statement.Embedding,
			&statement.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		statements = append(statements, statement)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return statements, nil
}

// GetByProjectID retrieves all statements for a specific project (via documents)
func (r *PostgresStatementRepository) GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]*Statement, error) {
	query := `
		SELECT s.id, s.document_id, s.text, s.position, s.line, s.embedding, s.created_at
		FROM statements s
		JOIN documents d ON s.document_id = d.id
		WHERE d.project_id = $1
		ORDER BY d.filename ASC, s.position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statements []*Statement
	for rows.Next() {
		statement := &Statement{}
		err := rows.Scan(
			&statement.ID,
			&statement.DocumentID,
			&statement.Text,
			&statement.Position,
			&statement.Line,
			&statement.Embedding,
			&statement.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		statements = append(statements, statement)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return statements, nil
}

// FindSimilar finds statements similar to the given embedding using pgvector cosine distance
func (r *PostgresStatementRepository) FindSimilar(ctx context.Context, embedding pgvector.Vector, limit int, threshold float64) ([]*StatementWithSimilarity, error) {
	if limit <= 0 {
		limit = 10
	}
	if threshold <= 0 {
		threshold = 0.75
	}

	// Use cosine distance: 1 - cosine_similarity
	// We filter where 1 - distance >= threshold (i.e., distance <= 1 - threshold)
	query := `
		SELECT id, document_id, text, position, line, embedding, created_at,
			   1 - (embedding <=> $1) as similarity
		FROM statements
		WHERE 1 - (embedding <=> $1) >= $2
		ORDER BY embedding <=> $1
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, embedding, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*StatementWithSimilarity
	for rows.Next() {
		statement := &Statement{}
		var similarity float64
		err := rows.Scan(
			&statement.ID,
			&statement.DocumentID,
			&statement.Text,
			&statement.Position,
			&statement.Line,
			&statement.Embedding,
			&statement.CreatedAt,
			&similarity,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, &StatementWithSimilarity{
			Statement:  statement,
			Similarity: similarity,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// Delete removes a statement from the database
func (r *PostgresStatementRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM statements WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteByDocumentID removes all statements for a document
func (r *PostgresStatementRepository) DeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error {
	query := `DELETE FROM statements WHERE document_id = $1`
	_, err := r.db.ExecContext(ctx, query, documentID)
	return err
}
