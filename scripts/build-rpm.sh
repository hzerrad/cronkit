#!/bin/bash
set -e

# Build RPM package for cronkit
# Usage: ./scripts/build-rpm.sh [VERSION] [ARCH]
# Example: ./scripts/build-rpm.sh 0.1.0 x86_64

VERSION=${1:-"0.1.0"}
ARCH=${2:-"x86_64"}

# Map Go architecture to RPM architecture
case "$ARCH" in
  amd64)
    RPM_ARCH="x86_64"
    ;;
  arm64)
    RPM_ARCH="aarch64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

PACKAGE_NAME="cronkit"
RPM_DIR="dist/rpm"
mkdir -p "${RPM_DIR}/BUILD" "${RPM_DIR}/RPMS" "${RPM_DIR}/SOURCES" "${RPM_DIR}/SPECS" "${RPM_DIR}/SRPMS"

echo "Building RPM package for ${PACKAGE_NAME} ${VERSION} (${RPM_ARCH})..."

# Download source tarball
SOURCE_URL="https://github.com/hzerrad/cronkit/archive/v${VERSION}.tar.gz"
SOURCE_FILE="${RPM_DIR}/SOURCES/cronkit-${VERSION}.tar.gz"

echo "Downloading source from ${SOURCE_URL}..."
if ! curl -Lf -o "${SOURCE_FILE}" "${SOURCE_URL}"; then
    echo "Error: Failed to download source. Make sure the release exists."
    exit 1
fi

# Download binary
BINARY_URL="https://github.com/hzerrad/cronkit/releases/download/v${VERSION}/cronkit-linux-${ARCH}"
BINARY_PATH="${RPM_DIR}/SOURCES/cronkit-linux-${ARCH}"

echo "Downloading binary from ${BINARY_URL}..."
if ! curl -Lf -o "${BINARY_PATH}" "${BINARY_URL}"; then
    echo "Error: Failed to download binary. Make sure the release exists."
    exit 1
fi

chmod +x "${BINARY_PATH}"

# Create spec file
cat > "${RPM_DIR}/SPECS/cronkit.spec" <<EOF
Name:           ${PACKAGE_NAME}
Version:        ${VERSION}
Release:        1%{?dist}
Summary:        Make cron human again - CLI tool for cron job management
License:        Apache-2.0
URL:            https://github.com/hzerrad/cronkit
Source0:        https://github.com/hzerrad/cronkit/archive/v%{version}.tar.gz

%description
Cronkit is a command-line tool that makes cron jobs human-readable, auditable, and visual.
It converts confusing cron syntax into plain English, generates upcoming run schedules,
provides ASCII timeline visualizations, and validates crontabs with severity levels
and diagnostic codes.

%prep
%setup -q -n cronkit-%{version}

%build
# Binary is pre-built, no build step needed

%install
mkdir -p %{buildroot}/usr/bin
cp ${RPM_DIR}/SOURCES/cronkit-linux-${ARCH} %{buildroot}/usr/bin/cronkit
chmod +x %{buildroot}/usr/bin/cronkit

%files
/usr/bin/cronkit

%changelog
* $(date '+%a %b %d %Y') hzerrad <your-email@example.com> - ${VERSION}-1
- Initial release
EOF

# Build RPM (requires rpmbuild)
if ! command -v rpmbuild &> /dev/null; then
    echo "Error: rpmbuild is not installed. Install with:"
    echo "  - Fedora/RHEL: sudo dnf install rpm-build"
    echo "  - Ubuntu/Debian: sudo apt install rpm"
    exit 1
fi

echo "Building RPM package..."
rpmbuild --define "_topdir $(pwd)/${RPM_DIR}" \
         --define "_arch ${RPM_ARCH}" \
         -bb "${RPM_DIR}/SPECS/cronkit.spec"

# Find the built RPM
RPM_FILE=$(find "${RPM_DIR}/RPMS" -name "*.rpm" | head -1)
if [ -n "$RPM_FILE" ]; then
    # Copy to dist directory with a cleaner name
    cp "$RPM_FILE" "dist/${PACKAGE_NAME}-${VERSION}-1.${RPM_ARCH}.rpm"
    echo "✓ Package built: dist/${PACKAGE_NAME}-${VERSION}-1.${RPM_ARCH}.rpm"
else
    echo "Error: RPM file not found"
    exit 1
fi

