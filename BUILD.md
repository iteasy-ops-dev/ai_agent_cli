# 🔨 Build Guide

This document explains how to build `syseng-agent` for multiple platforms.

## 🚀 Quick Start

### Using Build Scripts

**Unix/Linux/macOS:**
```bash
./build.sh
```

**Windows:**
```cmd
build.bat
```

### Using Make (Unix/Linux/macOS)

```bash
# Build for all platforms
make build-all

# Build for current platform only
make build

# Build for specific platforms
make build-windows
make build-macos
make build-linux
```

## 📦 Output Structure

After building, you'll find binaries in the `dist/` directory:

```
dist/
├── windows/
│   ├── syseng-agent-windows-amd64.exe
│   ├── syseng-agent-windows-amd64.exe.zip
│   ├── syseng-agent-windows-386.exe
│   └── syseng-agent-windows-386.exe.zip
├── macos/
│   ├── syseng-agent-macos-amd64
│   ├── syseng-agent-macos-amd64.tar.gz
│   ├── syseng-agent-macos-arm64
│   └── syseng-agent-macos-arm64.tar.gz
├── linux/
│   ├── syseng-agent-linux-amd64
│   ├── syseng-agent-linux-amd64.tar.gz
│   ├── syseng-agent-linux-arm64
│   ├── syseng-agent-linux-arm64.tar.gz
│   ├── syseng-agent-linux-386
│   └── syseng-agent-linux-386.tar.gz
└── checksums.txt
```

## 🎯 Supported Platforms

| OS | Architecture | Binary Name |
|---|---|---|
| Windows | amd64 | `syseng-agent-windows-amd64.exe` |
| Windows | 386 | `syseng-agent-windows-386.exe` |
| macOS | amd64 (Intel) | `syseng-agent-macos-amd64` |
| macOS | arm64 (M1/M2) | `syseng-agent-macos-arm64` |
| Linux | amd64 | `syseng-agent-linux-amd64` |
| Linux | arm64 | `syseng-agent-linux-arm64` |
| Linux | 386 | `syseng-agent-linux-386` |

## ⚙️ Build Options

### Environment Variables

- `VERSION`: Set the build version (default: `dev`)
  ```bash
  VERSION=v1.0.0 ./build.sh
  ```

### Build Script Commands

**build.sh / build.bat:**
- `./build.sh` - Build for all platforms
- `./build.sh clean` - Clean build directory
- `./build.sh help` - Show help

**Makefile:**
- `make help` - Show all available targets
- `make build` - Build for current platform
- `make build-all` - Build for all platforms
- `make clean` - Clean build artifacts
- `make test` - Run tests
- `make install` - Install to `$GOPATH/bin`

## 🔍 Version Information

Built binaries include version information:

```bash
./syseng-agent --version
```

Output:
```
syseng-agent v1.0.0
Built: 2024-01-15T10:30:00Z
Commit: abc1234
```

## 🛠️ Development Builds

For development, use:

```bash
# Quick build for current platform
make build

# Build and run
make dev

# Watch for changes (requires entr)
make watch
```

## 📋 Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/dl/)
- **Git** - For commit information
- **tar** (Unix/Linux) - For creating archives
- **PowerShell** (Windows) - For ZIP creation and checksums

### Optional Tools

- **make** - For using Makefile
- **golangci-lint** - For linting (`make lint`)
- **entr** - For file watching (`make watch`)

## 🐳 Docker Build

If you have Docker:

```bash
make docker-build
```

## 🚨 Troubleshooting

### Common Issues

1. **"Go not found"**
   - Install Go from https://golang.org/dl/
   - Ensure Go is in your PATH

2. **"Permission denied" on build.sh**
   ```bash
   chmod +x build.sh
   ```

3. **PowerShell execution policy (Windows)**
   ```powershell
   Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
   ```

### Clean Build

If you encounter issues:

```bash
# Clean everything
make clean
# or
./build.sh clean

# Rebuild
make build-all
```

## 📝 Build Scripts Features

- ✅ Cross-platform compilation
- ✅ Automatic archive creation (ZIP/tar.gz)
- ✅ Checksum generation (SHA256)
- ✅ Version injection
- ✅ Build time recording
- ✅ Git commit tracking
- ✅ Colored output
- ✅ Error handling
- ✅ Clean up functionality