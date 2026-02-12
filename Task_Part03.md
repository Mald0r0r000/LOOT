🗺️ LOOT - Roadmap Complète (Suite - Documentation Finale)

markdown<!-- docs/TROUBLESHOOTING.md -->
# Troubleshooting Guide

## Common Issues

### 1. "Source does not exist" Error

**Problem:**
```bash
$ loot /Volumes/CARD /Volumes/BACKUP
Error: Source file '/Volumes/CARD' does not exist.
```

**Solutions:**

a) Check if volume is mounted:
```bash
ls /Volumes
```

b) Volume name might have spaces:
```bash
# Wrong
loot /Volumes/SD CARD /Volumes/BACKUP

# Correct
loot "/Volumes/SD CARD" /Volumes/BACKUP
```

c) Use tab completion or drag & drop in terminal

---

### 2. "Permission denied" Error

**Problem:**
```bash
Error: failed to create dir: permission denied
```

**Solutions:**

a) Check destination permissions:
```bash
ls -la /Volumes/BACKUP
```

b) Ensure you have write access:
```bash
# Fix permissions
sudo chown -R $(whoami) /Volumes/BACKUP
```

c) Try with sudo (not recommended for normal use):
```bash
sudo loot /Volumes/CARD /Volumes/BACKUP
```

---

### 3. Verification Failed / Checksum Mismatch

**Problem:**
```bash
❌ Checksum mismatch!
```

**Possible Causes:**

1. **Disk errors during copy**
   - Check source disk health: `diskutil verifyDisk /Volumes/CARD`
   - Check destination disk: `diskutil verifyDisk /Volumes/BACKUP`

2. **File was modified during copy**
   - Ensure no other process is writing to source
   - Close camera, editing software, etc.

3. **Hardware issues**
   - Try different cable
   - Try different USB port
   - Check for bad sectors: `smartctl -a /dev/diskX`

**Recovery:**
```bash
# 1. Delete partial copy
rm -rf /Volumes/BACKUP/failed-copy

# 2. Re-run with verbose logging
loot --verbose --algorithm md5 /Volumes/CARD /Volumes/BACKUP 2>&1 | tee offload.log

# 3. If still fails, try different algorithm
loot --algorithm sha256 /Volumes/CARD /Volumes/BACKUP
```

---

### 4. Slow Transfer Speed

**Expected Speeds:**

| Connection | Expected Speed |
|------------|---------------|
| USB 2.0 | 20-40 MB/s |
| USB 3.0 | 100-200 MB/s |
| USB 3.1 | 300-500 MB/s |
| Thunderbolt 3 | 1-2 GB/s |

**Problem:** Transfer slower than expected

**Solutions:**

a) Check USB version:
```bash
system_profiler SPUSBDataType
```

b) Increase buffer size:
```bash
loot --buffer-size 8388608 /Volumes/CARD /Volumes/BACKUP  # 8MB
```

c) Use faster hash algorithm:
```bash
# Instead of SHA256
loot --algorithm xxhash64 /Volumes/CARD /Volumes/BACKUP
```

d) Check CPU usage:
```bash
# If CPU is bottleneck, reduce buffer or use lighter hash
top -o cpu
```

e) Disable Spotlight indexing on destination:
```bash
sudo mdutil -i off /Volumes/BACKUP
```

---

### 5. "Insufficient Space" Warning

**Problem:**
```bash
⚠️  INSUFFICIENT SPACE (need 458 GB)
```

**Solutions:**

a) Check actual space:
```bash
df -h /Volumes/BACKUP
```

b) Clean destination:
```bash
# Find large files
du -sh /Volumes/BACKUP/* | sort -h

# Remove old backups
rm -rf /Volumes/BACKUP/old-folder
```

c) Use compression (if supported by destination):
```bash
# Not built into LOOT, but you can:
rsync -avz /Volumes/CARD/ /Volumes/BACKUP/
```

---

### 6. Job Resume Issues

**Problem:** Can't resume interrupted job

**Solutions:**

a) List jobs to verify ID:
```bash
loot jobs list
```

b) Check job file exists:
```bash
ls ~/.loot/jobs/
cat ~/.loot/jobs/loot-20260212-143022.json
```

c) If corrupted, delete and restart:
```bash
rm ~/.loot/jobs/loot-20260212-143022.json
loot /Volumes/CARD /Volumes/BACKUP
```

d) Manual resume (copy remaining files):
```bash
# Find what was already copied
diff -r /Volumes/CARD /Volumes/BACKUP

# Copy missing files manually or restart LOOT
```

---

### 7. JSON Output Issues

**Problem:** Invalid JSON output

**Solutions:**

a) Ensure you're using `--json` flag:
```bash
loot --json /Volumes/CARD /Volumes/BACKUP
```

b) Capture only JSON (suppress TUI):
```bash
loot --json --quiet /Volumes/CARD /Volumes/BACKUP > output.json
```

c) Validate JSON:
```bash
loot --json /Volumes/CARD /Volumes/BACKUP | jq .
```

d) Check for stderr contamination:
```bash
loot --json /Volumes/CARD /Volumes/BACKUP 2>/dev/null
```

---

### 8. Memory Issues / Crash

**Problem:** LOOT crashes or uses too much memory

**Solutions:**

a) Reduce buffer size:
```bash
loot --buffer-size 1048576 /Volumes/CARD /Volumes/BACKUP  # 1MB
```

b) Check available memory:
```bash
vm_stat
```

c) Close other applications

d) For huge directories (100K+ files), process in batches:
```bash
# Split into subdirectories
for dir in /Volumes/CARD/*; do
  loot "$dir" "/Volumes/BACKUP/$(basename $dir)"
done
```

---

### 9. MHL/PDF Generation Failed

**Problem:** Reports not generated

**Solutions:**

