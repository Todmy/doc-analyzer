package clustering

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// KeywordExtractor extracts keywords from text using TF-IDF
type KeywordExtractor struct {
	stopWords map[string]bool
	minLength int
}

// NewKeywordExtractor creates a new keyword extractor
func NewKeywordExtractor() *KeywordExtractor {
	return &KeywordExtractor{
		stopWords: defaultStopWords(),
		minLength: 3,
	}
}

// Keyword represents a keyword with its TF-IDF score
type Keyword struct {
	Word  string
	Score float64
}

// ExtractKeywords extracts top-k keywords from texts using TF-IDF
func (ke *KeywordExtractor) ExtractKeywords(texts []string, topK int) []Keyword {
	if len(texts) == 0 {
		return []Keyword{}
	}

	// Tokenize all documents
	docs := make([][]string, len(texts))
	for i, text := range texts {
		docs[i] = ke.tokenize(text)
	}

	// Compute TF-IDF scores
	tfidf := ke.computeTFIDF(docs)

	// Sort by score
	keywords := make([]Keyword, 0, len(tfidf))
	for word, score := range tfidf {
		keywords = append(keywords, Keyword{Word: word, Score: score})
	}

	sort.Slice(keywords, func(i, j int) bool {
		return keywords[i].Score > keywords[j].Score
	})

	// Return top-k
	if topK > 0 && topK < len(keywords) {
		keywords = keywords[:topK]
	}

	return keywords
}

// ExtractClusterKeywords extracts keywords for each cluster
func (ke *KeywordExtractor) ExtractClusterKeywords(texts []string, labels []int, numClusters int, topK int) map[int][]Keyword {
	if len(texts) != len(labels) {
		return nil
	}

	// Group texts by cluster
	clusterTexts := make(map[int][]string)
	for i, label := range labels {
		clusterTexts[label] = append(clusterTexts[label], texts[i])
	}

	// Extract keywords for each cluster
	result := make(map[int][]Keyword)
	for cluster, cTexts := range clusterTexts {
		result[cluster] = ke.ExtractKeywords(cTexts, topK)
	}

	return result
}

func (ke *KeywordExtractor) tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Split into words
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	// Filter stop words and short words
	result := make([]string, 0)
	for _, word := range words {
		if len(word) >= ke.minLength && !ke.stopWords[word] {
			result = append(result, word)
		}
	}

	return result
}

func (ke *KeywordExtractor) computeTFIDF(docs [][]string) map[string]float64 {
	n := len(docs)
	if n == 0 {
		return nil
	}

	// Compute document frequency for each term
	df := make(map[string]int)
	for _, doc := range docs {
		seen := make(map[string]bool)
		for _, word := range doc {
			if !seen[word] {
				df[word]++
				seen[word] = true
			}
		}
	}

	// Compute TF-IDF for each term across all documents
	tfidf := make(map[string]float64)
	for _, doc := range docs {
		// Term frequency in this document
		tf := make(map[string]int)
		for _, word := range doc {
			tf[word]++
		}

		// Add TF-IDF contribution from this document
		docLen := len(doc)
		if docLen == 0 {
			continue
		}

		for word, count := range tf {
			// TF: normalized by document length
			termFreq := float64(count) / float64(docLen)
			// IDF: log(N / df)
			idf := math.Log(float64(n) / float64(df[word]))
			tfidf[word] += termFreq * idf
		}
	}

	// Normalize by number of documents
	for word := range tfidf {
		tfidf[word] /= float64(n)
	}

	return tfidf
}

func defaultStopWords() map[string]bool {
	words := []string{
		"a", "an", "and", "are", "as", "at", "be", "by", "for", "from",
		"has", "have", "he", "in", "is", "it", "its", "of", "on", "or",
		"she", "that", "the", "they", "this", "to", "was", "were", "will",
		"with", "you", "your", "we", "our", "their", "them", "there", "these",
		"those", "been", "being", "had", "having", "do", "does", "did", "doing",
		"would", "could", "should", "may", "might", "must", "can", "cannot",
		"about", "above", "after", "again", "against", "all", "am", "any",
		"because", "before", "below", "between", "both", "but", "during",
		"each", "few", "further", "here", "how", "if", "into", "just", "more",
		"most", "no", "nor", "not", "now", "only", "other", "out", "own",
		"same", "so", "some", "such", "than", "then", "through", "too", "under",
		"until", "up", "very", "what", "when", "where", "which", "while", "who",
		"whom", "why", "also", "however", "therefore", "thus", "hence", "yet",
	}

	result := make(map[string]bool)
	for _, w := range words {
		result[w] = true
	}
	return result
}
