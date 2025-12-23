"""Tests for document parser."""

import tempfile
from pathlib import Path

import pytest

from doc_analyzer.parser import parse_documents, get_file_stats
from doc_analyzer.models import Statement


class TestParseDocuments:
    def test_parses_markdown_paragraphs(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            md_file = Path(tmpdir) / "test.md"
            md_file.write_text("""# Header

This is the first paragraph with enough content to meet the minimum length requirement for testing purposes.

This is the second paragraph also with enough content to pass the filter and be included in results.
""")
            statements = parse_documents(Path(tmpdir), min_length=20, extensions=('.md',))
            assert len(statements) == 2
            assert "first paragraph" in statements[0].text
            assert "second paragraph" in statements[1].text

    def test_skips_short_paragraphs(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            md_file = Path(tmpdir) / "test.md"
            md_file.write_text("""# Header

Short.

This paragraph has enough content to meet the minimum length requirement for testing purposes and filtering.
""")
            statements = parse_documents(Path(tmpdir), min_length=50, extensions=('.md',))
            assert len(statements) == 1
            assert "enough content" in statements[0].text

    def test_skips_code_blocks(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            md_file = Path(tmpdir) / "test.md"
            md_file.write_text("""# Header

This is a normal paragraph with content that should be included in results.

```python
def hello():
    print("This should be skipped")
```

Another normal paragraph here with enough content for testing.
""")
            statements = parse_documents(Path(tmpdir), min_length=20, extensions=('.md',))
            # Should not include the code block
            for stmt in statements:
                assert "def hello" not in stmt.text
                assert "print" not in stmt.text

    def test_statement_has_metadata(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            md_file = Path(tmpdir) / "test.md"
            md_file.write_text("First paragraph with enough content for testing purposes and assertions.")
            statements = parse_documents(Path(tmpdir), min_length=20, extensions=('.md',))
            assert len(statements) == 1
            assert statements[0].source_file.name == "test.md"
            assert statements[0].line_number >= 1

    def test_parses_json_strings(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            json_file = Path(tmpdir) / "test.json"
            json_file.write_text('{"key": "This is a value with enough content to pass the filter for testing"}')
            statements = parse_documents(Path(tmpdir), min_length=20, extensions=('.json',))
            assert len(statements) >= 1
            assert any("enough content" in s.text for s in statements)

    def test_parses_nested_json(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            json_file = Path(tmpdir) / "test.json"
            json_file.write_text('{"outer": {"inner": "Nested value with sufficient content for testing purposes"}}')
            statements = parse_documents(Path(tmpdir), min_length=20, extensions=('.json',))
            assert any("Nested value" in s.text for s in statements)

    def test_filters_by_extension(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            md_file = Path(tmpdir) / "test.md"
            md_file.write_text("Markdown content with enough length for testing purposes and filtering.")

            txt_file = Path(tmpdir) / "test.txt"
            txt_file.write_text("Text content with enough length for testing purposes here and more.")

            # Only parse .md files
            statements = parse_documents(Path(tmpdir), min_length=20, extensions=('.md',))
            assert all(s.source_file.suffix == '.md' for s in statements)

    def test_handles_empty_directory(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            statements = parse_documents(Path(tmpdir), min_length=20, extensions=('.md',))
            assert statements == []


class TestGetFileStats:
    def test_returns_stats_dict(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            md_file = Path(tmpdir) / "test.md"
            md_file.write_text("Some content here.")

            stats = get_file_stats(tmpdir)
            assert "total_files" in stats
            assert stats["total_files"] >= 1