a) Check write permissions:
```bash
ls -la /Volumes/BACKUP
```

b) Ensure destination is writable:
```bash
touch /Volumes/BACKUP/test.txt && rm /Volumes/BACKUP/test.txt
```

c) Check for special characters in filenames:
```bash
# Sanitize if needed
# (LOOT should handle this, but check)
```

d) Manually verify files exist after copy:
```bash
ls -la /Volumes/BACKUP/*.pdf
ls -la /Volumes/BACKUP/*.mhl
```

---

### 10. Volume Detection Issues

**Problem:** Volumes not showing in TUI

**Solutions:**

a) Refresh volumes list:
```bash
# In TUI, press 'r' to refresh
```

b) Check `/Volumes`:
```bash
ls -la /Volumes
```

c) Remount volume:
```bash
diskutil unmount /Volumes/CARD
diskutil mount /dev/diskXsY
```

d) Use full path in CLI mode:
```bash
# If TUI doesn't detect, use CLI
loot /Volumes/CARD /Volumes/BACKUP
```

---

## Debug Mode

Enable verbose logging for troubleshooting:
```bash
# Save full logs
loot --verbose /Volumes/CARD /Volumes/BACKUP 2>&1 | tee loot-debug.log

# Check log
cat loot-debug.log
```

## Reporting Bugs

If you encounter a bug:

1. **Gather information:**
```bash
# System info
sw_vers
system_profiler SPHardwareDataType

# LOOT version
loot --version

# Reproduce with verbose
loot --verbose  2>&1 > bug-report.log
```

2. **Create issue on GitHub:**
   - Go to: https://github.com/Mald0r0r000/LOOT/issues
   - Include: OS version, LOOT version, command used, log file

3. **Minimal reproducible example:**
```bash
# Create small test case
mkdir /tmp/test-source
echo "test" > /tmp/test-source/file.txt
loot --verbose /tmp/test-source /tmp/test-dest
```

---

## FAQ

**Q: Can I cancel an ongoing copy?**  
A: Yes, press `Ctrl+C`. The job will be saved and can be resumed with `loot resume <job-id>`.

**Q: Why is MD5 so much slower than xxHash64?**  
A: MD5 is a cryptographic hash (more complex math). xxHash64 is optimized for speed, not cryptographic security.

**Q: Can I use LOOT over network?**  
A: Yes, but performance depends on network speed. Local drives recommended for best performance.

**Q: Does LOOT work with RAID arrays?**  
A: Yes, LOOT sees RAID as a single volume.

**Q: Can I offload directly to cloud?**  
A: Not directly. Offload to local storage first, then sync to cloud (see AUTOMATION.md).

**Q: Is LOOT compatible with ShotPut Pro MHL files?**  
A: Yes, LOOT generates standard MHL XML format compatible with industry tools.

**Q: Can I use LOOT in production?**  
A: Yes! LOOT is designed for professional DIT workflows. Always test with your specific setup first.

**Q: What's the maximum file size?**  
A: No hard limit. Tested with 500GB+ files successfully.

**Q: Does LOOT preserve metadata?**  
A: Yes, file timestamps, permissions, and extended attributes are preserved.

markdown<!-- docs/API.md -->
# JSON API Reference

## Overview

LOOT provides JSON output for integration with scripts, automation tools, and AI agents.

## Basic Usage
```bash
loot --json  
```

## Output Format

### Success Response
```json
{
  "job_id": "loot-20260212-143022",
  "status": "success",
  "source": "/Volumes/CFAST_A",
  "destinations": [
    "/Volumes/BACKUP/2026-02-12"
  ],
  "files_count": 245,
  "total_bytes": 52428800000,
  "duration_seconds": 127.3,
  "speed_mbps": 3284.5,
  "hash": {
    "xxhash64": "a3f2c91b7e8d4f21",
    "md5": "5d41402abc4b2a76b9719d911017c592",
    "sha256": ""
  },
  "algorithm": "md5",
  "dual_hash": true,
  "verified": true,
  "errors": [],
  "timestamp": "2026-02-12T14:30:22Z",
  "files": [
    {
      "path": "A001C001.mxf",
      "size": 214748364,
      "hash": {
        "xxhash64": "f1e2d3c4b5a69788",
        "md5": "098f6bcd4621d373cade4e832627b4f6",
        "sha256": ""
      },
      "verified": true
    }
  ]
}
```

### Failure Response
```json
{
  "job_id": "loot-20260212-143500",
  "status": "failed",
  "source": "/Volumes/CARD",
  "destinations": ["/Volumes/BACKUP"],
  "files_count": 0,
  "total_bytes": 0,
  "duration_seconds": 0.5,
  "speed_mbps": 0,
  "hash": {},
  "algorithm": "xxhash64",
  "dual_hash": false,
  "verified": false,
  "errors": [
    "source '/Volumes/CARD' does not exist"
  ],
  "timestamp": "2026-02-12T14:35:00Z"
}
```

## Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `job_id` | string | Unique job identifier (format: `loot-YYYYMMDD-HHMMSS`) |
| `status` | string | Job status: `success`, `failed`, `partial` |
| `source` | string | Source path |
| `destinations` | array[string] | List of destination paths |
| `files_count` | integer | Total number of files processed |
| `total_bytes` | integer | Total bytes transferred |
| `duration_seconds` | float | Time taken in seconds |
| `speed_mbps` | float | Average speed in megabits per second |
| `hash` | object | Hash results (see Hash Object) |
| `algorithm` | string | Primary hash algorithm used |
| `dual_hash` | boolean | Whether dual-hash was enabled |
| `verified` | boolean | Whether verification passed |
| `errors` | array[string] | List of errors (empty if success) |
| `timestamp` | string | ISO 8601 timestamp |
| `files` | array[object] | Individual file details (optional) |

