package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	shinyRate         = 1.0 / 128.0
	programDir        = "./" // Update this to the appropriate directory if necessary
	colorscriptsDir   = "./assets/colorscripts"
	regularSubdir     = "regular"
	shinySubdir       = "shiny"
	largeSubdir       = "large"
	smallSubdir       = "small"
)

var generations = map[string][2]int{
	"1":  {1, 151},
	"2":  {152, 251},
	"3":  {252, 386},
	"4":  {387, 493},
	"5":  {494, 649},
	"6":  {650, 721},
	"7":  {722, 809},
	"8":  {810, 898},
}

func printFile(filepath string) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	fmt.Print(string(content))
}

func listPokemonNames() {
	file, err := os.ReadFile(filepath.Join(programDir, "./assets/pokemon.json"))
	if err != nil {
		fmt.Println("Error reading pokemon.json:", err)
		return
	}

	var pokemon []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(file, &pokemon); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	for _, p := range pokemon {
		fmt.Println(p.Name)
	}
}

func showPokemonByName(name string, showTitle, shiny, isLarge bool, form string) {
	colorSubdir := regularSubdir
	if shiny {
		colorSubdir = shinySubdir
	}

	sizeSubdir := smallSubdir
	if isLarge {
		sizeSubdir = largeSubdir
	}

	file, err := os.ReadFile(filepath.Join(programDir, "./assets/pokemon.json"))
	if err != nil {
		fmt.Println("Error reading pokemon.json:", err)
		return
	}

	var pokemon []struct {
		Name   string   `json:"name"`
		Forms  []string `json:"forms"`
	}
	if err := json.Unmarshal(file, &pokemon); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	pokemonNames := make(map[string]struct{})
	for _, p := range pokemon {
		pokemonNames[p.Name] = struct{}{}
	}

	if _, exists := pokemonNames[name]; !exists {
		fmt.Printf("Invalid pokemon %s\n", name)
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
			fmt.Printf("Invalid form '%s' for pokemon %s\n", form, name)
			fmt.Println("Available alternate forms are:")
			for _, f := range alternateForms {
				fmt.Printf("- %s\n", f)
			}
			os.Exit(1)
		}
		name += "-" + form
	}

	pokemonFile := filepath.Join(colorscriptsDir, sizeSubdir, colorSubdir, name)
	if showTitle {
		if shiny {
			fmt.Printf("%s (shiny)\n", name)
		} else {
			fmt.Println(name)
		}
	}
	printFile(pokemonFile)
}

func showRandomPokemon(generationsStr string, showTitle, shiny, isLarge bool) {
	rand.Seed(time.Now().UnixNano())

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

	file, err := os.ReadFile(filepath.Join(programDir, "./assets/pokemon.json"))
	if err != nil {
		fmt.Println("Error reading pokemon.json:", err)
		return
	}

	var pokemon []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(file, &pokemon); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	startIdx, ok := generations[startGen]
	if !ok {
		fmt.Printf("Invalid generation '%s'\n", generationsStr)
		os.Exit(1)
	}

	endIdx, ok := generations[endGen]
	if !ok {
		fmt.Printf("Invalid generation '%s'\n", generationsStr)
		os.Exit(1)
	}

	randomIdx := rand.Intn(endIdx[1]-startIdx[0]+1) + startIdx[0]
	randomPokemon := pokemon[randomIdx-1].Name

	if !shiny && rand.Float64() <= shinyRate {
		shiny = true
	}
	showPokemonByName(randomPokemon, showTitle, shiny, isLarge, "")
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func main() {
	listPtr := flag.Bool("list", false, "Print list of all pokemon")
	namePtr := flag.String("name", "", "Select pokemon by name")
	formPtr := flag.String("form", "", "Show an alternate form of a pokemon")
	noTitlePtr := flag.Bool("no-title", false, "Do not display pokemon name")
	shinyPtr := flag.Bool("shiny", false, "Show the shiny version of the pokemon instead")
	bigPtr := flag.Bool("big", false, "Show a larger version of the sprite")
	randomPtr := flag.String("random", "1-8", "Show a random pokemon. This flag can optionally be followed by a generation number or range")

	flag.Parse()

	if *listPtr {
		listPokemonNames()
	} else if *namePtr != "" {
		showPokemonByName(*namePtr, !*noTitlePtr, *shinyPtr, *bigPtr, *formPtr)
	} else if *randomPtr != "" {
		if *formPtr != "" {
			fmt.Println("--form flag unexpected with --random")
			os.Exit(1)
		}
		showRandomPokemon(*randomPtr, !*noTitlePtr, *shinyPtr, *bigPtr)
	} else {
		flag.Usage()
	}
}
