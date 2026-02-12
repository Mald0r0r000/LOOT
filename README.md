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
- **🔒 Checksum Verification**: Uses **xxHash64** for extremely fast and reliable bit-for-bit verification.
- **📂 Custom File Browser**: Navigate your file system naturally with a dual-pane interface (Source & Destination).
- **📑 MHL Support**: Automatically generates standard **Media Hash List (MHL)** files (XML) for workflow compatibility.
- **📄 PDF Reports**: Generates detailed PDF reports proving the integrity of the copy.
- **💾 Volume Awareness**: Auto-detects mounted volumes in `/Volumes` for quick selection.

## 📦 Installation

### Prerequisites
- Go 1.21 or higher

### Build from Source
```bash
git clone https://github.com/antoinebedos/loot.git
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
| **↑ / ↓** | Navigate lists |
| **→ / Enter** | Enter directory |
| **←** | Go up / Back |
| **Space** | **Select** current folder as Source or Destination |
| **r** | Refresh volume/file list |
| **q / Esc** | Cancel / Back / Quit |

### Workflow
1. **Select Source**: Browse to your camera card or source folder and press `Space`.
2. **Select Destination**: Browse to your backup drive and press `Space`.
3. **Copy & Verify**: LOOT handles the transfer, verification, and report generation automatically.

## 🛠️ Roadmap

- [ ] **Recursive Copy**: Full support for deep directory structures (Currently strictly flat file/folder logic for MVP).
- [ ] **Multi-Target**: Offload to multiple drives simultaneously.
- [ ] **xxHash128 / MD5**: Support for additional checksum algorithms.
- [ ] **Resume**: Ability to resume interrupted transfers.

## 📝 License

Distributed under the MIT License. See `LICENSE` for more information.

## 👨‍💻 Credits

Developed by **Mald0r0r000**.
Built with Go and the Charm ecosystem.
