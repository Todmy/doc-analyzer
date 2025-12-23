"""Tests for embedding cache."""

import tempfile
from pathlib import Path

import numpy as np
import pytest

from doc_analyzer.cache import (
    get_cache_key,
    get_cached_embeddings,
    save_embeddings,
    clear_cache,
    get_cache_stats,
    embed_with_cache,
)
from doc_analyzer.models import Statement


@pytest.fixture
def sample_statements():
    """Create sample statements for testing."""
    return [
        Statement(text="First statement for testing", source_file=Path("a.md"), line_number=1),
        Statement(text="Second statement for testing", source_file=Path("b.md"), line_number=2),
        Statement(text="Third statement for testing", source_file=Path("c.md"), line_number=3),
    ]


@pytest.fixture
def sample_embeddings():
    """Create sample embeddings."""
    return np.array([
        [0.1, 0.2, 0.3],
        [0.4, 0.5, 0.6],
        [0.7, 0.8, 0.9],
    ])


class TestGetCacheKey:
    def test_generates_consistent_key(self):
        stmt = Statement(text="Test text", source_file=Path("test.md"), line_number=1)
        key1 = get_cache_key(stmt, "model-a")
        key2 = get_cache_key(stmt, "model-a")
        assert key1 == key2

    def test_different_models_different_keys(self):
        stmt = Statement(text="Test text", source_file=Path("test.md"), line_number=1)
        key1 = get_cache_key(stmt, "model-a")
        key2 = get_cache_key(stmt, "model-b")
        assert key1 != key2

    def test_different_text_different_keys(self):
        stmt1 = Statement(text="Text A", source_file=Path("test.md"), line_number=1)
        stmt2 = Statement(text="Text B", source_file=Path("test.md"), line_number=1)
        key1 = get_cache_key(stmt1, "model")
        key2 = get_cache_key(stmt2, "model")
        assert key1 != key2


class TestSaveAndGetCachedEmbeddings:
    def test_save_and_retrieve(self, sample_statements, sample_embeddings):
        model = "test-model"

        # Clear first
        clear_cache()

        # Save
        saved = save_embeddings(sample_statements, sample_embeddings, model)
        assert saved == 3

        # Retrieve
        cached, missing = get_cached_embeddings(sample_statements, model)
        assert len(missing) == 0
        assert cached is not None
        np.testing.assert_array_almost_equal(cached, sample_embeddings)

    def test_partial_cache_hit(self, sample_statements, sample_embeddings):
        model = "test-model"
        clear_cache()

        # Save only first 2
        save_embeddings(sample_statements[:2], sample_embeddings[:2], model)

        # Try to get all 3
        cached, missing = get_cached_embeddings(sample_statements, model)
        assert missing == [2]  # Third one is missing


class TestClearCache:
    def test_clears_all_entries(self, sample_statements, sample_embeddings):
        model = "test-model"

        # Save some
        save_embeddings(sample_statements, sample_embeddings, model)

        # Clear
        deleted = clear_cache()
        assert deleted >= 3

        # Verify cleared
        _, missing = get_cached_embeddings(sample_statements, model)
        assert len(missing) == 3


class TestGetCacheStats:
    def test_returns_stats_dict(self):
        stats = get_cache_stats()
        assert "total_entries" in stats
        assert "total_size_kb" in stats
        assert "cache_dir" in stats


class TestEmbedWithCache:
    def test_uses_cache(self, sample_statements, sample_embeddings):
        model = "test-model"
        clear_cache()

        embed_calls = []

        def mock_embed(stmts):
            embed_calls.append(stmts)
            return sample_embeddings[:len(stmts)]

        # First call - should call embed_fn
        result1 = embed_with_cache(sample_statements, mock_embed, model)
        assert len(embed_calls) == 1

        # Second call - should use cache
        result2 = embed_with_cache(sample_statements, mock_embed, model)
        assert len(embed_calls) == 1  # No additional calls

        np.testing.assert_array_almost_equal(result1, result2)
