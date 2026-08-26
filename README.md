# pokego

![AUR version](https://img.shields.io/aur/version/pokego-git)

Go version of phoney badger's [pokemon-colorscripts](https://gitlab.com/phoneybadger/pokemon-colorscripts) , with a boost on speed

<img src="logo.png" height=300>

## Table of contents

* [Features](#features)
* [Installation](#installation)
* [Usage](#usage)
  * [Examples](#examples)
* [Credits](#credits)
* [Similar projects](#similar-projects)

## Features

- Includes Pokémon from all generations, along with shiny, mega, Gigantamax, and regional variants
- Print random Pokémon, optionally filtered by generation or range
- Print a specific Pokémon by name, with an optional alternate form

## Installation

### Arch

If you're on Arch, you can also use the AUR:

```sh
yay -S pokego-bin
```

Or alternatively you can manually download the PKGBUILD file from the repository, then run

```sh
makepkg -si
```

### NixOS / Nix

Pokego is available in nixpkgs.

System-wide, add it to `environment.systemPackages`:

```nix
environment.systemPackages = with pkgs; [
  pokego
];
```

Or, per-user with [home-manager](https://github.com/nix-community/home-manager), add it to `home.packages`:

```nix
home.packages = with pkgs; [
  pokego
];
```

### For other Linux Distributions

Download the latest release. Unzip the executable

Then move the executable to your path

```sh
mv pokego ~/.local/bin
```

### Git

You can also clone the repository and compile manually by doing:

```sh
git clone https://github.com/rubiin/pokego.git
cd pokego
just build
```

(Or, without `just`: `go build -ldflags "-s -w" -o pokego`.)

Then move the executable to your path

```sh
mv pokego ~/.local/bin
```

There is also the development package [pokego-git](https://aur.archlinux.org/packages/pokego-bin) that tracks the main branch.

## Usage

Run `pokego --help` to see all options.

```sh
NAME:
   pokego - display Pokémon sprites in color directly in your terminal

USAGE:
   pokego [global options]

GLOBAL OPTIONS:
   --list, -l                  List all Pokémon
   --name string, -n string    Select Pokémon by name
   --form string, -f string    Show alternate form of a Pokémon
   --no-title, --nt            Do not display Pokémon name
   --shiny, -s                 Show shiny version
   --random string, -r string  Show random Pokémon, optionally by generation or range
   --version, -v               Show CLI version
   --help, -h                  show help
```

`--list` prints the base name of every Pokémon (905 in total); alternate forms are
not listed. Use forms with `--name`, e.g. `pokego --name charizard --form mega-y`.

### Examples

Print a specific pokemon

```
pokego --name charizard
```

Print a specific shiny pokemon

```
pokego --name spheal -s
```

Print a specific pokemon

```
pokego --name mudkip
```

Print an alternative form of a pokemon

```
pokego --name blastoise --form mega
```

Print random pokemon from generations 1-3 (range)

```
pokego --random 1-3
```

Print a random pokemon from generations 1,3 and 6

```
pokego --random 1,3,6
```

## Comparision

The start time is the mean of 5 consecutive run using `time` coreutil on my personal laptop[Acer Aspire 5] on `2024/10/06`

| Tool                | Start Time (S)   | Size (MB)    | Language Used                 |
|---------------------|----------------|----------------|-------------------------------|
| **Pokego**          | 0.005          | 27 MB          | Go                            |
| **Pokeget**         | 0.006          | 5 MB           | Rust                          |
| **Krabby**          | 0.016          | 23 MB          | Rust                          |
| **Pokemonscripts**  | 0.060          | 43 MB          | Python                        |

The ~27 MB Pokego binary is dominated by the embedded sprite assets (the raw corpus
is ~29 MB); `go build -ldflags "-s -w"` trims it to ~26 MB.

## Credits

Pokego's Pokémon sprites were sourced from [PokéSprite](https://msikma.github.io/pokesprite/) and transformed into Unicode format using Phoney Badger's [pokemon-generator-scripts](https://gitlab.com/phoneybadger/pokemon-generator-scripts).

## Similar projects

- [pokemon-colorscripts](https://gitlab.com/phoneybadger/pokemon-colorscripts)
- [pokeget](https://github.com/talwat/pokeget)
- [pokeshell](https://github.com/acxz/pokeshell)
- [krabby](https://github.com/yannjor/krabby)
