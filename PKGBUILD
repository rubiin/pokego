# Maintainer: Rubin Bhandari <roobin.bhandari@gmail.com>
pkgname=pokego-git
_pkgname=pokego
pkgrel=1
pkgdesc="CLI utility that prints unicode sprites of pokemon to the terminal"
arch=('any')
url="https://github.com/rubiin/pokego.git"
license=('MIT')
depends=('coreutils' 'go')
makedepends=('git')
source=("$_pkgname::git+$url")
md5sums=('SKIP')

pkgver() {
  cd "$_pkgname"
  printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short=8 HEAD)"
}

build() {
    cd "$_pkgname"
    go build -o pokego
}

package() {
	  cd "$_pkgname"
    rm -rf "$pkgdir/usr/bin/$_pkgname"
    install -Dm755 pokego "$pkgdir/usr/bin/pokego"
}
