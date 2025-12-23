"""Document parser: extracts statements from markdown, text, and JSON files."""

import json
import re
from pathlib import Path

from .models import Statement


def parse_documents(
    path: str | Path,
    min_length: int = 50,
    extensions: tuple[str, ...] = (".md", ".txt", ".json"),
) -> list[Statement]:
    """Parse all documents in path and extract statements.

    Args:
        path: File or directory path
        min_length: Minimum statement length (chars)
        extensions: File extensions to process

    Returns:
        List of Statement objects
    """
    path = Path(path)
    statements: list[Statement] = []

    if path.is_file():
        files = [path]
    else:
        files = [f for f in path.rglob("*") if f.suffix in extensions]

    for file_path in sorted(files):
        if file_path.suffix == ".md":
            statements.extend(_parse_markdown(file_path, min_length))
        elif file_path.suffix == ".txt":
            statements.extend(_parse_text(file_path, min_length))
        elif file_path.suffix == ".json":
            statements.extend(_parse_json(file_path, min_length))

    return statements


def _parse_text(file_path: Path, min_length: int) -> list[Statement]:
    """Parse plain text file into statements.

    Strategy:
    - Split by blank lines (paragraphs)
    - Skip lines that are too short
    """
    statements: list[Statement] = []

    with open(file_path, encoding="utf-8") as f:
        content = f.read()

    # Split into lines with line numbers
    lines = content.split("\n")

    # Group into paragraphs
    paragraphs = _group_into_paragraphs(lines)

    for para_text, start_line in paragraphs:
        # Clean up whitespace
        clean_text = " ".join(para_text.split())

        if len(clean_text) < min_length:
            continue

        statements.append(
            Statement(
                text=clean_text,
                source_file=file_path,
                line_number=start_line,
                context=para_text[:200] if len(para_text) > 200 else para_text,
            )
        )

    return statements


def _parse_markdown(file_path: Path, min_length: int) -> list[Statement]:
    """Parse markdown file into statements.

    Strategy:
    - Split by blank lines (paragraphs)
    - Skip code blocks
    - Skip headers-only lines
    - Skip lines that are too short
    """
    statements: list[Statement] = []

    with open(file_path, encoding="utf-8") as f:
        content = f.read()

    # Remove code blocks (both fenced and indented)
    content = _remove_code_blocks(content)

    # Split into lines with line numbers
    lines = content.split("\n")

    # Group into paragraphs
    paragraphs = _group_into_paragraphs(lines)

    for para_text, start_line in paragraphs:
        # Clean up the paragraph
        clean_text = _clean_paragraph(para_text)

        # Skip if too short or header-only
        if len(clean_text) < min_length:
            continue
        if _is_header_only(clean_text):
            continue
        if _is_table_or_list_structure(clean_text):
            continue

        statements.append(
            Statement(
                text=clean_text,
                source_file=file_path,
                line_number=start_line,
                context=para_text[:200] if len(para_text) > 200 else para_text,
            )
        )

    return statements


def _parse_json(file_path: Path, min_length: int) -> list[Statement]:
    """Parse JSON file into statements.

    Strategy:
    - Extract all string values recursively
    - Track path for context
    """
    statements: list[Statement] = []

    with open(file_path, encoding="utf-8") as f:
        try:
            data = json.load(f)
        except json.JSONDecodeError:
            return statements

    # Extract strings recursively
    strings = _extract_json_strings(data, [])

    for text, json_path in strings:
        if len(text) < min_length:
            continue

        statements.append(
            Statement(
                text=text,
                source_file=file_path,
                line_number=1,  # JSON doesn't have meaningful line numbers
                context=".".join(json_path),
            )
        )

    return statements


def _remove_code_blocks(content: str) -> str:
    """Remove fenced code blocks from markdown."""
    # Remove fenced code blocks (```...```)
    content = re.sub(r"```[\s\S]*?```", "", content)
    # Remove inline code (`...`)
    content = re.sub(r"`[^`]+`", "", content)
    return content


def _group_into_paragraphs(lines: list[str]) -> list[tuple[str, int]]:
    """Group lines into paragraphs with start line numbers.

    Returns list of (paragraph_text, start_line_number)
    """
    paragraphs: list[tuple[str, int]] = []
    current_para: list[str] = []
    start_line = 1

    for i, line in enumerate(lines, start=1):
        stripped = line.strip()

        if not stripped:
            # Blank line - end of paragraph
            if current_para:
                para_text = "\n".join(current_para)
                paragraphs.append((para_text, start_line))
                current_para = []
        else:
            if not current_para:
                start_line = i
            current_para.append(stripped)

    # Don't forget last paragraph
    if current_para:
        para_text = "\n".join(current_para)
        paragraphs.append((para_text, start_line))

    return paragraphs


def _clean_paragraph(text: str) -> str:
    """Clean up paragraph text."""
    # Remove markdown formatting
    text = re.sub(r"\*\*([^*]+)\*\*", r"\1", text)  # Bold
    text = re.sub(r"\*([^*]+)\*", r"\1", text)  # Italic
    text = re.sub(r"__([^_]+)__", r"\1", text)  # Bold
    text = re.sub(r"_([^_]+)_", r"\1", text)  # Italic

    # Remove links but keep text
    text = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", text)

    # Remove images
    text = re.sub(r"!\[([^\]]*)\]\([^)]+\)", "", text)

    # Remove HTML tags
    text = re.sub(r"<[^>]+>", "", text)

    # Normalize whitespace
    text = " ".join(text.split())

    return text.strip()


def _is_header_only(text: str) -> bool:
    """Check if text is just a header."""
    # Check for markdown headers
    if re.match(r"^#{1,6}\s+.+$", text):
        return True
    return False


def _is_table_or_list_structure(text: str) -> bool:
    """Check if text is primarily table or list structure."""
    lines = text.split("\n")
    if not lines:
        return False

    # Check for table indicators
    table_lines = sum(1 for line in lines if "|" in line)
    if table_lines / len(lines) > 0.5:
        return True

    # Check for list-only content
    list_lines = sum(1 for line in lines if re.match(r"^\s*[-*+]\s+", line))
    if list_lines / len(lines) > 0.8:
        return True

    return False


def _extract_json_strings(
    obj: object,
    path: list[str],
) -> list[tuple[str, list[str]]]:
    """Recursively extract strings from JSON object.

    Returns list of (string_value, json_path)
    """
    results: list[tuple[str, list[str]]] = []

    if isinstance(obj, str):
        # Clean up the string
        clean = obj.strip()
        if clean:
            results.append((clean, path.copy()))

    elif isinstance(obj, dict):
        for key, value in obj.items():
            results.extend(_extract_json_strings(value, path + [key]))

    elif isinstance(obj, list):
        for i, item in enumerate(obj):
            results.extend(_extract_json_strings(item, path + [str(i)]))

    return results


def get_file_stats(path: str | Path) -> dict:
    """Get statistics about files in path."""
    path = Path(path)

    if path.is_file():
        files = [path]
    else:
        files = list(path.rglob("*"))

    stats = {
        "total_files": 0,
        "md_files": 0,
        "json_files": 0,
        "other_files": 0,
        "total_size_kb": 0,
    }

    for f in files:
        if not f.is_file():
            continue

        stats["total_files"] += 1
        stats["total_size_kb"] += f.stat().st_size / 1024

        if f.suffix == ".md":
            stats["md_files"] += 1
        elif f.suffix == ".json":
            stats["json_files"] += 1
        else:
            stats["other_files"] += 1

    return stats
