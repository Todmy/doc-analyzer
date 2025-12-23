"""CLI: Command-line interface for doc-analyzer."""

import asyncio
from pathlib import Path
from typing import Optional, TYPE_CHECKING

import typer
from rich.console import Console
from rich.progress import Progress, SpinnerColumn, TextColumn, BarColumn

from . import __version__
from .config import Config, init_config, show_config, ensure_api_key

# Lazy imports for heavy modules (sklearn, numpy) - only loaded when needed
# This makes `doc-analyzer --help` fast
if TYPE_CHECKING:
    from .analyzer import analyze_pairs, test_claude_cli
    from .anomaly import detect_anomalies
    from .cache import clear_cache, embed_with_cache, embed_with_cache_async, get_cache_stats
    from .clusterer import cluster_statements, get_cluster_keywords
    from .embedder import embed_statements, embed_statements_async, close_async_client, test_connection
    from .models import AnalysisReport
    from .parser import parse_documents, get_file_stats
    from .reporter import generate_report, save_report
    from .similarity import find_similar_pairs
    from .stats import calculate_stats, format_stats_summary


_modules_loaded = False

def _import_heavy_modules():
    """Import heavy modules lazily with loading indicator."""
    global _modules_loaded
    global analyze_pairs, test_claude_cli, detect_anomalies
    global clear_cache, embed_with_cache, embed_with_cache_async, get_cache_stats
    global cluster_statements, get_cluster_keywords
    global embed_statements, embed_statements_async, close_async_client, test_connection
    global AnalysisReport, parse_documents, get_file_stats
    global generate_report, save_report, find_similar_pairs
    global calculate_stats, format_stats_summary

    if _modules_loaded:
        return

    with console.status("[dim]Loading...[/dim]"):
        from .analyzer import analyze_pairs, test_claude_cli
        from .anomaly import detect_anomalies
        from .cache import clear_cache, embed_with_cache, embed_with_cache_async, get_cache_stats
        from .clusterer import cluster_statements, get_cluster_keywords
        from .embedder import embed_statements, embed_statements_async, close_async_client, test_connection
        from .models import AnalysisReport
        from .parser import parse_documents, get_file_stats
        from .reporter import generate_report, save_report
        from .similarity import find_similar_pairs
        from .stats import calculate_stats, format_stats_summary

    _modules_loaded = True


def _import_cache_only():
    """Import only cache module (no sklearn/numpy needed)."""
    global clear_cache, get_cache_stats
    # These functions only use pathlib/json - no heavy deps
    from .cache import clear_cache, get_cache_stats

app = typer.Typer(
    name="doc-analyzer",
    help="Semantic document analysis: contradictions, anomalies, and statistics",
    add_completion=False,
)
config_app = typer.Typer(help="Configuration management")
app.add_typer(config_app, name="config")

console = Console()


