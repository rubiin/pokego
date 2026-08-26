# pokego

![AUR version](https://img.shields.io/aur/version/pokego-bin)
![Release](https://img.shields.io/github/v/release/rubiin/pokego)

A fast, Go rewrite of [phoney badger's pokemon-colorscripts](https://gitlab.com/phoneybadger/pokemon-colorscripts) that prints Pokémon sprites in color directly in your terminal.

<img src="logo.png" alt="pokego logo" width="300">

## Table of contents

* [Features](#features)
* [Installation](#installation)
  * [Arch Linux](#arch-linux)
  * [NixOS / Nix](#nixos--nix)
  * [Other platforms](#other-platforms)
  * [From source](#from-source)
* [Usage](#usage)
  * [Examples](#examples)
  * [Shell completions](#shell-completions)
  * [Shell integration](#shell-integration)
* [Comparison](#comparison)
* [Credits](#credits)
* [Similar projects](#similar-projects)

## Features

- Pokémon from generations 1–8 (905 total), including shiny, mega, Gigantamax, and regional variants
- Print random Pokémon, optionally filtered by generation or range
- Random Pokémon have a 1/128 chance of being shiny, just like in the games
- Print a specific Pokémon by name, with an optional alternate form

## Installation

### Arch Linux

Install the stable package from the AUR:

```sh
yay -S pokego-bin
```

Or build manually from the AUR by downloading the PKGBUILD, then running:

```sh
makepkg -si
```

If you want to track the latest main branch, there is also the development package [`pokego-git`](https://aur.archlinux.org/packages/pokego-git).

### NixOS / Nix

Pokego is available in nixpkgs.

System-wide, add it to `environment.systemPackages`:

```nix
environment.systemPackages = with pkgs; [
  pokego
];
```

Or per-user with [home-manager](https://github.com/nix-community/home-manager), add it to `home.packages`:

```nix
home.packages = with pkgs; [
  pokego
];
```

### Other platforms

Prebuilt binaries for **Linux**, **macOS**, and **Windows** are available on the [releases page](https://github.com/rubiin/pokego/releases).

Download the archive for your platform, extract it, and move the executable to a directory on your `PATH`:

```sh
mv pokego ~/.local/bin
```

### From source

Clone the repository and build with [just](https://github.com/casey/just):

```sh
git clone https://github.com/rubiin/pokego.git
cd pokego
just build
```

Or without `just`:

```sh
go build -ldflags "-s -w" -o pokego .
```

Then move the executable to a directory on your `PATH`.

> **Note:** The binary is ~26 MB because every sprite is embedded in it — no external assets are needed at runtime.

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

`--list` prints the base name of every Pokémon; alternate forms are not listed.
Use forms with `--name`, e.g. `pokego --name charizard --form mega-y`.
Valid generations for `--random` are `1` through `8`; ranges (`1-3`) and comma-separated lists (`1,3,6`) are both supported.

### Examples

Print a specific Pokémon:

```sh
pokego --name charizard
```

Print a shiny Pokémon:

```sh
pokego --name spheal --shiny
```

Print an alternate form of a Pokémon:

```sh
pokego --name blastoise --form mega
```

Hide the title:

```sh
pokego --name mudkip --no-title
```

Print a random Pokémon from generations 1-3:

```sh
pokego --random 1-3
```

Print a random Pokémon from generations 1, 3, and 6:

```sh
pokego --random 1,3,6
```

### Shell completions

Pokego can generate shell completion scripts for bash, zsh, fish, and PowerShell:

```sh
# .bashrc
source <(pokego completion bash)

# .zshrc
source <(pokego completion zsh)

# fish
pokego completion fish > ~/.config/fish/completions/pokego.fish
```

For PowerShell, save the script and run it:

```sh
pokego completion pwsh > ~\Documents\WindowsPowerShell\Scripts\pokego.ps1
```

### Shell integration

Add a Pokémon to your terminal greeting by appending one of these to your `.zshrc` or `.bashrc`:

```sh
# Random Pokémon on every new shell
pokego --random --no-title
```

For fish, add it to your `fish_greeting` function:

```fish
function fish_greeting
    pokego --random --no-title
end
```

> **Note:** Your terminal must support Unicode and ANSI colors (virtually all modern terminals do).

## Comparison

Start time is the mean of 5 consecutive runs using the `time` coreutil on an Acer Aspire 5, measured on 2024-10-06.

| Tool               | Start time (s) | Size (MB) | Language |
|--------------------|----------------|-----------|----------|
| **Pokego**         | 0.005          | 27        | Go       |
| **Pokeget**        | 0.006          | 5         | Rust     |
| **Krabby**         | 0.016          | 23        | Rust     |
| **Pokemonscripts** | 0.060          | 43        | Python   |

The ~27 MB Pokego binary is dominated by the embedded sprite assets (the raw corpus is ~29 MB); `go build -ldflags "-s -w"` trims it to ~26 MB.

## Credits

Pokego's Pokémon sprites were sourced from [PokéSprite](https://msikma.github.io/pokesprite/) and transformed into Unicode format using Phoney Badger's [pokemon-generator-scripts](https://gitlab.com/phoneybadger/pokemon-generator-scripts).

## Similar projects

- [pokemon-colorscripts](https://gitlab.com/phoneybadger/pokemon-colorscripts)
- [pokeget](https://github.com/talwat/pokeget)
- [pokeshell](https://github.com/acxz/pokeshell)
- [krabby](https://github.com/yannjor/krabby)
