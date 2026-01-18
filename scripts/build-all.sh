#!/bin/bash
set -e

VERSION=${1:-"dev"}
OUTPUT_DIR="dist"

echo "Building micro-git version: $VERSION"

# Create output directory
mkdir -p $OUTPUT_DIR

# Build for different platforms
platforms=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r -a parts <<< "$platform"
    GOOS="${parts[0]}"
    GOARCH="${parts[1]}"
    
    output_name="micro-git-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        output_name+=".exe"
    fi
    
    echo "Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X main.version=$VERSION" -o "$OUTPUT_DIR/$output_name"
done

# Create checksums
cd $OUTPUT_DIR
sha256sum micro-git-* > checksums.txt
cd ..

echo "Build complete! Binaries are in $OUTPUT_DIR/"
ls -lh $OUTPUT_DIR/