@app.command()
def analyze(
    path: Optional[Path] = typer.Argument(None, help="Path to documents (default: from config)"),
    output: Optional[Path] = typer.Option(None, "--output", "-o", help="Output file path"),
    format: str = typer.Option("markdown", "--format", "-f", help="Output format (markdown/json)"),
    threshold: float = typer.Option(0.75, "--threshold", "-t", help="Similarity threshold"),
    max_pairs: int = typer.Option(100, "--max-pairs", "-m", help="Max pairs to analyze"),
    no_contradictions: bool = typer.Option(False, "--no-contradictions", help="Skip contradiction analysis"),
    no_anomalies: bool = typer.Option(False, "--no-anomalies", help="Skip anomaly detection"),
    dry_run: bool = typer.Option(False, "--dry-run", help="Embeddings + clusters only, no LLM"),
    use_async: bool = typer.Option(True, "--async/--sync", help="Use async parallel API calls (faster)"),
    max_concurrent: int = typer.Option(5, "--max-concurrent", help="Max concurrent API requests (async mode)"),
    verbose: bool = typer.Option(False, "--verbose", "-v", help="Verbose output"),
    config_file: Optional[Path] = typer.Option(None, "--config", "-c", help="Config file path"),
):
    """Run full document analysis."""
    _import_heavy_modules()
    config = Config.load(config_file)
    config = ensure_api_key(config)
    config.analysis.similarity_threshold = threshold
    config.analysis.max_pairs_to_analyze = max_pairs

    # Use path from config if not provided
    docs_path = path or Path(config.documents.path)
    extensions = tuple(config.documents.extensions)

    # Parse documents
    console.print("[dim]Parsing documents...[/dim]", end="\r")
    statements = parse_documents(docs_path, config.analysis.min_statement_length, extensions)

    if not statements:
        console.print("[red]No statements found in documents[/red]")
        raise typer.Exit(1)

    console.print(f"[green]✓[/green] Parsed {len(statements)} statements     ")

    # Generate embeddings
    console.print("[dim]Generating embeddings...[/dim]")
    if use_async:
        async def run_async_embeddings():
            async def embed_fn_async(stmts):
                return await embed_statements_async(stmts, config, max_concurrent=max_concurrent)
            result = await embed_with_cache_async(
                statements, embed_fn_async, config.openrouter.embedding_model,
            )
            await close_async_client()
            return result
        embeddings = asyncio.run(run_async_embeddings())
    else:
        def embed_fn(stmts):
            return embed_statements(stmts, config)
        embeddings = embed_with_cache(
            statements, embed_fn, config.openrouter.embedding_model,
        )
    console.print(f"[green]✓[/green] Generated embeddings            ")

    # Cluster
    console.print("[dim]Clustering topics...[/dim]", end="\r")
    clusters = cluster_statements(embeddings)
    console.print(f"[green]✓[/green] Found {clusters.n_clusters} clusters           ")

    # Find similar pairs
    console.print("[dim]Finding similar pairs...[/dim]", end="\r")
    pairs = find_similar_pairs(
        embeddings, statements,
        threshold=threshold,
        skip_same_file=config.analysis.skip_same_file,
        max_pairs=max_pairs,
    )
    console.print(f"[green]✓[/green] Found {len(pairs)} similar pairs       ")

    # Analyze contradictions
    contradictions = []
    if not no_contradictions and not dry_run and pairs:
        console.print(f"[dim]Analyzing {len(pairs)} pairs with Claude...[/dim]")
        contradictions = analyze_pairs(pairs, statements, config)
        console.print(f"[green]✓[/green] Found {len(contradictions)} contradictions")

    # Detect anomalies
    anomalies = []
    if not no_anomalies and not dry_run:
        console.print("[dim]Detecting anomalies...[/dim]", end="\r")
        anomalies = detect_anomalies(embeddings, statements, clusters, config.anomaly)
        console.print(f"[green]✓[/green] Found {len(anomalies)} anomalies          ")

    # Calculate statistics
    console.print("[dim]Calculating statistics...[/dim]", end="\r")
    statistics = calculate_stats(statements, embeddings, clusters)
    console.print("[green]✓[/green] Statistics calculated            ")

    # Generate report
    report = AnalysisReport(
        statements=statements,
        clusters=clusters,
        contradictions=contradictions,
        anomalies=anomalies,
        statistics=statistics,
    )

    report_content = generate_report(report, format, config.output.group_by)

    # Output
    if output:
        save_report(report_content, output)
        console.print(f"[green]Report saved to {output}[/green]")
    else:
        console.print(report_content)


