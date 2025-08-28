package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
)

type Pokemon struct {
	Name  string   `json:"name"`
	Forms []string `json:"forms"`
}

var (
	version string

	// Cached Pokémon data
	allPokemon   []Pokemon
	pokemonIndex map[string]*Pokemon
)

//go:embed assets/*
var assets embed.FS

const (
	rootDir         = "assets"
	shinyRate       = 1.0 / 128.0
	colorscriptsDir = "colorscripts"
	regularSubdir   = "regular"
	shinySubdir     = "shiny"
)

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

// --- Helpers ---

func mustLoadPokemon() {
	data, err := assets.ReadFile(filepath.Join(rootDir, "pokemon.json"))
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, &allPokemon); err != nil {
		panic(err)
	}
	pokemonIndex = make(map[string]*Pokemon, len(allPokemon))
	for i := range allPokemon {
		pokemonIndex[allPokemon[i].Name] = &allPokemon[i]
	}
}

func printFile(path string) {
	content, err := assets.ReadFile(path)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	fmt.Print(string(content))
}

func listPokemonNames() {
	for _, p := range allPokemon {
		fmt.Println(p.Name)
	}
}

func showPokemonByName(name string, showTitle, shiny bool, form string) {
	p, ok := pokemonIndex[name]
	if !ok {
		fmt.Printf("invalid pokemon %s\n", name)
		os.Exit(1)
	}

	if form != "" {
		valid := false
		for _, f := range p.Forms {
			if f == form {
				valid = true
				break
			}
		}
		if !valid {
			fmt.Printf("invalid form '%s' for pokemon %s\n", form, name)
			fmt.Println("available alternate forms are:")
			for _, f := range p.Forms {
				fmt.Printf("- %s\n", f)
			}
			os.Exit(1)
		}
		name += "-" + form
	}

	colorSubdir := regularSubdir
	if shiny {
		colorSubdir = shinySubdir
	}

	if showTitle {
		if shiny {
			fmt.Printf("%s (shiny)\n", name)
		} else {
			fmt.Println(name)
		}
	}

	printFile(filepath.Join(rootDir, colorscriptsDir, colorSubdir, name))
}

func showRandomPokemon(genStr string, showTitle, shiny bool) {
	var startGen, endGen string
	genList := strings.Split(genStr, ",")

	if len(genList) > 1 {
		startGen = genList[rand.Intn(len(genList))]
		endGen = startGen
	} else if strings.Contains(genStr, "-") {
		parts := strings.SplitN(genStr, "-", 2)
		startGen, endGen = parts[0], parts[1]
	} else {
		startGen, endGen = genStr, genStr
	}

	startIdx, ok := generations[startGen]
	if !ok {
		fmt.Printf("invalid generation '%s'\n", genStr)
		os.Exit(1)
	}
	endIdx, ok := generations[endGen]
	if !ok {
		fmt.Printf("invalid generation '%s'\n", genStr)
		os.Exit(1)
	}

	randomIdx := rand.Intn(endIdx[1]-startIdx[0]+1) + startIdx[0]
	randomPokemon := allPokemon[randomIdx-1].Name

	if !shiny && rand.Float64() <= shinyRate {
		shiny = true
	}
	showPokemonByName(randomPokemon, showTitle, shiny, "")
}

func main() {
	mustLoadPokemon()

	app := &cli.Command{
		Name:  "pokego",
		Usage: "display Pokémon sprites in color directly in your terminal",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "list", Aliases: []string{"l"}, Usage: "List all Pokémon"},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Select Pokémon by name"},
			&cli.StringFlag{Name: "form", Aliases: []string{"f"}, Usage: "Show alternate form of a Pokémon"},
			&cli.BoolFlag{Name: "no-title", Aliases: []string{"nt"}, Usage: "Do not display Pokémon name"},
			&cli.BoolFlag{Name: "shiny", Aliases: []string{"s"}, Usage: "Show shiny version"},
			&cli.StringFlag{Name: "random", Aliases: []string{"r"}, Usage: "Show random Pokémon, optionally by generation or range"},
			&cli.BoolFlag{Name: "version", Aliases: []string{"v"}, Usage: "Show CLI version"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			switch {
			case cmd.Bool("list"):
				listPokemonNames()
			case cmd.Bool("version"):
				fmt.Println(version)
			case cmd.String("name") != "":
				showPokemonByName(cmd.String("name"), !cmd.Bool("no-title"), cmd.Bool("shiny"), cmd.String("form"))
			case cmd.String("random") != "":
				if cmd.String("form") != "" {
					fmt.Println("--form flag unexpected with --random")
					os.Exit(1)
				}
				showRandomPokemon(cmd.String("random"), !cmd.Bool("no-title"), cmd.Bool("shiny"))
			default:
				cli.ShowRootCommandHelp(cmd)
				os.Exit(1)
			}
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
	}
}
