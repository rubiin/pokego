package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"

	"github.com/urfave/cli/v3"
)

type Pokemon struct {
	Name  string   `json:"name"`
	Forms []string `json:"forms"`
}

var (
	// Default so source builds print something useful; release builds
	// override it via -ldflags "-X main.version=...".
	version = "dev"

	// Cached Pokémon data
	allPokemon []Pokemon
	// pokemonIndex maps names to pointers INTO allPokemon. This is a latent
	// hazard: if anything ever appends to (or re-slices) allPokemon, the
	// backing array may be reallocated and every stored pointer dangles.
	// Safe today — buildIndex() runs once, right after loadData(), and the
	// slice is never mutated afterwards — but future code must not append
	// to allPokemon after buildIndex() without rebuilding the index.
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

// loadData reads and parses pokemon.json into allPokemon and derives the final
// generation's ceiling from the data length (so dex 899-905 are reachable via
// --random). embed.FS paths must use forward slashes on every platform, so use
// path.Join (never filepath.Join, whose backslashes break Windows).
func loadData() error {
	data, err := assets.ReadFile(path.Join(rootDir, "pokemon.json"))
	if err != nil {
		return fmt.Errorf("loading pokemon data: %w", err)
	}
	if err := json.Unmarshal(data, &allPokemon); err != nil {
		return fmt.Errorf("parsing pokemon data: %w", err)
	}

	// The canonical ranges in `generations` end before the data does — the
	// Hisui Pokémon (dex 899-905) sit past gen 8's range. Derive the final
	// generation's ceiling from the real data length so every Pokémon is
	// reachable via --random.
	lastGen, lastEnd := "", 0
	for g, r := range generations {
		if r[1] > lastEnd {
			lastGen, lastEnd = g, r[1]
		}
	}
	if r := generations[lastGen]; r[1] < len(allPokemon) {
		r[1] = len(allPokemon)
		generations[lastGen] = r
	}
	return nil
}

// buildIndex builds the name→Pokemon lookup map from allPokemon. Only the
// --name path needs it, so it is built separately from loadData.
func buildIndex() {
	pokemonIndex = make(map[string]*Pokemon, len(allPokemon))
	for i := range allPokemon {
		pokemonIndex[allPokemon[i].Name] = &allPokemon[i]
	}
}

func printFile(path string) error {
	content, err := assets.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}
	// Write the raw bytes: fmt.Print(string(content)) would copy the whole
	// sprite into a new string first.
	if _, err := os.Stdout.Write(content); err != nil {
		return err
	}
	return nil
}

func listPokemonNames() {
	// Buffer the output: fmt.Println to the unbuffered os.Stdout would issue
	// one write(2) syscall per line (905 total); a single flush drops that to
	// a handful of writes.
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for _, p := range allPokemon {
		fmt.Fprintln(w, p.Name)
	}
}

// printPokemon writes the sprite (and optional title) for a known-valid
// Pokemon whose name already includes any form suffix. --random uses this
// directly (the name comes from allPokemon), so it never needs the lookup map.
func printPokemon(name string, showTitle, shiny bool) error {
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

	return printFile(path.Join(rootDir, colorscriptsDir, colorSubdir, name))
}

func showPokemonByName(name string, showTitle, shiny bool, form string) error {
	name = strings.ToLower(name)
	form = strings.ToLower(strings.TrimSpace(form))
	p, ok := pokemonIndex[name]
	if !ok {
		return fmt.Errorf("invalid pokemon %s", name)
	}

	// "regular" is the pokesprite default form listed for every Pokémon, but
	// no "name-regular" sprite files exist — treat it as "no form".
	if form != "" && form != "regular" {
		valid := false
		for _, f := range p.Forms {
			if f == form {
				valid = true
				break
			}
		}
		if !valid {
			msg := fmt.Sprintf("invalid form '%s' for pokemon %s\n", form, name)
			var alternates []string
			for _, f := range p.Forms {
				if f != "regular" {
					alternates = append(alternates, f)
				}
			}
			if len(alternates) > 0 {
				msg += "available alternate forms are:\n"
				for _, f := range alternates {
					msg += fmt.Sprintf("- %s\n", f)
				}
			}
			return errors.New(strings.TrimSuffix(msg, "\n"))
		}
		name += "-" + form
	}

	return printPokemon(name, showTitle, shiny)
}

