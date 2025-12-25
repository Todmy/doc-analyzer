package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
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

// extractStatements extracts statements from document content based on file extension
func extractStatements(content string, documentID uuid.UUID, ext string) []*storage.Statement {
	switch ext {
	case ".json":
		return extractStatementsFromJSON(content, documentID)
	case ".csv":
		return extractStatementsFromCSV(content, documentID)
	default:
		return extractStatementsFromText(content, documentID)
	}
}

// extractStatementsFromJSON extracts statements from JSON content
func extractStatementsFromJSON(content string, documentID uuid.UUID) []*storage.Statement {
	var statements []*storage.Statement
	var data interface{}

	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return statements
	}

	position := 0
	extractJSONStrings(data, documentID, &statements, &position)
	return statements
}

func extractJSONStrings(data interface{}, documentID uuid.UUID, statements *[]*storage.Statement, position *int) {
	switch v := data.(type) {
	case map[string]interface{}:
		for _, value := range v {
			extractJSONStrings(value, documentID, statements, position)
		}
	case []interface{}:
		for _, item := range v {
			extractJSONStrings(item, documentID, statements, position)
		}
	case string:
		text := strings.TrimSpace(v)
		if len(text) >= minStatementLength {
			if len(text) > maxStatementLength {
				text = text[:maxStatementLength] + "..."
			}
			*statements = append(*statements, &storage.Statement{
				DocumentID: documentID,
				Text:       text,
				Position:   *position,
				Line:       *position + 1,
				Embedding:  pgvector.NewVector(nil),
			})
			*position++
		}
	}
}

// extractStatementsFromCSV extracts statements from CSV content
func extractStatementsFromCSV(content string, documentID uuid.UUID) []*storage.Statement {
	var statements []*storage.Statement
	reader := csv.NewReader(strings.NewReader(content))

	records, err := reader.ReadAll()
	if err != nil {
		return statements
	}

	position := 0
	for lineNum, record := range records {
		// Combine all fields in the row
		rowText := strings.Join(record, " ")
		rowText = strings.TrimSpace(rowText)

		if len(rowText) >= minStatementLength {
			if len(rowText) > maxStatementLength {
				rowText = rowText[:maxStatementLength] + "..."
			}
			statements = append(statements, &storage.Statement{
				DocumentID: documentID,
				Text:       rowText,
				Position:   position,
				Line:       lineNum + 1,
				Embedding:  pgvector.NewVector(nil),
			})
			position++
		}
	}

	return statements
}

// extractStatementsFromText extracts statements from markdown/text content
func extractStatementsFromText(content string, documentID uuid.UUID) []*storage.Statement {
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
