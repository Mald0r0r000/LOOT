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
- **📂 File Browser & Volume Awareness**: Direct navigation and auto-detection of `/Volumes`.
- **📑 MHL & PDF Reports**: Generates industry-standard **Media Hash List (MHL)** and detailed **PDF Reports**.
- **🎥 Metadata Extraction**: Extracts technical metadata (Resolution, Codec, FPS) from video files (supports R3D, MOV, MXF).
- **🛡️ Merge Mode**: Safe copy logic that detects existing destinations and merges content instead of overwriting.
- **🧪 Dry Run**: Simulate transfers without writing to disk to check space and file counts.
- **🔄 Resume Capability**: Smartly skips existing valid files to resume interrupted jobs.
- **⚙️ Runtime Configuration**: Adjust Hash Algo, Metadata Mode, and Job Name on the fly via Settings.

## 📦 Installation
See [INSTALL.md](INSTALL.md) for detailed instructions on binaries, Go install, and building from source.

### Homebrew (macOS)
```bash
brew tap Mald0r0r000/loot
brew install loot
```

## ❓ Troubleshooting
Encountering issues? Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for solutions to common problems.

## 🎮 Usage

Run the tool interactively:
```bash
loot
```

### Key Bindings

| Key | Action |
| :--- | :--- |
| **Space** | **Select** Source / Destination / Toggle Setting |
| **Enter** | Open Directory / Confirm Action |
| **Tab** | **Toggle Job Manager** |
| **Esc** | Back / Settings Menu (from Root) |
| **x** | Cancel Active Job |
| **r** | Retry Failed Job |
| **q** | Quit (if no active jobs) |

### Operations
1. **Source**: Navigate and select source.
2. **Destination**: Navigate and select destination(s). You can select multiple.
3. **Settings**: Press `Esc` at the root menu to configure Hash Algo, Metadata Mode, etc.
4. **Confirm**: Review the summary. If destination exists, **Merge Mode** will be offered.
5. **Monitor**: Watch progress, speed, and ETA.

### CLI Mode & Flags
```bash
loot --source /card --dest /backup --md5 --job-name "Day01" --dry-run
```

**Common Flags:**
- `--algorithm`: `xxhash64` (default), `md5`, `sha256`
- `--metadata-mode`: `hybrid` (default), `header`, `exiftool`, `off`
- `--concurrency`: Number of workers (default 4)
- `--dry-run`: Simulate only
- `--resume`: Skip existing files

## 🛠️ Roadmap
- [x] **Metadata Extraction**
- [x] **Dry Run Mode**
- [x] **Merge/Resume Logic**
- [ ] **Verification-Only Mode**
- [ ] **xxHash128 Support**

## 📝 License
Distributed under the MIT License.

## 👨‍💻 Credits
Developed by **Mald0r0r000**.