### Hash Object
```json
{
  "xxhash64": "a3f2c91b7e8d4f21",
  "md5": "5d41402abc4b2a76b9719d911017c592",
  "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
```

Fields are populated based on `algorithm` and `dual_hash` settings.

### File Object
```json
{
  "path": "relative/path/to/file.mxf",
  "size": 214748364,
  "hash": {
    "xxhash64": "...",
    "md5": "...",
    "sha256": "..."
  },
  "verified": true
}
```

## Status Codes

| Status | Meaning |
|--------|---------|
| `success` | All files copied and verified successfully |
| `failed` | Operation failed completely |
| `partial` | Some files succeeded, some failed (future) |

## Parsing Examples

### Bash
```bash
#!/bin/bash

RESULT=$(loot --json /Volumes/CARD /Volumes/BACKUP)

# Extract status
STATUS=$(echo "$RESULT" | jq -r '.status')

if [ "$STATUS" == "success" ]; then
  FILES=$(echo "$RESULT" | jq -r '.files_count')
  SPEED=$(echo "$RESULT" | jq -r '.speed_mbps')
  
  echo "✅ Success: $FILES files at ${SPEED} Mbps"
else
  ERRORS=$(echo "$RESULT" | jq -r '.errors[]')
  echo "❌ Failed: $ERRORS"
fi
```

### Python
```python
import subprocess
import json

result = subprocess.run(
    ['loot', '--json', '/Volumes/CARD', '/Volumes/BACKUP'],
    capture_output=True,
    text=True
)

data = json.loads(result.stdout)

if data['status'] == 'success':
    print(f"✅ {data['files_count']} files")
    print(f"Speed: {data['speed_mbps'] / 8:.1f} MB/s")
    print(f"Hash: {data['hash']['md5']}")
else:
    print(f"❌ Errors: {', '.join(data['errors'])}")
```

### Node.js
```javascript
const { execSync } = require('child_process');

const output = execSync(
  'loot --json /Volumes/CARD /Volumes/BACKUP',
  { encoding: 'utf-8' }
);

const result = JSON.parse(output);

if (result.status === 'success') {
  console.log(`✅ ${result.files_count} files`);
  console.log(`Speed: ${(result.speed_mbps / 8).toFixed(1)} MB/s`);
} else {
  console.error(`❌ ${result.errors.join(', ')}`);
}
```

### Go
```go
package main

import (
    "encoding/json"
    "fmt"
    "os/exec"
)

type LootResult struct {
    JobID       string   `json:"job_id"`
    Status      string   `json:"status"`
    FilesCount  int      `json:"files_count"`
    TotalBytes  int64    `json:"total_bytes"`
    SpeedMbps   float64  `json:"speed_mbps"`
    Verified    bool     `json:"verified"`
    Errors      []string `json:"errors"`
}

func main() {
    cmd := exec.Command("loot", "--json", "/Volumes/CARD", "/Volumes/BACKUP")
    output, err := cmd.Output()
    if err != nil {
        panic(err)
    }
    
    var result LootResult
    json.Unmarshal(output, &result)
    
    if result.Status == "success" {
        fmt.Printf("✅ %d files verified\n", result.FilesCount)
        fmt.Printf("Speed: %.1f MB/s\n", result.SpeedMbps/8)
    } else {
        fmt.Printf("❌ Errors: %v\n", result.Errors)
    }
}
```

## Integration Patterns

### Slack Notification
```bash
#!/bin/bash

RESULT=$(loot --json --algorithm md5 /Volumes/CARD /Volumes/BACKUP)
STATUS=$(echo "$RESULT" | jq -r '.status')
FILES=$(echo "$RESULT" | jq -r '.files_count')
SIZE=$(echo "$RESULT" | jq -r '.total_bytes')

if [ "$STATUS" == "success" ]; then
  MESSAGE="✅ Offload complete: $FILES files ($(numfmt --to=iec $SIZE))"
  COLOR="good"
else
  MESSAGE="❌ Offload failed"
  COLOR="danger"
fi

curl -X POST -H 'Content-type: application/json' \
  --data "{
    \"attachments\": [{
      \"color\": \"$COLOR\",
      \"text\": \"$MESSAGE\"
    }]
  }" \
  $SLACK_WEBHOOK_URL
```

### Database Logging
```python
import subprocess
import json
import sqlite3
from datetime import datetime

# Run offload
result = subprocess.run(
    ['loot', '--json', '/Volumes/CARD', '/Volumes/BACKUP'],
    capture_output=True,
    text=True
)

data = json.loads(result.stdout)

# Store in database
conn = sqlite3.connect('offloads.db')
cursor = conn.cursor()

cursor.execute('''
    INSERT INTO offloads (
        job_id, status, source, files_count, 
        total_bytes, duration, verified, timestamp
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
''', (
    data['job_id'],
    data['status'],
    data['source'],
    data['files_count'],
    data['total_bytes'],
    data['duration_seconds'],
    data['verified'],
    data['timestamp']
))

conn.commit()
conn.close()
```

### Email Report
```python
import subprocess
import json
import smtplib
from email.message import EmailMessage

result = subprocess.run(
    ['loot', '--json', '/Volumes/CARD', '/Volumes/BACKUP'],
    capture_output=True,
    text=True
)

data = json.loads(result.stdout)

# Format email
msg = EmailMessage()
msg['Subject'] = f"Offload {data['status'].upper()} - {data['job_id']}"
msg['From'] = 'dit@production.com'
msg['To'] = 'supervisor@production.com'

body = f"""
Offload Report
==============

Job ID: {data['job_id']}
Status: {data['status']}
Source: {data['source']}
Files: {data['files_count']}
Size: {data['total_bytes'] / (1024**3):.2f} GB
Duration: {data['duration_seconds']:.1f}s
Speed: {data['speed_mbps'] / 8:.1f} MB/s
Verified: {'✅ Yes' if data['verified'] else '❌ No'}
"""

if data['errors']:
    body += f"\nErrors:\n" + "\n".join(f"- {e}" for e in data['errors'])

msg.set_content(body)

# Send
with smtplib.SMTP('smtp.gmail.com', 587) as smtp:
    smtp.starttls()
    smtp.login('user@gmail.com', 'password')
    smtp.send_message(msg)
```