@app.command()
def contradictions(
    path: Optional[Path] = typer.Argument(None, help="Path to documents (default: from config)"),
    threshold: float = typer.Option(0.75, "--threshold", "-t"),
    max_pairs: int = typer.Option(100, "--max-pairs", "-m"),
    output: Optional[Path] = typer.Option(None, "--output", "-o"),
    config_file: Optional[Path] = typer.Option(None, "--config", "-c"),
):
    """Find contradictions only."""
    _import_heavy_modules()
    config = Config.load(config_file)
    config = ensure_api_key(config)
    docs_path = path or Path(config.documents.path)
    extensions = tuple(config.documents.extensions)

    # Parse documents
    console.print("[dim]Parsing documents...[/dim]", end="\r")
    statements = parse_documents(docs_path, config.analysis.min_statement_length, extensions)

    if not statements:
        console.print("[red]No statements found in documents[/red]")
        raise typer.Exit(1)

    console.print(f"[green]✓[/green] Parsed {len(statements)} statements     ")

    # Generate embeddings
    console.print("[dim]Generating embeddings...[/dim]")
    embeddings = embed_statements(statements, config)
    console.print(f"[green]✓[/green] Generated embeddings            ")

    # Find similar pairs
    console.print("[dim]Finding similar pairs...[/dim]", end="\r")
    pairs = find_similar_pairs(embeddings, statements, threshold, max_pairs=max_pairs)
    console.print(f"[green]✓[/green] Found {len(pairs)} similar pairs       ")

    # Analyze contradictions
    if pairs:
        console.print(f"[dim]Analyzing {len(pairs)} pairs with Claude...[/dim]")
        results = analyze_pairs(pairs, statements, config)
    else:
        results = []

    console.print(f"\n[bold]Found {len(results)} contradictions[/bold]\n")

    for i, c in enumerate(results, 1):
        console.print(f"[yellow]{i}. {c.severity.value.upper()}[/yellow] ({c.confidence:.0%} confidence)")
        console.print(f"   A: {c.statement_a.source_file.name}:{c.statement_a.line_number}")
        console.print(f"   B: {c.statement_b.source_file.name}:{c.statement_b.line_number}")
        console.print(f"   → {c.explanation}")
        console.print()


@app.command()
def anomalies(
    path: Optional[Path] = typer.Argument(None, help="Path to documents (default: from config)"),
    method: str = typer.Option(None, "--method", "-m", help="Detection method: ensemble, isolation_forest, lof, hdbscan"),
    output: Optional[Path] = typer.Option(None, "--output", "-o"),
    config_file: Optional[Path] = typer.Option(None, "--config", "-c"),
):
    """Detect anomalies using hybrid ensemble approach."""
    _import_heavy_modules()
    config = Config.load(config_file)
    config = ensure_api_key(config)
    docs_path = path or Path(config.documents.path)
    extensions = tuple(config.documents.extensions)

    # Override method if specified
    if method:
        config.anomaly.method = method

    # Parse documents first (fast operation)
    # Parse documents
    console.print("[dim]Parsing documents...[/dim]", end="\r")
    statements = parse_documents(docs_path, config.analysis.min_statement_length, extensions)

    if not statements:
        console.print("[red]No statements found in documents[/red]")
        raise typer.Exit(1)

    console.print(f"[green]✓[/green] Parsed {len(statements)} statements     ")

    # Generate embeddings
    console.print("[dim]Generating embeddings...[/dim]")
    embeddings = embed_statements(statements, config)
    console.print(f"[green]✓[/green] Generated embeddings            ")

    # Cluster
    console.print("[dim]Clustering...[/dim]", end="\r")
    clusters = cluster_statements(embeddings)
    console.print(f"[green]✓[/green] Found {clusters.n_clusters} clusters           ")

    # Detect anomalies
    console.print(f"[dim]Detecting anomalies ({config.anomaly.method})...[/dim]", end="\r")
    results = detect_anomalies(embeddings, statements, clusters, config.anomaly)

    console.print(f"\n[bold]Found {len(results)} anomalies[/bold] (method: {config.anomaly.method})\n")

    for i, a in enumerate(results, 1):
        # Color based on severity
        methods_str = ", ".join(a.methods_flagged)
        if len(a.methods_flagged) >= 2:
            color = "red"
        else:
            color = "yellow"

        console.print(f"[{color}]{i}.[/{color}] {a.statement.source_file.name}:{a.statement.line_number}")
        console.print(f"   \"{a.statement.text[:80]}...\"")
        console.print(f"   Score: {a.score:.3f} | Methods: {methods_str}")
        console.print(f"   IF: {a.scores.isolation_forest:.3f} | LOF: {a.scores.lof:.3f} | HDBSCAN: {a.scores.hdbscan:.0f}")
        console.print(f"   Reason: {a.reason}")
        console.print()


