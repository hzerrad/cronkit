#!/bin/bash
set -e

# Build Debian package for cronkit
# Usage: ./scripts/build-deb.sh [VERSION] [ARCH]
# Example: ./scripts/build-deb.sh 0.1.0 amd64

VERSION=${1:-"0.1.0"}
ARCH=${2:-"amd64"}

# Map Go architecture to Debian architecture
case "$ARCH" in
  amd64)
    DEB_ARCH="amd64"
    ;;
  arm64)
    DEB_ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

PACKAGE_NAME="cronkit"
PACKAGE_VERSION="$VERSION"
DEB_DIR="dist/deb/${PACKAGE_NAME}_${PACKAGE_VERSION}_${DEB_ARCH}"

echo "Building Debian package for ${PACKAGE_NAME} ${PACKAGE_VERSION} (${DEB_ARCH})..."

# Clean previous build
rm -rf "${DEB_DIR}"

# Create package structure
mkdir -p "${DEB_DIR}/DEBIAN"
mkdir -p "${DEB_DIR}/usr/bin"
mkdir -p "${DEB_DIR}/usr/share/doc/${PACKAGE_NAME}"
mkdir -p "${DEB_DIR}/usr/share/man/man1"

# Download binary from GitHub release or use local build
BINARY_URL="https://github.com/hzerrad/cronkit/releases/download/v${VERSION}/cronkit-linux-${ARCH}"
BINARY_PATH="${DEB_DIR}/usr/bin/cronkit"

echo "Downloading binary from ${BINARY_URL}..."
if ! curl -Lf -o "${BINARY_PATH}" "${BINARY_URL}"; then
    echo "Error: Failed to download binary. Make sure the release exists."
    exit 1
fi

chmod +x "${BINARY_PATH}"

# Create control file
cat > "${DEB_DIR}/DEBIAN/control" <<EOF
Package: ${PACKAGE_NAME}
Version: ${PACKAGE_VERSION}
Section: utils
Priority: optional
Architecture: ${DEB_ARCH}
Maintainer: hzerrad <your-email@example.com>
Description: Make cron human again
 CLI tool that makes cron jobs human-readable, auditable, and visual.
 It converts confusing cron syntax into plain English, generates upcoming
 run schedules, provides ASCII timeline visualizations, and validates crontabs
 with severity levels and diagnostic codes.
Homepage: https://github.com/hzerrad/cronkit
EOF

# Create copyright file
cat > "${DEB_DIR}/usr/share/doc/${PACKAGE_NAME}/copyright" <<EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: cronkit
Source: https://github.com/hzerrad/cronkit

Files: *
Copyright: $(date +%Y) hzerrad
License: Apache-2.0

License: Apache-2.0
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 .
 http://www.apache.org/licenses/LICENSE-2.0
 .
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
EOF

# Create changelog
cat > "${DEB_DIR}/usr/share/doc/${PACKAGE_NAME}/changelog.Debian.gz" <<EOF | gzip -9 > "${DEB_DIR}/usr/share/doc/${PACKAGE_NAME}/changelog.Debian.gz"
cronkit (${PACKAGE_VERSION}) unstable; urgency=medium

  * Initial release

 -- hzerrad <your-email@example.com>  $(date -R)
EOF

# Build package
echo "Building Debian package..."
dpkg-deb --build "${DEB_DIR}" "dist/${PACKAGE_NAME}_${PACKAGE_VERSION}_${DEB_ARCH}.deb"

echo "✓ Package built: dist/${PACKAGE_NAME}_${PACKAGE_VERSION}_${DEB_ARCH}.deb"

