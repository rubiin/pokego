# Maintainer: Phoney Badger <phoneybadgercode.4ikc7 at simplelogin.co>
pkgname=pokego-git
_pkgname=pokego
pkgver=r8.76ddd43b
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
    go build -o pokego main.go
}

package() {
	  cd "$_pkgname"
    # Creating necessary directories and copying files
    rm -rf "$pkgdir/usr/local/opt/$_pkgname"
    mkdir -p "$pkgdir/usr/local/opt/$_pkgname/assets/colorscripts/regular"
    mkdir -p "$pkgdir/usr/local/opt/$_pkgname/assets/colorscripts/shiny"
    install -Dm755 pokego "$pkgdir/usr/bin/pokego"
    install -Dm644 assets/colorscripts/small/regular/* -t "$pkgdir/usr/local/opt/$_pkgname/assets/colorscripts/small/regular"
    install -Dm644 assets/colorscripts/small/shiny/* -t "$pkgdir/usr/local/opt/$_pkgname/assets/colorscripts/small/shiny"
    install -Dm644 assets/colorscripts/large/regular/* -t "$pkgdir/usr/local/opt/$_pkgname/assets/colorscripts/large/regular"
    install -Dm644 assets/colorscripts/large/shiny/* -t "$pkgdir/usr/local/opt/$_pkgname/assets/colorscripts/large/shiny"
    install -Dm644 assets/pokemon.json "$pkgdir/usr/local/opt/$_pkgname/pokemon.json"
    # install -Dm644 LICENSE.txt "$pkgdir/usr/share/licenses/$_pkgname/LICENSE"
    install -Dm644 README.md "$pkgdir/usr/share/doc/$_pkgname/README.md"
}
