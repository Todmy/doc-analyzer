package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := NewPostgresRepository(db)

	user := &User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), user)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if user.ID == "" {
		t.Error("expected user ID to be generated")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestPostgresRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := NewPostgresRepository(db)

	userID := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@example.com"
	passwordHash := "hashed_password"
	createdAt := time.Now()
	updatedAt := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at", "updated_at"}).
		AddRow(userID, email, passwordHash, createdAt, updatedAt)

	mock.ExpectQuery("SELECT (.+) FROM users WHERE id").
		WithArgs(userID).
		WillReturnRows(rows)

	user, err := repo.GetByID(context.Background(), userID)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if user == nil {
		t.Fatal("expected user to be returned")
	}

	if user.ID != userID {
		t.Errorf("expected ID %s, got %s", userID, user.ID)
	}

	if user.Email != email {
		t.Errorf("expected email %s, got %s", email, user.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestPostgresRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := NewPostgresRepository(db)

	userID := "nonexistent"

	mock.ExpectQuery("SELECT (.+) FROM users WHERE id").
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetByID(context.Background(), userID)
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}

	if user != nil {
		t.Error("expected nil user")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestPostgresRepository_GetByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := NewPostgresRepository(db)

	userID := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@example.com"
	passwordHash := "hashed_password"
	createdAt := time.Now()
	updatedAt := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "created_at", "updated_at"}).
		AddRow(userID, email, passwordHash, createdAt, updatedAt)

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs(email).
		WillReturnRows(rows)

	user, err := repo.GetByEmail(context.Background(), email)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if user == nil {
		t.Fatal("expected user to be returned")
	}

	if user.ID != userID {
		t.Errorf("expected ID %s, got %s", userID, user.ID)
	}

	if user.Email != email {
		t.Errorf("expected email %s, got %s", email, user.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestPostgresRepository_GetByEmail_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := NewPostgresRepository(db)

	email := "nonexistent@example.com"

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetByEmail(context.Background(), email)
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}

	if user != nil {
		t.Error("expected nil user")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
