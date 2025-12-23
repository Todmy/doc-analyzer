# -*- mode: python ; coding: utf-8 -*-
"""PyInstaller spec file for doc-analyzer.

Creates a single-file executable with bundled config.
The config.yaml in the project root is embedded into the executable.

Build:
    pyinstaller doc_analyzer.spec

Output:
    dist/doc-analyzer (or dist/doc-analyzer.exe on Windows)
"""

import sys
from pathlib import Path

block_cipher = None

# Project root
PROJECT_ROOT = Path(SPECPATH)

# Data files to bundle (config.yaml embedded in executable)
datas = [
    (str(PROJECT_ROOT / 'config.yaml'), '.'),
]

a = Analysis(
    [str(PROJECT_ROOT / 'src' / 'doc_analyzer' / '__main__.py')],
    pathex=[str(PROJECT_ROOT / 'src')],
    binaries=[],
    datas=datas,
    hiddenimports=[
        'doc_analyzer',
        'doc_analyzer.cli',
        'doc_analyzer.config',
        'doc_analyzer.models',
        'doc_analyzer.parser',
        'doc_analyzer.embedder',
        'doc_analyzer.clusterer',
        'doc_analyzer.similarity',
        'doc_analyzer.anomaly',
        'doc_analyzer.stats',
        'doc_analyzer.analyzer',
        'doc_analyzer.reporter',
        'doc_analyzer.cache',
        'sklearn.cluster._hdbscan',
        'sklearn.cluster._hdbscan.hdbscan',
        'sklearn.neighbors._partition_nodes',
    ],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[
        # Exclude unused database drivers that cause SSL conflicts
        'mysql',
        'MySQLdb',
        'pymysql',
        'psycopg2',
        'sqlalchemy',
        'pandas.io.sql',
        # Exclude test frameworks
        'pytest',
        'py',
        '_pytest',
    ],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name='doc-analyzer',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=True,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)
