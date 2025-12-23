"""Embedding cache: store and retrieve embeddings by content hash."""

import hashlib
import json
from pathlib import Path

import numpy as np

from .config import DEFAULT_CONFIG_DIR
from .models import Statement

CACHE_DIR = DEFAULT_CONFIG_DIR / "cache"


def get_cache_key(statement: Statement, model: str) -> str:
    """Generate cache key from statement content and model."""
    content = f"{model}:{statement.text}"
    return hashlib.sha256(content.encode()).hexdigest()[:16]


def get_cached_embeddings(
    statements: list[Statement],
    model: str,
) -> tuple[np.ndarray | None, list[int]]:
    """Get cached embeddings for statements.

    Returns:
        Tuple of (embeddings array or None, list of indices that were NOT found in cache)
    """
    CACHE_DIR.mkdir(parents=True, exist_ok=True)

    embeddings: list[list[float] | None] = [None] * len(statements)
    missing_indices: list[int] = []

    for i, stmt in enumerate(statements):
        cache_key = get_cache_key(stmt, model)
        cache_file = CACHE_DIR / f"{cache_key}.json"

        if cache_file.exists():
            try:
                with open(cache_file) as f:
                    data = json.load(f)
                embeddings[i] = data["embedding"]
            except (json.JSONDecodeError, KeyError):
                missing_indices.append(i)
        else:
            missing_indices.append(i)

    # If all found, return as array
    if not missing_indices:
        return np.array(embeddings), []

    # If none found, return None
    if len(missing_indices) == len(statements):
        return None, missing_indices

    # Partial cache hit - return what we have
    return embeddings, missing_indices  # type: ignore


def save_embeddings(
    statements: list[Statement],
    embeddings: np.ndarray,
    model: str,
    indices: list[int] | None = None,
) -> int:
    """Save embeddings to cache.

    Args:
        statements: List of statements
        embeddings: Embeddings array
        model: Model name used
        indices: Optional specific indices to save (default: all)

    Returns:
        Number of embeddings saved
    """
    CACHE_DIR.mkdir(parents=True, exist_ok=True)

    if indices is None:
        indices = list(range(len(statements)))

    saved = 0
    for i, idx in enumerate(indices):
        if idx >= len(statements) or i >= len(embeddings):
            continue

        stmt = statements[idx]
        embedding = embeddings[i].tolist() if isinstance(embeddings[i], np.ndarray) else embeddings[i]

        cache_key = get_cache_key(stmt, model)
        cache_file = CACHE_DIR / f"{cache_key}.json"

        data = {
            "text_hash": hashlib.sha256(stmt.text.encode()).hexdigest()[:16],
            "model": model,
            "embedding": embedding,
        }

        with open(cache_file, "w") as f:
            json.dump(data, f)

        saved += 1

    return saved


def clear_cache() -> int:
    """Clear all cached embeddings.

    Returns:
        Number of cache files deleted
    """
    if not CACHE_DIR.exists():
        return 0

    deleted = 0
    for cache_file in CACHE_DIR.glob("*.json"):
        cache_file.unlink()
        deleted += 1

    return deleted


def get_cache_stats() -> dict:
    """Get cache statistics."""
    if not CACHE_DIR.exists():
        return {
            "total_entries": 0,
            "total_size_kb": 0,
            "cache_dir": str(CACHE_DIR),
        }

    files = list(CACHE_DIR.glob("*.json"))
    total_size = sum(f.stat().st_size for f in files)

    return {
        "total_entries": len(files),
        "total_size_kb": round(total_size / 1024, 2),
        "cache_dir": str(CACHE_DIR),
    }


def embed_with_cache(
    statements: list[Statement],
    embed_fn,
    model: str,
    progress=None,
    task_id=None,
) -> np.ndarray:
    """Embed statements with caching.

    Args:
        statements: Statements to embed
        embed_fn: Function to call for uncached embeddings (takes list of Statement)
        model: Model name for cache key
        progress: Optional rich Progress
        task_id: Optional task ID

    Returns:
        Complete embeddings array
    """
    # Check cache
    cached, missing_indices = get_cached_embeddings(statements, model)

    if not missing_indices:
        # All cached
        if progress and task_id is not None:
            progress.update(task_id, advance=len(statements))
        return cached  # type: ignore

    # Get uncached statements
    uncached_statements = [statements[i] for i in missing_indices]

    # Embed uncached
    new_embeddings = embed_fn(uncached_statements)

    # Save to cache
    save_embeddings(uncached_statements, new_embeddings, model)

    # Merge results
    if cached is None:
        return new_embeddings

    # Fill in missing
    result = np.array(cached, dtype=object)
    for i, idx in enumerate(missing_indices):
        result[idx] = new_embeddings[i]

    return np.array(result.tolist())


async def embed_with_cache_async(
    statements: list[Statement],
    embed_fn,
    model: str,
    progress=None,
    task_id=None,
) -> np.ndarray:
    """Embed statements with caching (async version).

    Args:
        statements: Statements to embed
        embed_fn: Async function to call for uncached embeddings
        model: Model name for cache key
        progress: Optional rich Progress
        task_id: Optional task ID

    Returns:
        Complete embeddings array
    """
    # Check cache (sync - file I/O is fast)
    cached, missing_indices = get_cached_embeddings(statements, model)

    if not missing_indices:
        # All cached
        if progress and task_id is not None:
            progress.update(task_id, advance=len(statements))
        return cached  # type: ignore

    # Get uncached statements
    uncached_statements = [statements[i] for i in missing_indices]

    # Embed uncached (async)
    new_embeddings = await embed_fn(uncached_statements)

    # Save to cache (sync - fast)
    save_embeddings(uncached_statements, new_embeddings, model)

    # Merge results
    if cached is None:
        return new_embeddings

    # Fill in missing
    result = np.array(cached, dtype=object)
    for i, idx in enumerate(missing_indices):
        result[idx] = new_embeddings[i]

    return np.array(result.tolist())
