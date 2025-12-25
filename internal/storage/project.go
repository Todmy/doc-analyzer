package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Project represents a project in the system
type Project struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProjectRepository defines the interface for project storage operations
type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Project, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostgresProjectRepository implements ProjectRepository using PostgreSQL
type PostgresProjectRepository struct {
	db *sql.DB
}

// NewPostgresProjectRepository creates a new PostgresProjectRepository
func NewPostgresProjectRepository(db *sql.DB) *PostgresProjectRepository {
	return &PostgresProjectRepository{db: db}
}

// Create inserts a new project into the database
func (r *PostgresProjectRepository) Create(ctx context.Context, project *Project) error {
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}

	now := time.Now()
	if project.CreatedAt.IsZero() {
		project.CreatedAt = now
	}
	if project.UpdatedAt.IsZero() {
		project.UpdatedAt = now
	}

	query := `
		INSERT INTO projects (id, user_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		project.ID,
		project.UserID,
		project.Name,
		project.CreatedAt,
		project.UpdatedAt,
	)

	return err
}

// GetByID retrieves a project by its ID
func (r *PostgresProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM projects
		WHERE id = $1
	`

	project := &Project{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&project.ID,
		&project.UserID,
		&project.Name,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return project, nil
}

// GetByUserID retrieves all projects for a specific user
func (r *PostgresProjectRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Project, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM projects
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}
		err := rows.Scan(
			&project.ID,
			&project.UserID,
			&project.Name,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return projects, nil
}

// Update modifies an existing project
func (r *PostgresProjectRepository) Update(ctx context.Context, project *Project) error {
	project.UpdatedAt = time.Now()

	query := `
		UPDATE projects
		SET name = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		project.ID,
		project.Name,
		project.UpdatedAt,
	)

	return err
}

// Delete removes a project from the database
func (r *PostgresProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
