#!/bin/bash
set -euo pipefail

# AI Control Plane - Leadership Materials PDF Generator
#
# Purpose: Generates PDF documents from Markdown files using Marp CLI
#
# Usage: ./generate-pdfs.sh [--help]
#
# Options:
#   --help    Show this help message and exit
#
# Examples:
#   ./generate-pdfs.sh
#       Generate PDFs from all markdown files in the directory
#
# Exit codes:
#   0   - Success
#   1   - Marp CLI not found or generation failed
#
# Requires: Marp CLI (npm install -g @marp-team/marp-cli)
# Optional: Playwright/Chromium for HTML to PDF

# Show help if requested before changing directory
if [[ "${1:-}" == "--help" ]]; then
    cat <<'EOF'
AI Control Plane - Leadership Materials PDF Generator

Purpose: Generates PDF documents from Markdown files using Marp CLI

Usage: ./generate-pdfs.sh [--help]

Options:
  --help    Show this help message and exit

Examples:
  ./generate-pdfs.sh
      Generate PDFs from all markdown files in the directory

Exit codes:
  0   - Success
  1   - Marp CLI not found or generation failed

Requires: Marp CLI (npm install -g @marp-team/marp-cli)
Optional: Playwright/Chromium for HTML to PDF
EOF
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== AI Control Plane Leadership Materials ==="
echo ""

# Check for Marp CLI
if ! command -v marp &>/dev/null; then
    echo "❌ Marp CLI not found. Install with:"
    echo "   npm install -g @marp-team/marp-cli"
    echo ""
    echo "Alternatively, you can open the HTML file directly in a browser"
    echo "and use Print to PDF."
    exit 1
fi

echo "✓ Marp CLI found"
echo ""

# Generate PDF from Marp deck
echo "Generating PDF from leadership deck..."
marp ai-control-plane-leadership-deck.md \
    --pdf \
    --output ai-control-plane-leadership-deck.pdf \
    --allow-local-files \
    --theme default

echo "✓ Generated: ai-control-plane-leadership-deck.pdf"
echo ""

# For HTML one-pager, provide instructions
echo "=== HTML One-Pager ==="
echo "The executive-one-pager.html can be:"
echo "  1. Opened directly in any web browser"
echo "  2. Printed to PDF (recommended for sharing)"
echo "     - Chrome: Open → Ctrl+P → Save as PDF → Enable Background Graphics"
echo "     - Safari: Open → File → Export as PDF"
echo ""

echo "=== All Materials Generated ==="
ls -lh *.pdf *.html *.md 2>/dev/null | grep -E '\.(pdf|html|md)$' || true