## Dry Run API
```bash
loot --dry-run --json /Volumes/CARD /Volumes/BACKUP
```

Returns:
```json
{
  "dry_run": true,
  "source": "/Volumes/CARD",
  "destinations": ["/Volumes/BACKUP"],
  "files": [
    {
      "path": "A001C001.mxf",
      "size": 214748364
    }
  ],
  "total_bytes": 52428800000,
  "destination_info": [
    {
      "path": "/Volumes/BACKUP",
      "available_bytes": 107374182400,
      "sufficient_space": true
    }
  ]
}
```

## Resume API

Job state JSON (in `~/.loot/jobs/<job-id>.json`):
```json
{
  "id": "loot-20260212-143022",
  "source": "/Volumes/CARD",
  "destinations": ["/Volumes/BACKUP"],
  "status": "paused",
  "algorithm": "md5",
  "dual_hash": false,
  "files": [
    {
      "path": "file1.mxf",
      "size": 214748364,
      "bytes_copied": 214748364,
      "hash": {},
      "completed": true
    },
    {
      "path": "file2.mxf",
      "size": 214748364,
      "bytes_copied": 107374182,
      "hash": {},
      "completed": false
    }
  ],
  "total_bytes": 429496728,
  "copied_bytes": 322122546,
  "created_at": "2026-02-12T14:30:22Z",
  "updated_at": "2026-02-12T14:32:15Z"
}
```

## Error Codes

LOOT uses exit codes for scripting:

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Source not found |
| 4 | Destination error |
| 5 | Verification failed |
| 6 | Insufficient space |

Usage:
```bash
loot /Volumes/CARD /Volumes/BACKUP
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
  echo "Success"
elif [ $EXIT_CODE -eq 5 ]; then
  echo "Verification failed - DO NOT FORMAT CARD"
fi
```

TASK 4.2: Homebrew Formula Creation
Priority: 🔴 CRITICAL
Estimated time: 2h
Dependencies: TASK 3.3 (GitHub releases)
Context:

Distribution via Homebrew est le standard macOS
Simplifie installation pour les utilisateurs
Updates automatiques

Files to create:

homebrew-loot/Formula/loot.rb
homebrew-loot/README.md

Implementation:
ruby# homebrew-loot/Formula/loot.rb
class Loot < Formula
  desc "Professional media offload tool for DITs and filmmakers"
  homepage "https://github.com/Mald0r0r000/LOOT"
  license "MIT"
  version "1.0.0"

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/Mald0r0r000/LOOT/releases/download/v1.0.0/loot-1.0.0-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_ARM64"
    else
      url "https://github.com/Mald0r0r000/LOOT/releases/download/v1.0.0/loot-1.0.0-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_AMD64"
    end
  end

  def install
    if Hardware::CPU.arm?
      bin.install "loot-darwin-arm64" => "loot"
    else
      bin.install "loot-darwin-amd64" => "loot"
    end

    # Install shell completions (when implemented)
    # bash_completion.install "completions/loot.bash"
    # zsh_completion.install "completions/_loot"
    # fish_completion.install "completions/loot.fish"

    # Install man page (when created)
    # man1.install "docs/loot.1"
  end

  def post_install
    # Create config directory
    (var/"loot").mkpath
    (var/"loot/jobs").mkpath

    # Create default config if doesn't exist
    config_file = etc/"loot/config.yaml"
    unless config_file.exist?
      config_file.write <<~EOS
        # LOOT Configuration
        # Default settings for media offload operations

        defaults:
          # Hash algorithm: xxhash64 (fastest), md5 (compatible), sha256 (forensic)
          hash_algorithm: xxhash64
          
          # Buffer size for I/O operations (bytes)
          buffer_size: 4194304  # 4MB
          
          # Dual hash: calculate both xxhash64 and md5
          dual_hash: false

        # Workflow profiles
        workflows:
          quick:
            hash_algorithm: xxhash64
            dual_hash: false
            
          delivery:
            hash_algorithm: md5
            dual_hash: false
            
          archive:
            hash_algorithm: sha256
            dual_hash: false
            
          production:
            hash_algorithm: md5
            dual_hash: true
      EOS
    end

    ohai "LOOT installed successfully!"
    ohai "Config directory: #{etc}/loot"
    ohai "Job storage: #{var}/loot/jobs"
  end

  test do
    # Test version output
    assert_match "loot version #{version}", shell_output("#{bin}/loot --version")
    
    # Test help output
    assert_match "Usage:", shell_output("#{bin}/loot --help 2>&1", 1)
    
    # Test dry-run with temp files
    (testpath/"source").mkpath
    (testpath/"source/test.txt").write "test data"
    
    output = shell_output("#{bin}/loot --dry-run --json #{testpath}/source #{testpath}/dest 2>&1")
    
    # Verify JSON output contains expected fields
    assert_match "\"dry_run\": true", output
    assert_match "\"source\":", output
  end

  def caveats
    <<~EOS
      LOOT has been installed!

      Quick start:
        # Interactive mode
        loot

        # CLI mode
        loot /Volumes/CARD /Volumes/BACKUP

      Documentation:
        https://github.com/Mald0r0r000/LOOT/tree/main/docs

      Configuration:
        #{etc}/loot/config.yaml

      Examples:
        # Fast offload with xxHash64
        loot /Volumes/SD_CARD /Volumes/BACKUP

        # Industry-standard MD5 verification
        loot --algorithm md5 /Volumes/CFAST /Volumes/RAID

        # Forensic-grade SHA256 + JSON output
        loot --algorithm sha256 --json /Volumes/CARD /Volumes/ARCHIVE

        # Resume interrupted job
        loot jobs list
        loot resume <job-id>

      For more examples and workflows:
        https://github.com/Mald0r0r000/LOOT/blob/main/docs/WORKFLOWS.md
    EOS
  end
