#!/bin/bash

# Syseng-Agent Cross-Platform Build Script
# Supports: Windows, macOS, Linux for multiple architectures

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build configuration
APP_NAME="iteasy-ai-agent"
BUILD_DIR="dist"
VERSION=${VERSION:-"dev"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=${GITHUB_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}

# Platform configurations (using simpler approach for better compatibility)
PLATFORMS="windows/amd64:windows-amd64.exe windows/386:windows-386.exe darwin/amd64:macos-amd64 darwin/arm64:macos-arm64 linux/amd64:linux-amd64 linux/arm64:linux-arm64 linux/386:linux-386"

print_header() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                ItEasy AI Agent Build Script                 ║"
    echo "║                   Cross-Platform Builder                    ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_build_info() {
    echo -e "${BLUE}Build Information:${NC}"
    echo "  Version: $VERSION"
    echo "  Build Time: $BUILD_TIME"
    echo "  Git Commit: $GIT_COMMIT"
    echo "  Go Version: $(go version)"
    echo ""
}

clean_build_dir() {
    print_info "Cleaning build directory..."
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"/{windows,macos,linux}
}

build_platform() {
    local goos=$1
    local goarch=$2
    local output_name=$3
    local platform_name="$goos/$goarch"
    
    print_info "Building for $platform_name..."
    
    # Determine output directory
    local output_dir
    case $goos in
        "windows") output_dir="$BUILD_DIR/windows" ;;
        "darwin") output_dir="$BUILD_DIR/macos" ;;
        "linux") output_dir="$BUILD_DIR/linux" ;;
    esac
    
    local output_path="$output_dir/$APP_NAME-$output_name"
    
    # Build command with ldflags for version info
    local ldflags="-X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT"
    
    env GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build \
        -ldflags="$ldflags" \
        -o "$output_path" \
        .
    
    if [ $? -eq 0 ]; then
        local file_size=$(du -h "$output_path" | cut -f1)
        print_info "✓ Built $platform_name ($file_size)"
        
        # Create compressed archive
        create_archive "$goos" "$output_path" "$output_name"
    else
        print_error "✗ Failed to build $platform_name"
        return 1
    fi
}

create_archive() {
    local goos=$1
    local binary_path=$2
    local output_name=$3
    
    local archive_dir=$(dirname "$binary_path")
    local binary_name=$(basename "$binary_path")
    
    cd "$archive_dir"
    
    if [ "$goos" = "windows" ]; then
        # Create ZIP for Windows
        local zip_name="$APP_NAME-$output_name.zip"
        zip -q "$zip_name" "$binary_name"
        print_info "  → Created $zip_name"
    else
        # Create tar.gz for Unix-like systems
        local tar_name="$APP_NAME-$output_name.tar.gz"
        tar -czf "$tar_name" "$binary_name"
        print_info "  → Created $tar_name"
    fi
    
    cd - > /dev/null
}

generate_checksums() {
    print_info "Generating checksums..."
    
    # Create checksums file
    local checksum_file="$BUILD_DIR/checksums.txt"
    > "$checksum_file"
    
    for dir in "$BUILD_DIR"/*; do
        if [ -d "$dir" ]; then
            for file in "$dir"/*; do
                if [ -f "$file" ]; then
                    local filename=$(basename "$file")
                    local dirname=$(basename "$dir")
                    if [[ "$filename" == *.zip ]] || [[ "$filename" == *.tar.gz ]]; then
                        local checksum=$(shasum -a 256 "$file" | cut -d' ' -f1)
                        echo "$checksum  $dirname/$filename" >> "$checksum_file"
                    fi
                fi
            done
        fi
    done
    
    print_info "✓ Checksums saved to checksums.txt"
}

show_build_summary() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗"
    echo -e "║                     Build Summary                            ║"
    echo -e "╚══════════════════════════════════════════════════════════════╝${NC}"
    
    echo -e "${BLUE}Built binaries:${NC}"
    find "$BUILD_DIR" -name "$APP_NAME-*" -not -name "*.zip" -not -name "*.tar.gz" | while read file; do
        local size=$(du -h "$file" | cut -f1)
        local rel_path=$(echo "$file" | sed "s|$BUILD_DIR/||")
        echo "  $rel_path ($size)"
    done
    
    echo ""
    echo -e "${BLUE}Archives:${NC}"
    find "$BUILD_DIR" -name "*.zip" -o -name "*.tar.gz" | while read file; do
        local size=$(du -h "$file" | cut -f1)
        local rel_path=$(echo "$file" | sed "s|$BUILD_DIR/||")
        echo "  $rel_path ($size)"
    done
    
    echo ""
    echo -e "${GREEN}Build completed successfully!${NC}"
    echo -e "Output directory: ${BLUE}$BUILD_DIR${NC}"
}

main() {
    print_header
    print_build_info
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we're in a Go module
    if [ ! -f "go.mod" ]; then
        print_error "go.mod not found. Please run this script from the project root."
        exit 1
    fi
    
    clean_build_dir
    
    # Build for all platforms
    local failed_builds=0
    for platform_entry in $PLATFORMS; do
        IFS=':' read -r platform output_name <<< "$platform_entry"
        IFS='/' read -r goos goarch <<< "$platform"
        
        if ! build_platform "$goos" "$goarch" "$output_name"; then
            ((failed_builds++))
        fi
    done
    
    if [ $failed_builds -eq 0 ]; then
        generate_checksums
        show_build_summary
    else
        print_error "$failed_builds builds failed"
        exit 1
    fi
}

# Parse command line arguments
case "${1:-}" in
    "clean")
        print_info "Cleaning build directory..."
        rm -rf "$BUILD_DIR"
        print_info "✓ Cleaned"
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [clean|help]"
        echo ""
        echo "Commands:"
        echo "  (no args)  Build for all platforms"
        echo "  clean      Clean build directory"
        echo "  help       Show this help"
        echo ""
        echo "Environment variables:"
        echo "  VERSION    Set build version (default: dev)"
        ;;
    "")
        main
        ;;
    *)
        print_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac