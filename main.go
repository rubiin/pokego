package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

type Pokemon struct {
	Name  string   `json:"name"`
	Forms []string `json:"forms"`
}

var (
	// Overridden by release builds via -ldflags "-X main.version=...".
	version = "dev"

	// Cached Pokémon data.
	allPokemon []Pokemon
	// Name→entry pointers into allPokemon. Don't append to allPokemon after
	// buildIndex() — the pointers would dangle.
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

// loadData parses pokemon.json and extends the last generation to the data
// length so dex 899-905 are reachable via --random. Use path.Join: embed
// paths always use forward slashes (filepath.Join would break Windows).
func loadData() error {
	data, err := assets.ReadFile(path.Join(rootDir, "pokemon.json"))
	if err != nil {
		return fmt.Errorf("loading pokemon data: %w", err)
	}
	if err := json.Unmarshal(data, &allPokemon); err != nil {
		return fmt.Errorf("parsing pokemon data: %w", err)
	}

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

// buildIndex precomputes the name→Pokemon map used by --name.
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
	// Write raw bytes: fmt.Print(string(content)) would copy the sprite.
	if _, err := os.Stdout.Write(content); err != nil {
		return err
	}
	return nil
}

// listPokemonNames prints one name per line, buffered to avoid a syscall per
// name (905 total).
func listPokemonNames() {
	w := bufio.NewWriter(os.Stdout)
	defer func() { _ = w.Flush() }()
	for _, p := range allPokemon {
		_, _ = fmt.Fprintln(w, p.Name)
	}
}

// printPokemon writes the sprite for a name that is already known valid, with
// an optional title and shiny color. --random passes names straight from
// allPokemon, so it never needs the lookup map.
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

	// "regular" is the pokesprite default: no "name-regular" sprite files
	// exist, so treat it as no form.
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
			// Show "regular" too when it's the only alternate, so users see
			// the full set.
			if len(alternates) > 0 && len(p.Forms) > len(alternates) {
				alternates = append([]string{"regular"}, alternates...)
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
	// Drop empty entries so "1," or ",,1,3" don't select an empty generation.
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
		// List entries must be plain generations; ranges can't be mixed in.
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

	// Guard reversed ranges like "3-1": rand.Intn would panic on a negative.
	if startIdx[0] > end {
		return fmt.Errorf("invalid generation range '%s'", genStr)
	}

	randomIdx := rand.Intn(end-startIdx[0]+1) + startIdx[0]
	randomPokemon := allPokemon[randomIdx-1].Name

	if !shiny && rand.Float64() <= shinyRate {
		shiny = true
	}
	// The name comes from allPokemon, so it is always valid; no lookup needed.
	return printPokemon(randomPokemon, showTitle, shiny)
}

// --- CLI ---

func newApp() *cobra.Command {
	var (
		list, showVersion, noTitle, shiny bool
		name, form, random                string
	)

	app := &cobra.Command{
		Use:   "pokego",
		Short: "display Pokémon sprites in color directly in your terminal",
		// Accept positional args (and ignore them), like the old flag CLI;
		// otherwise cobra would report them as unknown commands.
		Args: cobra.ArbitraryArgs,
		// Suppress the usage dump cobra appends to errors; help is printed
		// only in the default branch below when no mode flag was given.
		SilenceUsage: true,
		// The built-in completion command prints help instead of failing on
		// unknown shells, so a validating one is added below.
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Payload loads lazily: --version and --help never touch the JSON.
			switch {
			case list:
				if err := loadData(); err != nil {
					return err
				}
				listPokemonNames()
				return nil
			case showVersion:
				fmt.Println(version)
				return nil
			case name != "":
				if err := loadData(); err != nil {
					return err
				}
				buildIndex()
				return showPokemonByName(name, !noTitle, shiny, form)
			case random != "":
				if form != "" {
					return errors.New("--form flag unexpected with --random")
				}
				if err := loadData(); err != nil {
					return err
				}
				return showRandomPokemon(random, !noTitle, shiny)
			default:
				// Help to stdout like the old CLI; the error goes to stderr.
				_ = cmd.Help()
				return errors.New("no command or flags specified")
			}
		},
	}

	f := app.Flags()
	f.BoolVarP(&list, "list", "l", false, "List all Pokémon")
	f.StringVarP(&name, "name", "n", "", "Select Pokémon by name")
	f.StringVarP(&form, "form", "f", "", "Show alternate form of a Pokémon")
	// pflag shorthands are single ASCII characters, so the old urfave `-nt`
	// alias for --no-title can't be expressed.
	f.BoolVarP(&noTitle, "no-title", "", false, "Do not display Pokémon name")
	f.BoolVarP(&shiny, "shiny", "s", false, "Show shiny version")
	f.StringVarP(&random, "random", "r", "", "Show random Pokémon, optionally by generation or range")
	f.BoolVarP(&showVersion, "version", "v", false, "Show CLI version")

	app.AddCommand(newCompletionCmd())

	return app
}

// newCompletionCmd generates shell completion scripts and only accepts the
// shells it knows; unknown ones fail instead of printing help.
func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate the autocompletion script for the specified shell",
		Args: cobra.MatchAll(cobra.ExactArgs(1), func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash", "zsh", "fish", "powershell":
				return nil
			default:
				return fmt.Errorf("unsupported shell %q (choose bash, zsh, fish, or powershell)", args[0])
			}
		}),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, out := cmd.Root(), cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			default: // powershell
				return root.GenPowerShellCompletionWithDesc(out)
			}
		},
	}
}

func main() {
	if err := newApp().Execute(); err != nil {
		os.Exit(1)
	}
}
