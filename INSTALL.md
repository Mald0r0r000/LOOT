# Installation Guide

## Option 1: Binary Release (Recommended)
Download the pre-compiled binary for your operating system from the [Releases](https://github.com/Mald0r0r000/LOOT/releases) page.

1.  Download the archive for your OS (macOS/Linux).
2.  Extract the archive.
3.  Move the binary to your path:
    ```bash
    sudo mv loot /usr/local/bin/
    chmod +x /usr/local/bin/loot
    ```

## Option 2: Go Install
If you have Go installed (1.21+):

```bash
go install github.com/Mald0r0r000/LOOT/cmd/loot@latest
```
Ensure `$(go env GOPATH)/bin` is in your `PATH`.

## Option 3: Build from Source

1.  Clone the repository:
    ```bash
    git clone https://github.com/Mald0r0r000/LOOT.git
    cd LOOT
    ```

2.  Build using Make:
    ```bash
    make build
    ```

3.  Run:
    ```bash
    ./bin/loot
    ```
