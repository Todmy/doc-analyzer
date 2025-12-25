package api

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"github.com/todmy/doc-analyzer/internal/storage"
)

const (
	minStatementLength = 50
	maxStatementLength = 1000
)

// extractStatements extracts statements from document content
func extractStatements(content string, documentID uuid.UUID) []*storage.Statement {
	var statements []*storage.Statement

	// Split by paragraph (double newline) or single newline for lists
	paragraphs := splitIntoParagraphs(content)

	position := 0
	line := 1

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)

		// Skip empty paragraphs
		if para == "" {
			line++
			continue
		}

		// Skip code blocks and headers
		if strings.HasPrefix(para, "```") || strings.HasPrefix(para, "#") {
			line += strings.Count(para, "\n") + 1
			continue
		}

		// Clean the paragraph
		para = cleanText(para)

		// Check length requirements
		if len(para) < minStatementLength {
			line += strings.Count(para, "\n") + 1
			continue
		}

		// Truncate if too long
		if len(para) > maxStatementLength {
			para = para[:maxStatementLength] + "..."
		}

		statements = append(statements, &storage.Statement{
			DocumentID: documentID,
			Text:       para,
			Position:   position,
			Line:       line,
			Embedding:  pgvector.NewVector(nil), // Will be filled by embedding generation
		})

		position++
		line += strings.Count(para, "\n") + 1
	}

	return statements
}

// splitIntoParagraphs splits content into paragraphs
func splitIntoParagraphs(content string) []string {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Split by double newline (paragraphs)
	paragraphs := strings.Split(content, "\n\n")

	// Also split single-line items that might be list items
	var result []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// Check if this is a multi-line paragraph or list
		lines := strings.Split(p, "\n")
		if len(lines) > 1 {
			// Check if it's a list
			if isListItem(lines[0]) {
				// Split list items individually
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						result = append(result, line)
					}
				}
				continue
			}
		}

		result = append(result, p)
	}

	return result
}

// isListItem checks if a line is a list item
func isListItem(line string) bool {
	line = strings.TrimSpace(line)
	// Check for bullet points
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ") {
		return true
	}
	// Check for numbered lists
	matched, _ := regexp.MatchString(`^\d+\.?\s`, line)
	return matched
}

// cleanText removes markdown formatting and cleans up text
func cleanText(text string) string {
	// Remove markdown links but keep text
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	text = linkRegex.ReplaceAllString(text, "$1")

	// Remove bold/italic markers
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")

	// Remove inline code markers
	text = strings.ReplaceAll(text, "`", "")

	// Remove list markers
	text = strings.TrimPrefix(text, "- ")
	text = strings.TrimPrefix(text, "* ")
	text = strings.TrimPrefix(text, "+ ")

	// Normalize whitespace
	spaceRegex := regexp.MustCompile(`\s+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// generateEmbeddingsForStatements generates embeddings for statements using the embedding client
func (s *Server) generateEmbeddingsForStatements(ctx context.Context, statements []*storage.Statement) error {
	if s.embeddingClient == nil {
		// If no embedding client, store statements without embeddings
		return nil
	}

	if len(statements) == 0 {
		return nil
	}

	// Extract texts
	texts := make([]string, len(statements))
	for i, stmt := range statements {
		texts[i] = stmt.Text
	}

	// Generate embeddings
	embeddings, err := s.embeddingClient.EmbedTexts(ctx, texts)
	if err != nil {
		return err
	}

	// Assign embeddings to statements
	for i, emb := range embeddings {
		statements[i].Embedding = pgvector.NewVector(emb)
	}

	return nil
}
