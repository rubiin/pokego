package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

// Pokemon struct represents the data structure for a Pokémon
type Pokemon struct {
	Name  string   `json:"name"`
	Forms []string `json:"forms"`
}

var (
	version string
)

// Embed assets directory
//
//go:embed assets/*
var assets embed.FS

const (
	rootDir         = "assets"
	shinyRate       = 1.0 / 128.0
	colorscriptsDir = "colorscripts"
	regularSubdir   = "regular"
	shinySubdir     = "shiny"
)

// Generation ranges for Pokémon
var generations = map[string][2]int{
	"1": {1, 151},
	"2": {152, 251},
	"3": {252, 386},
	"4": {387, 493},
	"5": {494, 649},
	"6": {650, 721},
	"7": {722, 809},
	"8": {810, 898},
}

// printFile prints the content of the specified file
func printFile(filepath string) {
	content, err := assets.ReadFile(filepath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	fmt.Print(string(content))
}

// readPokemonJSON reads the pokemon.json file from the embedded assets
func readPokemonJSON() []Pokemon {
	file, err := assets.ReadFile(filepath.Join(rootDir, "pokemon.json"))
	if err != nil {
		panic(err)
	}

	var pokemon []Pokemon
	if err := json.Unmarshal(file, &pokemon); err != nil {
		panic(err)
	}
	return pokemon
}

// listPokemonNames lists the names of all Pokémon
func listPokemonNames() {
	pokemon := readPokemonJSON()
	for _, p := range pokemon {
		fmt.Println(p.Name)
	}
}

// showPokemonByName displays Pokémon information based on its name
func showPokemonByName(name string, showTitle, shiny bool, form string) {
	colorSubdir := regularSubdir
	if shiny {
		colorSubdir = shinySubdir
	}

	pokemon := readPokemonJSON()
	pokemonNames := make(map[string]struct{})

	for _, p := range pokemon {
		pokemonNames[p.Name] = struct{}{}
	}

	if _, exists := pokemonNames[name]; !exists {
		fmt.Printf("invalid pokemon %s\n", name)
		os.Exit(1)
	}

	if form != "" {
		var alternateForms []string
		for _, p := range pokemon {
			if p.Name == name {
				alternateForms = p.Forms
				break
			}
		}
		if !contains(alternateForms, form) {
			fmt.Printf("invalid form '%s' for pokemon %s\n", form, name)
			fmt.Println("available alternate forms are:")
			for _, f := range alternateForms {
				fmt.Printf("- %s\n", f)
			}
			os.Exit(1)
		}
		name += "-" + form
	}

	pokemonFile := filepath.Join(rootDir, colorscriptsDir, colorSubdir, name)
	if showTitle {
		if shiny {
			fmt.Printf("%s (shiny)\n", name)
		} else {
			fmt.Println(name)
		}
	}
	printFile(pokemonFile)
}

// showRandomPokemon displays a random Pokémon based on specified generations
func showRandomPokemon(generationsStr string, showTitle, shiny bool) {
	var startGen, endGen string
	genList := strings.Split(generationsStr, ",")

	if len(genList) > 1 {
		startGen = genList[rand.Intn(len(genList))]
		endGen = startGen
	} else if strings.Contains(generationsStr, "-") {
		parts := strings.Split(generationsStr, "-")
		startGen, endGen = parts[0], parts[1]
	} else {
		startGen = generationsStr
		endGen = startGen
	}

	pokemon := readPokemonJSON()
	startIdx, ok := generations[startGen]
	if !ok {
		fmt.Printf("invalid generation '%s'\n", generationsStr)
		os.Exit(1)
	}

	endIdx, ok := generations[endGen]
	if !ok {
		fmt.Printf("invalid generation '%s'\n", generationsStr)
		os.Exit(1)
	}

	randomIdx := rand.Intn(endIdx[1]-startIdx[0]+1) + startIdx[0]
	randomPokemon := pokemon[randomIdx-1].Name

	if !shiny && rand.Float64() <= shinyRate {
		shiny = true
	}
	showPokemonByName(randomPokemon, showTitle, shiny, "")
}

// contains checks if a slice contains a specific item
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// main function to handle command-line flags and execute appropriate actions
func main() {
	app := &cli.App{
		Name:  "pokego",
		Usage: "command-line tool that lets you display Pokémon sprites in color directly in your terminal",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "Print list of all pokemon",
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "Select pokemon by name",
			},
			&cli.StringFlag{
				Name:    "form",
				Aliases: []string{"f"},
				Usage:   "Show an alternate form of a pokemon",
			},
			&cli.BoolFlag{
				Name:  "no-title",
				Usage: "Do not display pokemon name",
			},
			&cli.BoolFlag{
				Name:    "shiny",
				Aliases: []string{"s"},
				Usage:   "Show the shiny version of the pokemon instead",
			},
			&cli.StringFlag{
				Name:    "random",
				Aliases: []string{"r"},
				Usage:   "Show a random pokemon. This flag can optionally be followed by a generation number or range",
			},
			&cli.BoolFlag{
				Name:  "version",
				Usage: "Show the cli version",
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Bool("list") {
				listPokemonNames()
			} else if ctx.Bool("version") {
				fmt.Println(version)
			} else if ctx.String("name") != "" {
				showPokemonByName(ctx.String("name"), !ctx.Bool("no-title"), ctx.Bool("shiny"), ctx.String("form"))
			} else if ctx.String("random") != "" {
				if ctx.String("form") != "" {
					fmt.Println("--form flag unexpected with --random")
					os.Exit(1)
				}
				showRandomPokemon(ctx.String("random"), !ctx.Bool("no-title"), ctx.Bool("shiny"))
			} else {
				cli.ShowAppHelpAndExit(ctx, 1)
			}
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}
}
