# LOOT 💰

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)

**LOOT** is a high-performance, professional media offload tool built for the terminal. It is designed to be a lightweight, open-source alternative to industry standards like ShotPut Pro or Silverstack, providing reliable verification and reporting for DITs (Digital Imaging Technicians) and media professionals.

```text
██╗      ██████╗  ██████╗ ████████╗
██║     ██╔══/██╗██╔══/██╗╚══██╔══╝
██║     ██║ / ██║██║ / ██║   ██║   
██║     ██║/  ██║██║/  ██║   ██║   
███████╗╚██████╔╝╚██████╔╝   ██║   
╚══════╝ ╚═════╝  ╚═════╝    ╚═╝   
```

## ✨ Features

- **🚀 TUI Dashboard**: A modern, interactive terminal user interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).
- **🔒 Checksum Verification**: Supports **xxHash64**, **MD5**, and **SHA256** for reliable bit-for-bit verification.
- **⚡ Parallel Processing**: High-performance copy engine with configurable concurrency.
- **📂 Custom File Browser**: Navigate your file system naturally with a dual-pane interface.
- **📑 MHL Support**: Automatically generates **Media Hash List (MHL)** files.
- **📄 PDF Reports**: Generates detailed PDF reports proving the integrity of the copy.
- **💾 Volume Awareness**: Auto-detects mounted volumes in `/Volumes`.
- **🔄 Job Management**: Queue, Pause, Cancel, and Retry offload jobs.
- **⏯️ Resume Capability**: Skip existing verified files to resume interrupted transfers.

## 📦 Installation

### Prerequisites
- Go 1.21 or higher

### Build from Source
```bash
git clone https://github.com/Mald0r0r000/LOOT.git
cd loot
go build -o loot cmd/loot/main.go
mv loot /usr/local/bin/ # Optional
```

## 🎮 Usage

Run the tool:
```bash
loot
```

### Controls

| Key | Action |
| :--- | :--- |
| **↑ / ↓** | Navigate Menu / Lists |
| **← / →** | Navigate Directory |
| **Enter** | Enter Directory / Select Option |
| **Space** | **Select** Source / Destination / Toggle Setting |
| **Tab** | **Toggle Job Manager** |
| **x / X** | Cancel Active Job |
| **r / R** | Retry Failed/Cancelled Job |
| **Esc / q** | Back / Cancel / Quit |

### Workflow
1. **Settings (Optional)**: Select your preferred hash algorithm (xxHash, MD5, SHA256).
2. **Select Source**: Browse to your camera card or source folder and press `Space`.
3. **Select Destination**: Browse to your backup drive and press `Space`.
4. **Copy & Verify**: LOOT handles the transfer, verification, and report generation automatically.
5. **Monitor**: Use the Job Manager to track progress or cancel operations.

### CLI Mode

LOOT can also be used in headless mode for automation:

```bash
loot --source /path/to/card --dest /path/to/backup --md5 --concurrency 8 --json
```

**Flags:**
- `--algorithm <algo>`: Set hash algorithm (xxhash64, md5, sha256)
- `--dual-hash`: Calculate both xxHash and MD5
- `--concurrency <N>`: Set number of parallel workers (Default: 4)
- `--resume`: Skip existing files that match size/time
- `--no-verify`: Skip verification phase
- `--json`: Output results in JSON format
- `--quiet`: Suppress output


## 🛠️ Roadmap

- [x] **Recursive Copy**: Full support for deep directory structures.
- [x] **Multi-Target**: Offload to multiple drives simultaneously.
- [x] **xxHash128 / MD5**: Support for additional checksum algorithms (MD5, SHA256).
- [x] **Resume**: Ability to resume interrupted transfers.
- [ ] **Verification-Only Mode**: Verify existing backups without copying.
- [ ] **xxHash128**: Implement xxHash128 support.

## 📝 License

Distributed under the MIT License. See `LICENSE` for more information.

## 👨‍💻 Credits

Developed by **Mald0r0r000**.
Built with Go and the Charm ecosystem.
