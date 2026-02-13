# LOOT Troubleshooting Guide

This guide covers common issues and solutions when using LOOT.

## Common Errors

### "Destination Exists" Warning
LOOT protects you from accidental data overwrites. If you see this warning:
- **Solution**: Confirm with `Y` (or `Enter`) to enable **Merge/Resume Mode**.
- This will skip existing files that match in size and only copy new/changed files.

### "Permission Denied"
Occurs when reading from source or writing to destination.
- **Solution on macOS**: Ensure LOOT has "Full Disk Access" in System Settings if running from a terminal that lacks permissions.
- **Solution (General)**: Check file ownership with `ls -l` and run with `sudo` if necessary (caution advised).

### "No Space Left on Device"
The destination drive is full.
- **Solution**: Free up space or choose a different destination.
- **Dry Run**: Use `--dry-run` or the Settings menu toggle to simulate the transfer and check if it fits before copying.

### "Metadata Extraction Failed"
LOOT supports multiple metadata modes. If R3D headers or ExifTool fail:
- **Solution**: Switch Metadata Mode in **Settings** (press `Esc`, go to Settings).
    - try `exiftool` (slower but robust)
    - try `off` if you don't need metadata in the report.

### Slower than Expected Performance
- **Hash Algorithm**: XXHash64 is significantly faster than MD5 or SHA256. Checks **Settings**.
- **Concurrency**: Default concurrency is 4. Use `-c 8` or `-c 16` for fast SSDs.
- **Small Files**: Transferring thousands of small files is slower than large files due to overhead.

## Reporting Issues

If you encounter a bug, please run LOOT with the verbose flag and capture the output:
```bash
loot --verbose /source /dest > loot_debug.log 2>&1
```
Include this log when reporting issues.