func showRandomPokemon(genStr string, showTitle, shiny bool) error {
	// Split on commas, dropping empty entries so inputs like "1," or ",,1,3"
	// don't randomly select an empty generation.
	var gens []string
	for _, g := range strings.Split(genStr, ",") {
		if g = strings.TrimSpace(g); g != "" {
			gens = append(gens, g)
		}
	}
	if len(gens) == 0 {
		return fmt.Errorf("invalid generation '%s'", genStr)
	}

	var startGen, endGen string
	if len(gens) > 1 {
		// Comma-separated list: every entry must be a plain generation, and
		// ranges (like "1,2-3") can't be mixed in.
		for _, g := range gens {
			if strings.Contains(g, "-") {
				return fmt.Errorf("cannot mix generation ranges with lists: '%s'", genStr)
			}
			if _, ok := generations[g]; !ok {
				return fmt.Errorf("invalid generation '%s'", g)
			}
		}
		startGen = gens[rand.Intn(len(gens))]
		endGen = startGen
	} else if strings.Contains(genStr, "-") {
		// Range like "1-8": sample across both generations.
		parts := strings.SplitN(genStr, "-", 2)
		startGen, endGen = parts[0], parts[1]
	} else {
		startGen, endGen = gens[0], gens[0]
	}

	startIdx, ok := generations[startGen]
	if !ok {
		return fmt.Errorf("invalid generation '%s'", startGen)
	}
	endIdx, ok := generations[endGen]
	if !ok {
		return fmt.Errorf("invalid generation '%s'", endGen)
	}

	// Never sample past the actual data, whatever the generation map says.
	end := endIdx[1]
	if end > len(allPokemon) {
		end = len(allPokemon)
	}

	// Guard against reversed ranges like "3-1": rand.Intn would get a
	// negative argument and panic with "invalid argument to Intn".
	if startIdx[0] > end {
		return fmt.Errorf("invalid generation range '%s'", genStr)
	}

	randomIdx := rand.Intn(end-startIdx[0]+1) + startIdx[0]
	randomPokemon := allPokemon[randomIdx-1].Name

	if !shiny && rand.Float64() <= shinyRate {
		shiny = true
	}
	// The name comes straight from allPokemon, so it is always valid and the
	// lookup map (built only for --name) is not needed here.
	return printPokemon(randomPokemon, showTitle, shiny)
}

func newApp() *cli.Command {
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
			// Pokemon data is loaded lazily: --version and --help never touch
			// the JSON, and --list/--name/--random load it only when needed.
			switch {
			case cmd.Bool("list"):
				if err := loadData(); err != nil {
					return err
				}
				listPokemonNames()
			case cmd.Bool("version"):
				fmt.Println(version)
			case cmd.String("name") != "":
				if err := loadData(); err != nil {
					return err
				}
				buildIndex()
				return showPokemonByName(cmd.String("name"), !cmd.Bool("no-title"), cmd.Bool("shiny"), cmd.String("form"))
			case cmd.String("random") != "":
				if cmd.String("form") != "" {
					return errors.New("--form flag unexpected with --random")
				}
				if err := loadData(); err != nil {
					return err
				}
				return showRandomPokemon(cmd.String("random"), !cmd.Bool("no-title"), cmd.Bool("shiny"))
			default:
				cli.ShowRootCommandHelp(cmd)
				return errors.New("no command or flags specified")
			}
			return nil
		},
	}

	// cli prints flag-usage errors itself ("Incorrect Usage: ..."); this
	// prints Action errors exactly once, to stderr, before they are returned
	// to main, which only converts them into an exit code.
	app.ExitErrHandler = func(ctx context.Context, cmd *cli.Command, err error) {
		fmt.Fprintln(cmd.ErrWriter, err)
	}
	return app
}

func main() {
	if err := newApp().Run(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
}
