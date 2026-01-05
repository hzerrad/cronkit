# Maintainer: hzerrad <your-email@example.com>
pkgname=cronic
pkgver=0.1.0
pkgrel=1
pkgdesc="Make cron human again - CLI tool for cron job management"
arch=('x86_64' 'aarch64')
url="https://github.com/hzerrad/cronic"
license=('Apache')
depends=()
makedepends=('go')
source=("${pkgname}-${pkgver}.tar.gz::https://github.com/hzerrad/cronic/archive/v${pkgver}.tar.gz")
sha256sums=('9c647f092c1a0c4b7eeb1ec47108e476dbd0cf1abc235bf0344af4db86799e05')

build() {
  cd "${pkgname}-${pkgver}"
  make build
}

package() {
  cd "${pkgname}-${pkgver}"
  install -Dm755 bin/cronic "${pkgdir}/usr/bin/cronic"
}