@app.command()
def stats(
    path: Optional[Path] = typer.Argument(None, help="Path to documents (default: from config)"),
    output: Optional[Path] = typer.Option(None, "--output", "-o"),
    config_file: Optional[Path] = typer.Option(None, "--config", "-c"),
):
    """Show statistics only."""
    _import_heavy_modules()
    config = Config.load(config_file)
    config = ensure_api_key(config)
    docs_path = path or Path(config.documents.path)
    extensions = tuple(config.documents.extensions)

    # Parse documents first (fast operation)
    # Parse documents
    console.print("[dim]Parsing documents...[/dim]", end="\r")
    statements = parse_documents(docs_path, config.analysis.min_statement_length, extensions)

    if not statements:
        console.print("[red]No statements found in documents[/red]")
        raise typer.Exit(1)

    console.print(f"[green]✓[/green] Parsed {len(statements)} statements     ")

    # Generate embeddings
    console.print("[dim]Generating embeddings...[/dim]")
    embeddings = embed_statements(statements, config)
    console.print(f"[green]✓[/green] Generated embeddings            ")

    # Cluster
    console.print("[dim]Clustering topics...[/dim]", end="\r")
    clusters = cluster_statements(embeddings)
    console.print(f"[green]✓[/green] Found {clusters.n_clusters} clusters           ")

    # Calculate statistics
    console.print("[dim]Calculating statistics...[/dim]", end="\r")
    statistics = calculate_stats(statements, embeddings, clusters)
    console.print("[green]✓[/green] Statistics calculated            ")

    summary = format_stats_summary(statistics)
    console.print(summary)


@app.command()
def clusters(
    path: Optional[Path] = typer.Argument(None, help="Path to documents (default: from config)"),
    show_samples: bool = typer.Option(False, "--samples", "-s", help="Show sample statements"),
    config_file: Optional[Path] = typer.Option(None, "--config", "-c"),
):
    """Show topic clusters."""
    _import_heavy_modules()
    config = Config.load(config_file)
    config = ensure_api_key(config)
    docs_path = path or Path(config.documents.path)
    extensions = tuple(config.documents.extensions)

    # Parse documents
    console.print("[dim]Parsing documents...[/dim]", end="\r")
    statements = parse_documents(docs_path, config.analysis.min_statement_length, extensions)

    if not statements:
        console.print("[red]No statements found in documents[/red]")
        raise typer.Exit(1)

    console.print(f"[green]✓[/green] Parsed {len(statements)} statements     ")

    # Generate embeddings
    console.print("[dim]Generating embeddings...[/dim]")
    embeddings = embed_statements(statements, config)
    console.print(f"[green]✓[/green] Generated embeddings            ")

    # Cluster
    console.print("[dim]Clustering topics...[/dim]", end="\r")
    clusters = cluster_statements(embeddings)

    console.print(f"\n[bold]Found {clusters.n_clusters} topic clusters[/bold]\n")

    sizes = clusters.get_cluster_sizes()
    for cluster_id in sorted(sizes.keys()):
        if cluster_id == -1:
            label = "Noise"
        else:
            keywords = get_cluster_keywords(statements, clusters, cluster_id)
            label = ", ".join(keywords) if keywords else f"Cluster {cluster_id}"

        console.print(f"[cyan]{label}[/cyan] ({sizes[cluster_id]} statements)")

        if show_samples:
            indices = clusters.get_cluster_indices(cluster_id)[:3]
            for idx in indices:
                text = statements[idx].text[:60]
                console.print(f"  • \"{text}...\"")

        console.print()


# Config subcommands
@config_app.command("init")
def config_init():
    """Initialize default config file."""
    path = init_config()
    console.print(f"[green]Config initialized at {path}[/green]")


@config_app.command("show")
def config_show(
    config_file: Optional[Path] = typer.Option(None, "--config", "-c"),
):
    """Show current configuration."""
    config = Config.load(config_file)
    console.print(show_config(config))


