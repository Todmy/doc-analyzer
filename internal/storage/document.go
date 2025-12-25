package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Document represents a document in the system
type Document struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Filename    string
	Content     string
	ContentHash string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DocumentRepository defines the interface for document storage operations
type DocumentRepository interface {
	Create(ctx context.Context, document *Document) error
	GetByID(ctx context.Context, id uuid.UUID) (*Document, error)
	GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]*Document, error)
	GetByHash(ctx context.Context, projectID uuid.UUID, hash string) (*Document, error)
	Update(ctx context.Context, document *Document) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByProjectID(ctx context.Context, projectID uuid.UUID) error
}

// PostgresDocumentRepository implements DocumentRepository using PostgreSQL
type PostgresDocumentRepository struct {
	db *sql.DB
}

// NewPostgresDocumentRepository creates a new PostgresDocumentRepository
func NewPostgresDocumentRepository(db *sql.DB) *PostgresDocumentRepository {
	return &PostgresDocumentRepository{db: db}
}

// Create inserts a new document into the database
func (r *PostgresDocumentRepository) Create(ctx context.Context, document *Document) error {
	if document.ID == uuid.Nil {
		document.ID = uuid.New()
	}

	now := time.Now()
	if document.CreatedAt.IsZero() {
		document.CreatedAt = now
	}
	if document.UpdatedAt.IsZero() {
		document.UpdatedAt = now
	}

	query := `
		INSERT INTO documents (id, project_id, filename, content, content_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		document.ID,
		document.ProjectID,
		document.Filename,
		document.Content,
		document.ContentHash,
		document.CreatedAt,
		document.UpdatedAt,
	)

	return err
}

// GetByID retrieves a document by its ID
func (r *PostgresDocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*Document, error) {
	query := `
		SELECT id, project_id, filename, content, content_hash, created_at, updated_at
		FROM documents
		WHERE id = $1
	`

	document := &Document{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&document.ID,
		&document.ProjectID,
		&document.Filename,
		&document.Content,
		&document.ContentHash,
		&document.CreatedAt,
		&document.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return document, nil
}

// GetByProjectID retrieves all documents for a specific project
func (r *PostgresDocumentRepository) GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]*Document, error) {
	query := `
		SELECT id, project_id, filename, content, content_hash, created_at, updated_at
		FROM documents
		WHERE project_id = $1
		ORDER BY filename ASC
	`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []*Document
	for rows.Next() {
		document := &Document{}
		err := rows.Scan(
			&document.ID,
			&document.ProjectID,
			&document.Filename,
			&document.Content,
			&document.ContentHash,
			&document.CreatedAt,
			&document.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return documents, nil
}

// GetByHash retrieves a document by its content hash within a project
func (r *PostgresDocumentRepository) GetByHash(ctx context.Context, projectID uuid.UUID, hash string) (*Document, error) {
	query := `
		SELECT id, project_id, filename, content, content_hash, created_at, updated_at
		FROM documents
		WHERE project_id = $1 AND content_hash = $2
	`

	document := &Document{}
	err := r.db.QueryRowContext(ctx, query, projectID, hash).Scan(
		&document.ID,
		&document.ProjectID,
		&document.Filename,
		&document.Content,
		&document.ContentHash,
		&document.CreatedAt,
		&document.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return document, nil
}

// Update modifies an existing document
func (r *PostgresDocumentRepository) Update(ctx context.Context, document *Document) error {
	document.UpdatedAt = time.Now()

	query := `
		UPDATE documents
		SET filename = $2, content = $3, content_hash = $4, updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		document.ID,
		document.Filename,
		document.Content,
		document.ContentHash,
		document.UpdatedAt,
	)

	return err
}

// Delete removes a document from the database
func (r *PostgresDocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteByProjectID removes all documents for a project
func (r *PostgresDocumentRepository) DeleteByProjectID(ctx context.Context, projectID uuid.UUID) error {
	query := `DELETE FROM documents WHERE project_id = $1`
	_, err := r.db.ExecContext(ctx, query, projectID)
	return err
}
