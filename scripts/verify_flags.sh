#!/bin/bash
set -e

# Setup
TEST_DIR="/tmp/loot_test_flags"
SRC="$TEST_DIR/src"
DST="$TEST_DIR/dst"
mkdir -p "$SRC" "$DST"
echo "test file" > "$SRC/test.txt"

echo "=== Verifying LOOT CLI Flags ==="

# 1. Positional Args
echo "[1] Positional Args..."
go run cmd/loot/main.go --dry-run "$SRC" "$DST" > /dev/null
echo "✅ Passed"

# 2. Named Args (--source, --dest)
echo "[2] Named Args..."
go run cmd/loot/main.go --dry-run --source "$SRC" --dest "$DST" > /dev/null
echo "✅ Passed"

# 3. Algorithms
echo "[3] Algorithm Flags..."
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --md5 > /dev/null
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --sha256 > /dev/null
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --xxhash64 > /dev/null
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --algorithm md5 > /dev/null
echo "✅ Passed"

# 4. Metadata Modes
echo "[4] Metadata Modes..."
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --metadata-mode off > /dev/null
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --metadata-mode hybrid > /dev/null
echo "✅ Passed"

# 5. Behavior Flags
echo "[5] Behavior Flags..."
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --json > /dev/null
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --quiet > /dev/null
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --no-verify > /dev/null
go run cmd/loot/main.go --dry-run "$SRC" "$DST" --resume > /dev/null
echo "✅ Passed"

# 6. Mixed Flags (User Request Scenario)
echo "[6] Mixed Order..."
go run cmd/loot/main.go --md5 --json --source "$SRC" --dest "$DST" --dry-run > /dev/null
echo "✅ Passed"

# Cleanup
rm -rf "$TEST_DIR"
echo "🎉 All Flag Verification Tests Passed!"
