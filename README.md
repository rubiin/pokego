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
- Ability to print random Pokémon with options to filter by generation and form
- Print specific Pokémon by name
- Display both the sprite and Pokédex entry

## Installation

From the AUR using your favorite AUR helper

```sh
yay -S pokego-git
```

Or alternatively you can manually download the PKGBUILD file from the repository, then run

```sh
makepkg -si
```

## Usage
Run the help command `pokego --help` to see the following help message.

```
USAGE:
    pokego <SUBCOMMAND>

OPTIONS:
    -h, --help       Print help information

SUBCOMMANDS:
    help      Print this message or the help of the given subcommand(s)
    list      Print list of all pokemon
    name      Select pokemon by name. Generally spelled like in the games. A few exceptions are
                  nidoran-f, nidoran-m, mr-mime, farfetchd, flabebe type-null etc. Perhaps grep the
                  output of list if in doubt
    random    Show a random pokemon. This command can optionally be followed by a generation
                  number or range (1-8) to show random pokemon from a specific generation or range
                  of generations. The generations can be provided as a continuous range (eg. 1-3) or
                  as a list of generations (1,3,6)
```

To get the help of the random subcommand.

### Examples
Print a specific pokemon
```
pokego --name charizard
```
Print a specific shiny pokemon
```
pokego --name spheal -s
```
Print a specific pokemon together with its pokedex entry
```
pokego --name mudkip
```
Print an alternative form of a pokemon
```
pokego --name blastoise --form mega
```
Print a random pokemon (gens 1-8)
```
pokego
```
Print random pokemon from generations 1-3
```
pokego --random 1-3
```
Print a random pokemon from generations 1,3 and 6
```
pokego --random 1,3,6
```

## Credits
Pokego's Pokémon sprites were sourced from [PokéSprite](https://msikma.github.io/pokesprite/) and transformed into Unicode format using Phoney Badger's [pokemon-generator-scripts](https://gitlab.com/phoneybadger/pokemon-generator-scripts).


## Similar projects
- [pokemon-colorscripts](https://gitlab.com/phoneybadger/pokemon-colorscripts)
- [pokeget](https://github.com/talwat/pokeget)
- [pokeshell](https://github.com/acxz/pokeshell)
- [krabby](https://github.com/yannjor/krabby)