end
markdown<!-- homebrew-loot/README.md -->
# Homebrew Tap for LOOT

Official Homebrew tap for LOOT - Professional media offload tool.

## Installation
```bash
brew tap Mald0r0r000/loot
brew install loot
```

## Usage

See the [main repository](https://github.com/Mald0r0r000/LOOT) for full documentation.

Quick start:
```bash
# Interactive mode
loot

# CLI mode
loot /Volumes/CARD /Volumes/BACKUP
```

## Updating
```bash
brew update
brew upgrade loot
```

## Uninstalling
```bash
brew uninstall loot
brew untap Mald0r0r000/loot
```

## Development

### Testing the Formula Locally
```bash
# Install from local formula
brew install --build-from-source Formula/loot.rb

# Test
brew test loot

# Audit formula
brew audit --strict loot
```

### Updating the Formula

1. Update version in `Formula/loot.rb`
2. Download new release tarballs
3. Calculate SHA256:
```bash
   shasum -a 256 loot-1.0.0-darwin-arm64.tar.gz
   shasum -a 256 loot-1.0.0-darwin-amd64.tar.gz
```
4. Update SHA256 values in formula
5. Commit and push

## Support

For issues with the Homebrew formula, please open an issue in this repository.

For issues with LOOT itself, see the [main repository](https://github.com/Mald0r0r000/LOOT/issues).
Setup Instructions:
bash# Create tap repository
mkdir homebrew-loot
cd homebrew-loot
git init

# Create Formula directory
mkdir -p Formula

# Create formula file (as shown above)
# Formula/loot.rb

# Create README
# README.md

# Commit
git add .
git commit -m "Initial formula for LOOT v1.0.0"

# Push to GitHub
git remote add origin https://github.com/Mald0r0r000/homebrew-loot.git
git push -u origin main
Testing the Formula:
bash# Test installation locally
brew install --build-from-source ./Formula/loot.rb

# Run tests
brew test loot

# Audit
brew audit --strict --online loot

# Uninstall
brew uninstall loot
Release Checklist:

Create GitHub release with binaries
Calculate SHA256 for both architectures:

bash   shasum -a 256 loot-1.0.0-darwin-arm64.tar.gz
   shasum -a 256 loot-1.0.0-darwin-amd64.tar.gz

Update formula with correct SHA256 values
Test formula locally
Push to tap repository
Announce on README.md, Twitter, etc.

Success criteria:

 Formula installs successfully
 brew test loot passe
 brew audit sans warnings
 Post-install message s'affiche
 Config créée dans ~/.loot/


TASK 4.3: README.md Final avec badges et comparaison
Priority: 🟠 HIGH
Estimated time: 2h
Dependencies: Toutes les features complétées
Context:

README est la première impression
Doit vendre le produit
Comparaison claire avec alternatives

Files to modify:

README.md

Implementation:
markdown# LOOT 💰

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8.svg)
![Release](https://img.shields.io/github/v/release/Mald0r0r000/LOOT)
![Downloads](https://img.shields.io/github/downloads/Mald0r0r000/LOOT/total)
![Tests](https://github.com/Mald0r0r000/LOOT/workflows/Test/badge.svg)
![Coverage](https://codecov.io/gh/Mald0r0r000/LOOT/branch/main/graph/badge.svg)

**LOOT** is a high-performance, professional media offload tool built for the terminal. It is designed to be a **free, open-source alternative** to industry standards like ShotPut Pro and Silverstack, providing reliable verification and reporting for DITs (Digital Imaging Technicians) and media professionals.

Perfect for the AI agent era — automate your entire offload workflow with Claude, ChatGPT, or any AI assistant.
```text
██╗      ██████╗  ██████╗ ████████╗
██║     ██╔══/██╗██╔══/██╗╚══██╔══╝
██║     ██║ / ██║██║ / ██║   ██║   
██║     ██║/  ██║██║/  ██║   ██║   
███████╗╚██████╔╝╚██████╔╝   ██║   
╚══════╝ ╚═════╝  ╚═════╝    ╚═╝   
```

![Demo GIF](docs/demo.gif)

## ✨ Features

- **🚀 Blazing Fast**: xxHash64 verification at 3.5 GB/s on Apple Silicon
- **🔒 Industry Standard**: MD5 and SHA256 support for delivery and archival
- **📊 Dual-Hash**: Calculate both xxHash64 and MD5 simultaneously
- **🔄 Resume**: Interrupted transfers resume exactly where they left off
- **📂 Multi-Destination**: Copy to multiple drives simultaneously with independent verification
- **📑 MHL Support**: Generates standard Media Hash List (MHL) files for workflow compatibility
- **📄 PDF Reports**: Detailed verification reports with transfer statistics
- **🤖 AI-Ready**: JSON output perfect for automation with AI agents
- **💻 Modern TUI**: Beautiful terminal interface with real-time progress
- **🆓 100% Free**: No subscriptions, no limitations, fully open-source

## 🎯 Why LOOT?

| Feature | ShotPut Pro | LOOT |
|---------|-------------|------|
| **Price** | $99/year | **Free** |
| **Open Source** | ❌ | **✅** |
| **CLI Native** | ❌ | **✅** |
| **AI Agent Ready** | ❌ | **✅ JSON API** |
| **Resume Jobs** | ✅ | **✅** |
| **Multi-Destination** | ✅ | **✅** |
| **Hash Speed** | MD5: ~350 MB/s | **xxHash64: 3.5 GB/s** |
| **Dual-Hash** | ❌ | **✅ xxHash + MD5** |
| **MHL Export** | ✅ | **✅** |
| **PDF Reports** | ✅ | **✅** |
| **Homebrew Install** | ❌ | **✅** |
| **Scriptable** | Limited | **Full JSON API** |

## 📦 Installation

### Homebrew (Recommended)
```bash
brew tap Mald0r0r000/loot
brew install loot
```

### Manual Download

Download the latest release for your platform:
- [macOS Apple Silicon (M1/M2/M3)](https://github.com/Mald0r0r000/LOOT/releases/latest/download/loot-darwin-arm64.tar.gz)
- [macOS Intel](https://github.com/Mald0r0r000/LOOT/releases/latest/download/loot-darwin-amd64.tar.gz)
```bash
tar xzf loot-*.tar.gz
sudo mv loot-* /usr/local/bin/loot
```

### Build from Source
```bash
git clone https://github.com/Mald0r0r000/LOOT.git
cd LOOT
make install
```

## 🚀 Quick Start

### Interactive Mode (TUI)
```bash
loot
```

Navigate with arrows, select source/destination with `Space`.

### CLI Mode
```bash
# Basic offload with xxHash64 (fastest)
loot /Volumes/SD_CARD /Volumes/BACKUP

# Industry-standard MD5 verification
loot --algorithm md5 /Volumes/CFAST /Volumes/RAID

# Forensic-grade SHA256
loot --algorithm sha256 /Volumes/CARD /Volumes/ARCHIVE

# Dual-hash (best of both worlds)
loot --dual-hash /Volumes/CARD /Volumes/BACKUP

# Multi-destination
loot /Volumes/CARD /Volumes/BACKUP_A /Volumes/BACKUP_B

# JSON output for automation
loot --json --algorithm md5 /Volumes/CARD /Volumes/BACKUP
```

## 🤖 AI Agent Integration

LOOT is designed for the AI agent era. Use with Claude, ChatGPT, or any AI assistant:

**User:** "Offload my SD card with MD5 verification and email me the report"

**Claude generates:**
```bash
#!/bin/bash
RESULT=$(loot --json --algorithm md5 /Volumes/SD_CARD /Volumes/BACKUP)
STATUS=$(echo "$RESULT" | jq -r '.status')

if [ "$STATUS" == "success" ]; then
  REPORT=$(echo "$RESULT" | jq -r '.destinations[0]').pdf
  echo "Offload complete!" | mail -s "Offload Success ✅" -A "$REPORT" you@example.com
fi
```

See [AUTOMATION.md](docs/AUTOMATION.md) for more examples.

## 📖 Documentation

- [Installation Guide](docs/INSTALLATION.md)
- [Usage Guide](docs/USAGE.md)
- [DIT Workflows](docs/WORKFLOWS.md)
- [AI Automation](docs/AUTOMATION.md)
- [JSON API Reference](docs/API.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

## 🎬 Real-World Usage

### Daily DIT Workflow
```bash
# Morning card offload with MD5
loot --algorithm md5 /Volumes/CFAST_A /Volumes/RAID/Dailies/$(date +%Y-%m-%d)

# Generates:
# - /Volumes/RAID/Dailies/2026-02-12/
# - /Volumes/RAID/Dailies/2026-02-12.pdf (verification report)
# - /Volumes/RAID/Dailies/2026-02-12.mhl (hash manifest)
```

### Production Safety
```bash
# Dual-destination with dual-hash
loot --dual-hash \
  /Volumes/CARD \
  /Volumes/PRIMARY \
  /Volumes/SECONDARY

# Verifies both copies independently
# Generates separate reports for each destination
```

### Resume Interrupted Transfer
```bash
# Transfer interrupted? No problem!
loot jobs list
loot resume loot-20260212-143022

# Continues exactly where it stopped
# No duplicate copying
```

## 🏗️ Architecture
```
LOOT/
├── cmd/loot/           # Main entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── hash/           # Multi-algorithm hashing
│   ├── offload/        # Core copy & verify logic
│   ├── job/            # Job persistence & resume
│   ├── output/         # JSON & human-readable output
│   ├── report/         # PDF generation
│   ├── mhl/            # MHL file generation
│   └── ui/             # Terminal UI (Bubble Tea)
├── docs/               # Documentation
└── test/               # Integration tests
```

## 🧪 Performance Benchmarks

Tested on MacBook Pro M3 Max:

| Algorithm | Speed | Use Case |
|-----------|-------|----------|
| xxHash64 | 3.5 GB/s | Daily workflows, speed critical |
| MD5 | 350 MB/s | Industry standard, client delivery |
| SHA256 | 180 MB/s | Long-term archive, forensic |

Transfer speeds (USB 3.1):
- SSD → SSD: 400-500 MB/s
- CFast → SSD: 300-400 MB/s

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).
```bash
# Setup development environment
git clone https://github.com/Mald0r0r000/LOOT.git
cd LOOT
make test

# Run tests
make test

# Build
make build
```

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Credits

Developed by **Mald0r0r000**

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [xxHash](https://github.com/cespare/xxhash) - Ultra-fast hashing
- [fpdf](https://github.com/go-pdf/fpdf) - PDF generation

Inspired by industry tools like ShotPut Pro and Silverstack, but built for the modern, AI-powered workflow era.

## ⭐ Star History

[![Star History Chart](https://api.star-history.com/svg?repos=Mald0r0r000/LOOT&type=Date)](https://star-history.com/#Mald0r0r000/LOOT&Date)

## 📢 Community

- [GitHub Discussions](https://github.com/Mald0r0r000/LOOT/discussions)
- [Issue Tracker](https://github.com/Mald0r0r000/LOOT/issues)
- Twitter: [@LOOT_DIT](https://twitter.com/LOOT_DIT) (placeholder)

---

Made with ❤️ for the film industry. Free forever.
Success criteria:

 Badges fonctionnels
 Comparaison claire vs ShotPut Pro
 Screenshots/GIFs inclus
 Links vers docs fonctionnent
 Professional et crédible


🎯 PHASE 5: Release & Marketing (Semaine 5)
TASK 5.1: Prep for v1.0.0 Release
Priority: 🔴 CRITICAL
Estimated time: 3h
Dependencies: Toutes les tâches précédentes
Checklist complet:
Code Quality:

 Tous les tests passent (make test)
 Coverage >= 60%
 Lint sans erreurs (golangci-lint run)
 Pas de TODOs critiques dans le code
 Version string correcte dans main.go

Documentation:

 README.md complet avec badges
 INSTALLATION.md avec toutes les méthodes
 USAGE.md avec tous les flags
 WORKFLOWS.md avec exemples réels
 AUTOMATION.md avec AI agent examples
 API.md avec JSON schema complet
 TROUBLESHOOTING.md avec solutions
 CONTRIBUTING.md pour contributeurs

Distribution:

 GitHub release avec binaries (arm64 + amd64)
 Checksums SHA256 pour chaque binary
 Homebrew formula testée et fonctionnelle
 Formula dans tap repository
 Release notes générées

Testing:

 Tests manuels sur macOS Intel
 Tests manuels sur macOS Apple Silicon
 Integration tests passent
 Dry-run fonctionne correctement
 Resume fonctionne correctement
 Multi-destination vérifié
 Tous les algorithmes testés

Files to create:

CHANGELOG.md
CONTRIBUTING.md
.github/ISSUE_TEMPLATE/bug_report.md
.github/ISSUE_TEMPLATE/feature_request.md

CHANGELOG.md:
markdown# Changelog

All notable changes to LOOT will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-02-15

### Added
- Initial public release
- Interactive TUI mode with file browser
- CLI mode with comprehensive flags
- Multi-algorithm hash support (xxHash64, MD5, SHA256)
- Dual-hash mode (xxHash64 + MD5 simultaneously)
- Multi-destination offload
- Job persistence and resume functionality
- JSON output for automation and AI agents
- Dry-run mode
- PDF report generation
- MHL (Media Hash List) generation
- Real-time progress tracking
- Homebrew installation support
- Comprehensive documentation

### Performance
- xxHash64: 3.5 GB/s on Apple Silicon
- MD5: 350 MB/s on Apple Silicon
- SHA256: 180 MB/s on Apple Silicon

### Platforms
- macOS 12+ (Apple Silicon and Intel)

## [Unreleased]

### Planned Features
- Linux support
- Windows support (WSL)
- Cloud upload integration (S3, Backblaze)
- Metadata extraction (EXIF, camera info)
- Watch folder automation
- GUI application
- Plugin system for extensibility

---

[1.0.0]: https://github.com/Mald0r0r000/LOOT/releases/tag/v1.0.0
CONTRIBUTING.md:
markdown# Contributing to LOOT

Thank you for considering contributing to LOOT! We welcome contributions from the community.

## Code of Conduct

Be respectful, inclusive, and professional. We're all here to make better tools for the film industry.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/Mald0r0r000/LOOT/issues)
2. If not, create a new issue with:
   - Clear title
   - Steps to reproduce
   - Expected vs actual behavior
   - LOOT version (`loot --version`)
   - OS version
   - Relevant logs (`loot --verbose ...`)

### Suggesting Features

1. Check [Discussions](https://github.com/Mald0r0r000/LOOT/discussions) first
2. Create a feature request issue with:
   - Use case explanation
   - Proposed solution
   - Alternatives considered

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`make test`)
6. Ensure code is linted (`golangci-lint run`)
7. Commit with clear messages
8. Push to your fork
9. Create a Pull Request

#### PR Guidelines

- One feature/fix per PR
- Include tests
- Update documentation if needed
- Keep commits focused and atomic
- Follow existing code style
- Add yourself to CONTRIBUTORS.md

## Development Setup
```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/LOOT.git
cd LOOT

# Install dependencies
go mod download

# Run tests
make test

# Build
make build

# Run
./loot --version
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write clear, descriptive variable names
- Comment complex logic
- Keep functions small and focused

## Testing

- Write unit tests for new functionality
- Add integration tests for workflows
- Aim for >60% coverage
- Test on both Intel and Apple Silicon if possible

## Documentation

- Update README.md for user-facing changes
- Update relevant docs/ files
- Add examples for new features
- Keep CHANGELOG.md updated

## Release Process

(For maintainers)

1. Update version in `cmd/loot/main.go`
2. Update CHANGELOG.md
3. Commit: `git commit -m "Release vX.Y.Z"`
4. Tag: `git tag vX.Y.Z`
5. Push: `git push origin main --tags`
6. GitHub Actions will create release
7. Update Homebrew formula with new SHA256

## Questions?

Join [Discussions](https://github.com/Mald0r0r000/LOOT/discussions) or open an issue.

Thank you for contributing! 🎬
Bug Report Template:
markdown<!-- .github/ISSUE_TEMPLATE/bug_report.md -->
---
name: Bug Report
about: Report a bug in LOOT
title: '[BUG] '
labels: bug
assignees: ''
---

## Bug Description
<!-- A clear description of the bug -->

## Steps to Reproduce
1. 
2. 
3. 

## Expected Behavior
<!-- What should happen -->

## Actual Behavior
<!-- What actually happens -->

## Environment
- LOOT version: <!-- run `loot --version` -->
- OS: <!-- e.g., macOS 14.2, Apple Silicon -->
- Installation method: <!-- Homebrew / Manual / Source -->

## Logs
<!-- If applicable, include verbose output -->
```bash
loot --verbose  2>&1
```

<!-- Paste output here -->

## Additional Context
<!-- Screenshots, related issues, etc. -->
Feature Request Template:
markdown<!-- .github/ISSUE_TEMPLATE/feature_request.md -->
---
name: Feature Request
about: Suggest a feature for LOOT
title: '[FEATURE] '
labels: enhancement
assignees: ''
---

## Feature Description
<!-- Clear description of the feature -->

## Use Case
<!-- Why do you need this feature? -->

## Proposed Solution
<!-- How should it work? -->

## Alternatives Considered
<!-- Other ways to solve this problem -->

## Additional Context
<!-- Screenshots, examples from other tools, etc. -->
```

**Success criteria:**
- [ ] v1.0.0 tag créé
- [ ] Release GitHub publiée
- [ ] Binaries uploadés
- [ ] Homebrew formula updated
- [ ] CHANGELOG complet
- [ ] Issue templates en place

---

### **TASK 5.2: Marketing & Launch**
**Priority:** 🟡 MEDIUM  
**Estimated time:** Ongoing  
**Dependencies:** TASK 5.1

**Launch Strategy:**

**Pre-Launch (1 semaine avant):**
1. **Beta testing**
   - Reach out à 5-10 DITs pour tester
   - Gather feedback
   - Fix critical bugs

2. **Content création**
   - Record demo GIF/video
   - Write launch blog post
   - Prepare social media posts

3. **Community setup**
   - Enable GitHub Discussions
   - Create Twitter account (optional)
   - Prepare FAQ based on beta feedback

**Launch Day:**

1. **GitHub**
   - Publish v1.0.0 release
   - Pin announcement issue

2. **Hacker News**
```
   Title: "Show HN: LOOT – Free, open-source alternative to ShotPut Pro for media offload"
   
   Body:
   Hi HN! I'm a DIT (Digital Imaging Technician) and got tired of paying $99/year for media offload tools. I built LOOT as a free, open-source alternative.
   
   Key features:
   - 10x faster verification (xxHash64 at 3.5 GB/s)
   - Resume interrupted transfers
   - AI-ready JSON output
   - Multi-destination backup
   - Industry-standard MHL reports
   
   Built with Go and Bubble Tea. Works great with AI agents like Claude for workflow automation.
   
   Would love feedback from the community!
```

3. **Product Hunt**
   - Submit product
   - Prepare tagline: "Free, open-source media offload tool for film professionals"
   - Upload screenshots/demo

4. **Reddit**
   - r/cinematography
   - r/videography
   - r/editors
   - r/DataHoarder
```
   Title: [Tool] I built a free alternative to ShotPut Pro
   
   Body:
   Hey r/cinematography! I'm a DIT and created LOOT, a free and open-source media offload tool.
   
   Why I built it:
   - ShotPut Pro costs $99/year
   - Wanted CLI for automation
   - AI agents need JSON output
   
   Features:
   - xxHash64 (10x faster than MD5)
   - Resume interrupted transfers
   - Multi-destination
   - MHL & PDF reports
   - Homebrew install: brew install loot
   
   GitHub: [link]
   
   Would appreciate feedback from fellow DITs!

Social Media

Twitter thread with demo GIF
LinkedIn post targeting film industry



Post-Launch (First Month):

Content marketing

Blog post: "How to automate your DIT workflow with AI agents"
Tutorial video on YouTube
Guest post on film tech blogs


Community engagement

Answer every GitHub issue within 24h
Join cinematography Discord/Slack
Participate in r/cinematography discussions


Feature improvements

Implement most-requested features
Fix reported bugs
v1.1.0 release with improvements



Success Metrics:

 100+ GitHub stars in first week
 50+ Homebrew installs
 5+ community contributions
 Featured on at least 1 film tech blog
 Positive feedback from beta testers


📊 COMPLETE ROADMAP SUMMARY
Phase 1: Fondations CLI (Semaine 1-2)

✅ TASK 1.1: Versioning system
✅ TASK 1.2: Config package
✅ TASK 1.3: Integration config dans main
✅ TASK 1.4: Multi-algorithmes hash

Phase 2: JSON & Job Management (Semaine 2-3)

✅ TASK 2.1: JSON output system
✅ TASK 2.2: Job persistence
✅ TASK 2.3: Resume functionality
✅ TASK 2.4: Dry-run mode

Phase 3: Testing & Quality (Semaine 3-4)

✅ TASK 3.1: Testing infrastructure
✅ TASK 3.2: Integration tests
✅ TASK 3.3: CI/CD GitHub Actions

Phase 4: Documentation & Distribution (Semaine 4-5)

✅ TASK 4.1: Documentation complète
✅ TASK 4.2: Homebrew formula
✅ TASK 4.3: README final

Phase 5: Release & Marketing (Semaine 5)

✅ TASK 5.1: Prep v1.0.0
✅ TASK 5.2: Launch strategy


🎯 Pour Windsurf/Cursor
Comment utiliser cette roadmap:

Commencer par Phase 1, Task 1.1
Copier la section TASK complète dans l'IDE
Suivre Implementation step-by-step
Tester avec les Tests to validate
Cocher Success criteria
Passer à la TASK suivante

Chaque TASK contient:

Context (pourquoi)
Files to create/modify (quoi)
Implementation (code complet)
Tests to validate (comment vérifier)
Success criteria (définition de "done")

Tips pour l'agent:

Ne pas skip les tests
Lire la section Context avant de coder
Utiliser les Success criteria comme checklist
Si bloqué, relire le Context
Chaque TASK est atomique et indépendante
