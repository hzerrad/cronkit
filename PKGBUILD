# Maintainer: hzerrad <your-email@example.com>
pkgname=cronkit
pkgver=0.1.0
pkgrel=1
pkgdesc="Make cron human again - CLI tool for cron job management"
arch=('x86_64' 'aarch64')
url="https://github.com/hzerrad/cronkit"
license=('Apache')
depends=()
makedepends=('go')
source=("${pkgname}-${pkgver}.tar.gz::https://github.com/hzerrad/cronkit/archive/v${pkgver}.tar.gz")
sha256sums=('9c647f092c1a0c4b7eeb1ec47108e476dbd0cf1abc235bf0344af4db86799e05')

build() {
  cd "${pkgname}-${pkgver}"
  make build
}

package() {
  cd "${pkgname}-${pkgver}"
  install -Dm755 bin/cronkit "${pkgdir}/usr/bin/cronkit"
}

