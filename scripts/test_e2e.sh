#!/bin/bash
set -e

APP="./bin/loot"
SRC="/tmp/loot_test_src"
DST="/tmp/loot_test_dst"

echo "=== LOOT E2E Test ==="

# 1. Build
echo "[1/5] Building LOOT..."
make build

# 2. Setup
echo "[2/5] Setting up test environment..."
rm -rf "$SRC" "$DST"
mkdir -p "$SRC"

# Create test files
echo "File 1 Content" > "$SRC/file1.txt"
echo "File 2 Content" > "$SRC/file2.txt"
dd if=/dev/urandom of="$SRC/random.bin" bs=1M count=1 2>/dev/null

# 3. Execution (CLI Mode)
echo "[3/5] Running LOOT copy..."
# Check if binary exists
if [ ! -f "$APP" ]; then
    echo "Error: Binary not found at $APP"
    exit 1
fi

$APP --quiet --no-verify "$SRC" "$DST"

# 4. Verification
echo "[4/5] Verifying output..."

if [ ! -d "$DST" ]; then
    echo "Error: Destination directory not created"
    exit 1
fi

# Check file1
if cmp -s "$SRC/file1.txt" "$DST/file1.txt"; then
    echo "  [OK] file1.txt matches"
else
    echo "  [FAIL] file1.txt mismatch"
    exit 1
fi

# Check random file
# Note: LOOT copies contents of SRC into DEST/SRC_BASENAME? 
# Wait, let's verify behavior. 
# If I run `loot /tmp/src /tmp/dst`, and `src` is a dir.
# Does it put files in `/tmp/dst/file1.txt` or `/tmp/dst/src/file1.txt`?
# Based on unit test failure earlier where I suspected merging:
# If `filepath.Rel` is used, and dest is root.
# Let's inspect offloader logic implementation via memory or assume standard rsync behavior?
# `cp -r src dst` -> `dst/src/` if dst exists?
# The `offloader.go` code: `relPath, _ := filepath.Rel(o.Source, path); destPath := filepath.Join(dstRoot, relPath)`
# If `o.Source` is `/tmp/src` and `path` is `/tmp/src/file1.txt`, rel is `file1.txt`.
# `destPath` is `/tmp/dst/file1.txt`.
# So it merges content into DST.
# My script checks `$DST/file1.txt`. This is correct.

if cmp -s "$SRC/random.bin" "$DST/random.bin"; then
    echo "  [OK] random.bin matches"
else
    echo "  [FAIL] random.bin mismatch"
    exit 1
fi

# 5. Cleanup
echo "[5/5] Cleanup..."
rm -rf "$SRC" "$DST"

echo "=== SUCCESS: E2E Test Passed ==="
exit 0