@config_app.command("set")
def config_set(
    key: str = typer.Argument(..., help="Config key (e.g., openrouter.api_key)"),
    value: str = typer.Argument(..., help="Value to set"),
    config_file: Optional[Path] = typer.Option(None, "--config", "-c"),
):
    """Set a config value."""
    config = Config.load(config_file)
    try:
        config.set(key, value)
        config.save(config_file)
        console.print(f"[green]Set {key} = {value}[/green]")
    except (KeyError, AttributeError) as e:
        console.print(f"[red]Invalid key: {key}[/red]")
        raise typer.Exit(1)


@config_app.command("test")
def config_test(
    config_file: Optional[Path] = typer.Option(None, "--config", "-c"),
):
    """Test API connections."""
    _import_heavy_modules()
    config = Config.load(config_file)

    # Test OpenRouter
    console.print("Testing OpenRouter API...", end=" ")
    if test_connection(config):
        console.print("[green]OK[/green]")
    else:
        console.print("[red]FAILED[/red]")

    # Test Claude CLI
    console.print("Testing Claude CLI...", end=" ")
    if test_claude_cli(config):
        console.print("[green]OK[/green]")
    else:
        console.print("[red]FAILED[/red]")


# Cache commands
@app.command("cache-clear")
def cache_clear():
    """Clear embedding cache."""
    _import_cache_only()
    deleted = clear_cache()
    console.print(f"[green]Cleared {deleted} cached embeddings[/green]")


@app.command("cache-stats")
def cache_stats():
    """Show cache statistics."""
    _import_cache_only()
    stats = get_cache_stats()
    console.print(f"Cache directory: {stats['cache_dir']}")
    console.print(f"Total entries: {stats['total_entries']}")
    console.print(f"Total size: {stats['total_size_kb']:.2f} KB")


@app.command()
def version():
    """Show version."""
    console.print(f"doc-analyzer {__version__}")


@app.command("interactive")
def interactive_mode(
    path: Path = typer.Argument(..., help="Path to documents"),
    config_file: Optional[Path] = typer.Option(None, "--config", "-c"),
):
    """Interactive mode: load once, run many commands quickly."""
    _import_heavy_modules()
    config = Config.load(config_file)
    config = ensure_api_key(config)
    docs_path = path
    extensions = tuple(config.documents.extensions)

    # Initial load
    console.print(f"\n[bold]Interactive Mode[/bold] - {docs_path}\n")

    console.print("[dim]Parsing documents...[/dim]")
    statements = parse_documents(docs_path, config.analysis.min_statement_length, extensions)

    if not statements:
        console.print("[red]No statements found in documents[/red]")
        raise typer.Exit(1)

    console.print(f"[dim]Found {len(statements)} statements[/dim]")

    # Generate embeddings with progress
    with Progress(
        SpinnerColumn(),
        TextColumn("[progress.description]{task.description}"),
        BarColumn(),
        TextColumn("{task.completed}/{task.total}"),
        console=console,
        transient=False,
    ) as progress:
        task = progress.add_task("Generating embeddings...", total=len(statements))

        async def embed_fn_async(stmts):
            return await embed_statements_async(stmts, config, progress=progress, task_id=task)

        embeddings = asyncio.run(embed_with_cache_async(
            statements, embed_fn_async, config.openrouter.embedding_model,
            progress=progress, task_id=task
        ))
        asyncio.run(close_async_client())

    console.print("[dim]Clustering...[/dim]")
    clusters = cluster_statements(embeddings)

    console.print(f"\n[green]Ready![/green] {len(statements)} statements, {clusters.n_clusters} clusters\n")

    # REPL
    commands_help = """[bold]Commands:[/bold]
  [cyan]stats[/cyan]           Show document statistics
  [cyan]clusters[/cyan]        Show topic clusters
  [cyan]anomalies[/cyan]       Detect anomalies
  [cyan]contradictions[/cyan]  Find contradictions (uses Claude CLI)
  [cyan]analyze[/cyan]         Run full analysis
  [cyan]help[/cyan]            Show this help
  [cyan]exit[/cyan]            Exit interactive mode
"""
    console.print(commands_help)

    while True:
        try:
            cmd = console.input("[bold blue]>[/bold blue] ").strip().lower()
        except (KeyboardInterrupt, EOFError):
            console.print("\n[dim]Goodbye![/dim]")
            break

        if not cmd:
            continue

        if cmd in ("exit", "quit", "q"):
            console.print("[dim]Goodbye![/dim]")
            break

        elif cmd == "help":
            console.print(commands_help)

        elif cmd == "stats":
            statistics = calculate_stats(statements, embeddings, clusters)
            console.print(format_stats_summary(statistics))

        elif cmd == "clusters":
            console.print(f"\n[bold]Found {clusters.n_clusters} topic clusters[/bold]\n")
            sizes = clusters.get_cluster_sizes()
            for cluster_id in sorted(sizes.keys()):
                if cluster_id == -1:
                    label = "Noise"
                else:
                    keywords = get_cluster_keywords(statements, clusters, cluster_id)
                    label = ", ".join(keywords) if keywords else f"Cluster {cluster_id}"
                console.print(f"[cyan]{label}[/cyan] ({sizes[cluster_id]} statements)")

        elif cmd == "anomalies":
            with console.status("Detecting anomalies..."):
                results = detect_anomalies(embeddings, statements, clusters, config.anomaly)
            console.print(f"\n[bold]Found {len(results)} anomalies[/bold]\n")
            for i, a in enumerate(results[:10], 1):  # Show top 10
                console.print(f"{i}. {a.statement.source_file.name}:{a.statement.line_number}")
                console.print(f"   \"{a.statement.text[:60]}...\"")
                console.print(f"   Score: {a.score:.3f} | {a.reason}\n")
            if len(results) > 10:
                console.print(f"[dim]...and {len(results) - 10} more[/dim]")

        elif cmd == "contradictions":
            with console.status("Finding similar pairs..."):
                pairs = find_similar_pairs(
                    embeddings, statements,
                    threshold=config.analysis.similarity_threshold,
                    max_pairs=config.analysis.max_pairs_to_analyze
                )
            if not pairs:
                console.print("[yellow]No similar pairs found[/yellow]")
                continue
            console.print(f"[dim]Analyzing {len(pairs)} pairs with Claude...[/dim]")
            with Progress(SpinnerColumn(), TextColumn("{task.description}"), console=console) as progress:
                task = progress.add_task(f"Analyzing...", total=len(pairs))
                results = analyze_pairs(pairs, statements, config, progress, task)
            console.print(f"\n[bold]Found {len(results)} contradictions[/bold]\n")
            for i, c in enumerate(results[:10], 1):
                console.print(f"[yellow]{i}. {c.severity.value.upper()}[/yellow] ({c.confidence:.0%})")
                console.print(f"   {c.explanation}\n")

        elif cmd == "analyze":
            console.print("[dim]Running full analysis...[/dim]")
            # Similar pairs
            pairs = find_similar_pairs(
                embeddings, statements,
                threshold=config.analysis.similarity_threshold,
                max_pairs=config.analysis.max_pairs_to_analyze
            )
            # Contradictions
            contradictions = []
            if pairs:
                with Progress(SpinnerColumn(), TextColumn("{task.description}"), console=console) as progress:
                    task = progress.add_task("Analyzing contradictions...", total=len(pairs))
                    contradictions = analyze_pairs(pairs, statements, config, progress, task)
            # Anomalies
            anomalies = detect_anomalies(embeddings, statements, clusters, config.anomaly)
            # Stats
            statistics = calculate_stats(statements, embeddings, clusters)
            # Report
            report = AnalysisReport(
                statements=statements, clusters=clusters,
                contradictions=contradictions, anomalies=anomalies, statistics=statistics
            )
            console.print(generate_report(report, "markdown", config.output.group_by))

        else:
            console.print(f"[red]Unknown command:[/red] {cmd}")
            console.print("[dim]Type 'help' for available commands[/dim]")


def main():
    """Entry point."""
    app()


if __name__ == "__main__":
    main()
