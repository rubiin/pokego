package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// Load the embedded data once for all tests. loadData is idempotent.
func TestMain(m *testing.M) {
	if err := loadData(); err != nil {
		panic(err)
	}
	buildIndex()
	os.Exit(m.Run())
}

// silent and capture swap the process-global os.Stdout; they must not be used
// with t.Parallel.

// newTestApp returns a fresh app with error output discarded, so the
// ExitErrHandler doesn't spam test logs on expected error paths.
func newTestApp() *cli.Command {
	app := newApp()
	app.ErrWriter = io.Discard
	return app
}

// silent runs fn with stdout redirected to the null device so sprite output
// stays out of test logs.
func silent(fn func()) {
	old := os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		panic(err)
	}
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()
	fn()
}

// These tests are intentionally coupled to the embedded corpus (905 entries,
// known first/last names, pikachu's alola-cap form, mudkip having no real
// alternates) — the corpus is the fixture.
func TestDataIntegrity(t *testing.T) {
	if len(allPokemon) != 905 {
		t.Errorf("allPokemon has %d entries, want 905", len(allPokemon))
	}
	if allPokemon[0].Name != "bulbasaur" {
		t.Errorf("first pokemon = %q, want bulbasaur", allPokemon[0].Name)
	}
	if last := allPokemon[len(allPokemon)-1]; last.Name != "enamorus" {
		t.Errorf("last pokemon = %q, want enamorus", last.Name)
	}
	for i, p := range allPokemon {
		if p.Name != strings.ToLower(p.Name) {
			t.Errorf("entry %d has non-lowercase name %q", i, p.Name)
		}
		hasRegular := false
		for _, f := range p.Forms {
			if f == "regular" {
				hasRegular = true
			}
		}
		if !hasRegular {
			t.Errorf("pokemon %s is missing the 'regular' form", p.Name)
		}
	}
	if len(pokemonIndex) != len(allPokemon) {
		t.Errorf("index has %d entries, want %d", len(pokemonIndex), len(allPokemon))
	}
}

// Bug 3: the final generation's ceiling must be derived from the data length
// so dex 899-905 (the Hisui Pokémon) are reachable via --random.
func TestGenerationsCeiling(t *testing.T) {
	if got := generations["8"][1]; got != len(allPokemon) {
		t.Errorf("gen 8 ends at %d, want %d (the data length)", got, len(allPokemon))
	}
}

func TestShowPokemonByName(t *testing.T) {
	// Valid lookups: no error (sprite output silenced).
	silent(func() {
		for _, tc := range []struct {
			name, form string
		}{{"pikachu", ""}, // base name
			{"PIKACHU", ""},            // bug 7: case-insensitive lookup
			{"bulbasaur", "regular"},   // bug 2: "regular" is a no-op, not a filename suffix
			{"pikachu", "alola-cap"},   // real alternate form resolves to a sprite file
			{"pikachu", "ALOLA-CAP"},   // bug 13: form lookup is case-insensitive
			{"pikachu", " alola-cap "}, // bug 13: form value is trimmed
		} {
			if err := showPokemonByName(tc.name, false, false, tc.form); err != nil {
				t.Errorf("showPokemonByName(%q, form=%q): unexpected error %v", tc.name, tc.form, err)
			}
		}
	})

	// Bug 8: errors are returned, not os.Exit.
	if err := showPokemonByName("missingmon", false, false, ""); err == nil || !strings.Contains(err.Error(), "invalid pokemon") {
		t.Errorf("missing pokemon: got %v, want 'invalid pokemon' error", err)
	}
	if err := showPokemonByName("pikachu", false, false, "bogus"); err == nil || !strings.Contains(err.Error(), "invalid form") {
		t.Errorf("bogus form: got %v, want 'invalid form' error", err)
	}
}

func TestInvalidFormListsAlternates(t *testing.T) {
	err := showPokemonByName("pikachu", false, false, "bogus")
	if err == nil {
		t.Fatal("expected error for bogus form")
	}
	msg := err.Error()
	if !strings.Contains(msg, "- alola-cap") {
		t.Errorf("alternates list missing '- alola-cap':\n%s", msg)
	}
	if strings.Contains(msg, "- regular") {
		t.Errorf("alternates list must not include 'regular':\n%s", msg)
	}

	// mudkip has no real alternates, so no list section should be shown.
	err = showPokemonByName("mudkip", false, false, "bogus")
	if err == nil {
		t.Fatal("expected error for bogus form")
	}
	if strings.Contains(err.Error(), "available alternate forms") {
		t.Errorf("mudkip has no alternates but list section shown:\n%s", err.Error())
	}
}

func TestPrintFile(t *testing.T) {
	silent(func() {
		if err := printFile("assets/colorscripts/regular/bulbasaur"); err != nil {
			t.Errorf("existing sprite: unexpected error %v", err)
		}
	})
	// Bug 2: a missing sprite is an error, not a silent exit 0.
	if err := printFile("assets/colorscripts/regular/bulbasaur-regular"); err == nil || !strings.Contains(err.Error(), "error reading file") {
		t.Errorf("missing sprite: got %v, want 'error reading file'", err)
	}
}

// Bugs 1 & 6: --random parsing must be robust, with precise error messages.
func TestShowRandomPokemonErrors(t *testing.T) {
	tests := []struct {
		gen  string
		want string
	}{
		{"99", "invalid generation '99'"},
		{"3-1", "invalid generation range"},
		{"1,3,9", "invalid generation '9'"}, // per-entry message, not the whole string
		{"1,2-3", "cannot mix generation ranges with lists"},
		{",,", "invalid generation"},
	}
	for _, tc := range tests {
		err := showRandomPokemon(tc.gen, false, false)
		if err == nil {
			t.Errorf("--random %s: expected error containing %q, got nil", tc.gen, tc.want)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("--random %s: got %q, want containing %q", tc.gen, err.Error(), tc.want)
		}
	}
}

func TestShowRandomPokemonValid(t *testing.T) {
	gens := []string{
		"1",       // single generation
		"8",       // final generation (covers dex 810-905 after the bug 3 fix)
		"1-8",     // full range
		"8-8",     // same-gen range
		"1,",      // bug 6: trailing comma must not randomly pick an empty entry
		",1,3,6,", // commas around every entry
		"1, 3, 6", // whitespace tolerance
		"1,3,6",   // list
	}
	for _, gen := range gens {
		silent(func() {
			if err := showRandomPokemon(gen, false, false); err != nil {
				t.Errorf("--random %s: unexpected error %v", gen, err)
			}
		})
	}
}

// Action-level behavior (bug 8): errors are returned through the cli Action.
func TestAction(t *testing.T) {
	run := func(args ...string) error {
		return newTestApp().Run(context.Background(), append([]string{"pokego"}, args...))
	}

	silent(func() {
		if err := run("--name", "pikachu", "--no-title"); err != nil {
			t.Errorf("--name pikachu: unexpected error %v", err)
		}
		if err := run("--list"); err != nil {
			t.Errorf("--list: unexpected error %v", err)
		}
		if err := run("--version"); err != nil {
			t.Errorf("--version: unexpected error %v", err)
		}
	})

	if err := run("--name", "missingmon"); err == nil {
		t.Error("--name missingmon: expected error")
	}
	if err := run("--random", "2", "--form", "x"); err == nil {
		t.Error("--random with --form: expected error")
	}
	if err := run("--random", "3-1"); err == nil {
		t.Error("--random 3-1: expected error")
	}
	if err := run("--random", "99"); err == nil {
		t.Error("--random 99: expected error")
	}

	// No args: help is shown and an error is returned.
	silent(func() {
		if err := run(); err == nil {
			t.Error("no args: expected error")
		}
	})
}

// EnableShellCompletion must expose the hidden `completion` command for every
// supported shell, with each script referencing the program name.
func TestShellCompletion(t *testing.T) {
	run := func(args ...string) string {
		var err error
		out := capture(func() {
			err = newTestApp().Run(context.Background(), append([]string{"pokego"}, args...))
		})
		if err != nil {
			t.Errorf("%v: unexpected error %v", args, err)
		}
		return out
	}

	for _, shell := range []string{"bash", "zsh", "fish", "pwsh"} {
		out := run("completion", shell)
		// pwsh completion scripts use $MyInvocation.MyCommand.Name to
		// resolve the program name at runtime, so they don't hardcode it.
		if shell == "pwsh" {
			if !strings.Contains(out, "Register-ArgumentCompleter") {
				t.Errorf("completion pwsh: script missing Register-ArgumentCompleter")
			}
		} else if !strings.Contains(out, "pokego") {
			t.Errorf("completion %s: script missing program name", shell)
		}
	}

	// An unknown shell must fail rather than print a bogus script.
	if err := newTestApp().Run(context.Background(), []string{"pokego", "completion", "tcsh"}); err == nil {
		t.Error("completion tcsh: expected error")
	}
}

// capture runs fn with stdout redirected to a temp file and returns the
// output it wrote. cli v3's Setup assigns cmd.Writer from os.Stdout inside
// Run, so this also captures --help and other cli-rendered output.
func capture(fn func()) string {
	old := os.Stdout
	f, err := os.CreateTemp("", "pokego-out-*")
	if err != nil {
		panic(err)
	}
	os.Stdout = f
	defer func() {
		os.Stdout = old
		f.Close()
		os.Remove(f.Name())
	}()
	fn()
	f.Close()
	b, err := os.ReadFile(f.Name())
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestListOutput(t *testing.T) {
	out := capture(listPokemonNames)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 905 {
		t.Errorf("--list printed %d lines, want 905", len(lines))
	}
	if lines[0] != "bulbasaur" || lines[len(lines)-1] != "enamorus" {
		t.Errorf("first/last lines = %q / %q", lines[0], lines[len(lines)-1])
	}
}

func TestTitleOutput(t *testing.T) {
	tests := []struct {
		name       string
		showTitle  bool
		shiny      bool
		form       string
		wantPrefix string
	}{
		{"pikachu", true, false, "", "pikachu\n"},
		{"pikachu", true, true, "", "pikachu (shiny)\n"},
		{"pikachu", true, false, "alola-cap", "pikachu-alola-cap\n"},
		{"pikachu", false, false, "", ""}, // --no-title: no title line
	}
	for _, tc := range tests {
		out := capture(func() {
			if err := showPokemonByName(tc.name, tc.showTitle, tc.shiny, tc.form); err != nil {
				t.Errorf("showPokemonByName(%s, title=%v, shiny=%v, form=%q): %v", tc.name, tc.showTitle, tc.shiny, tc.form, err)
			}
		})
		if tc.wantPrefix != "" {
			if first := strings.SplitN(out, "\n", 2)[0]; first+"\n" != tc.wantPrefix {
				t.Errorf("%s: title = %q, want %q", tc.name, first, tc.wantPrefix)
			}
		} else if strings.HasPrefix(out, tc.name) {
			t.Errorf("%s: title should be suppressed with --no-title", tc.name)
		}
	}
}

// The printed sprite must be byte-identical to the embedded sprite file, for
// both the regular and shiny subdirectories.
func TestSpriteOutputMatchesFile(t *testing.T) {
	want, err := assets.ReadFile("assets/colorscripts/regular/pikachu")
	if err != nil {
		t.Fatal(err)
	}
	out := capture(func() {
		if err := showPokemonByName("pikachu", false, false, ""); err != nil {
			t.Error(err)
		}
	})
	if out != string(want) {
		t.Errorf("regular sprite output differs from embedded file (%d vs %d bytes)", len(out), len(want))
	}

	wantShiny, err := assets.ReadFile("assets/colorscripts/shiny/pikachu")
	if err != nil {
		t.Fatal(err)
	}
	out = capture(func() {
		if err := showPokemonByName("pikachu", false, true, ""); err != nil {
			t.Error(err)
		}
	})
	if out != string(wantShiny) {
		t.Errorf("shiny sprite output differs from embedded file (%d vs %d bytes)", len(out), len(wantShiny))
	}
}

// Bug 3: --random 8 must only sample dex 810-905 and must be able to reach
// all seven Hisui Pokémon (dex 899-905). Statistical: each name has a
// ~(95/96)^2000 chance of never appearing, so a miss is effectively impossible.
func TestRandomGen8Coverage(t *testing.T) {
	hisui := map[string]bool{"wyrdeer": true, "kleavor": true, "ursaluna": true, "basculegion": true, "sneasler": true, "overqwil": true, "enamorus": true}
	expected := make(map[string]bool, 96)
	for i := 809; i < len(allPokemon); i++ {
		expected[allPokemon[i].Name] = true
	}
	seen := make(map[string]bool)
	for i := 0; i < 2000; i++ {
		out := capture(func() {
			if err := showRandomPokemon("8", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 8 sampled %q outside dex 810-905", title)
		}
		if hisui[title] {
			seen[title] = true
		}
	}
	for n := range hisui {
		if !seen[n] {
			t.Errorf("%s (dex 899-905) was never sampled in 1200 draws", n)
		}
	}
}

// List-mode --random must only sample within the selected generations.
func TestRandomListTitlesInRange(t *testing.T) {
	expected := make(map[string]bool)
	for _, g := range []string{"1", "3", "6"} {
		r := generations[g]
		for i := r[0]; i <= r[1]; i++ {
			expected[allPokemon[i-1].Name] = true
		}
	}
	for i := 0; i < 60; i++ {
		out := capture(func() {
			if err := showRandomPokemon("1,3,6", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 1,3,6 sampled %q outside gens 1/3/6", title)
		}
	}
}

func TestActionOutput(t *testing.T) {
	run := func(args ...string) (string, error) {
		var err error
		out := capture(func() {
			err = newTestApp().Run(context.Background(), append([]string{"pokego"}, args...))
		})
		return out, err
	}

	out, err := run("--version")
	if err != nil {
		t.Errorf("--version: %v", err)
	}
	if out != "dev\n" {
		t.Errorf("--version output = %q, want %q", out, "dev\n")
	}

	out, err = run("--list")
	if err != nil {
		t.Errorf("--list: %v", err)
	}
	if got := strings.Count(out, "\n"); got != 905 {
		t.Errorf("--list printed %d lines, want 905", got)
	}

	out, err = run("--help")
	if err != nil {
		t.Errorf("--help: %v", err)
	}
	for _, flag := range []string{"--list", "--name", "--form", "--no-title", "--shiny", "--random", "--version"} {
		if !strings.Contains(out, flag) {
			t.Errorf("--help output missing %q", flag)
		}
	}

	// No args: help is shown and an error is returned.
	out, err = run()
	if err == nil {
		t.Error("no args: expected error")
	}
	if !strings.Contains(out, "GLOBAL OPTIONS") {
		t.Error("no args: help output missing GLOBAL OPTIONS")
	}
}

func TestActionAliases(t *testing.T) {
	run := func(args ...string) error {
		return newTestApp().Run(context.Background(), append([]string{"pokego"}, args...))
	}
	silent(func() {
		for _, args := range [][]string{
			{"-n", "pikachu", "-nt"},
			{"-l"},
			{"-r", "3"},
			{"-v"},
			{"-n", "pikachu", "-f", "alola-cap"},
			{"-n", "pikachu", "-s", "-nt"},
			{"-r", "2", "-s"},
		} {
			if err := run(args...); err != nil {
				t.Errorf("aliases %v: unexpected error %v", args, err)
			}
		}
	})
	if err := run("-n", "missingmon"); err == nil {
		t.Error("-n missingmon: expected error")
	}
}

// Bug 2: --form regular is a no-op through the full CLI path.
func TestActionFormRegular(t *testing.T) {
	run := func(args ...string) error {
		return newTestApp().Run(context.Background(), append([]string{"pokego"}, args...))
	}
	silent(func() {
		if err := run("--name", "bulbasaur", "--form", "regular"); err != nil {
			t.Errorf("--form regular via Action: unexpected error %v", err)
		}
	})
}

// Bug 4: errors must be printed exactly once to stderr — flag-usage errors
// by cli itself, Action errors by the ExitErrHandler — and both must surface
// as a returned error (which main maps to exit code 1).
// isSpriteOutput reports whether out looks like sprite art: after leading
// indentation it must start with an ANSI escape sequence (a title line would not).
func isSpriteOutput(out string) bool {
	return strings.HasPrefix(strings.TrimLeft(out, " \t"), "\x1b[")
}

// Deliberately locks the Action's switch order: list > version > name > random.
// Reordering the switch for a legitimate reason will break this test on purpose.
func TestFlagPrecedence(t *testing.T) {
	run := func(args ...string) (string, error) {
		var err error
		out := capture(func() {
			err = newTestApp().Run(context.Background(), append([]string{"pokego"}, args...))
		})
		return out, err
	}

	out, err := run("--list", "--version")
	if err != nil {
		t.Fatalf("--list --version: %v", err)
	}
	if got := strings.Count(out, "\n"); got != 905 {
		t.Errorf("--list --version: list should win, got %d lines", got)
	}

	out, err = run("--version", "--name", "pikachu")
	if err != nil {
		t.Fatalf("--version --name: %v", err)
	}
	if out != "dev\n" {
		t.Errorf("--version --name: version should win, got %q", out)
	}

	out, err = run("--name", "pikachu", "--random", "2")
	if err != nil {
		t.Fatalf("--name --random: %v", err)
	}
	if first := strings.SplitN(out, "\n", 2)[0]; first != "pikachu" {
		t.Errorf("--name --random: name should win, title = %q", first)
	}
}

// version defaults to "dev" but release builds inject it via ldflags; the
// Action must print whatever value the var holds.
func TestVersionInjected(t *testing.T) {
	old := version
	defer func() { version = old }()
	version = "1.2.3"

	out := capture(func() {
		if err := newApp().Run(context.Background(), []string{"pokego", "--version"}); err != nil {
			t.Error(err)
		}
	})
	if out != "1.2.3\n" {
		t.Errorf("--version output = %q, want %q", out, "1.2.3\n")
	}
}

// -h is an alias for --help.
func TestHelpAlias(t *testing.T) {
	out := capture(func() {
		if err := newApp().Run(context.Background(), []string{"pokego", "-h"}); err != nil {
			t.Errorf("-h: unexpected error %v", err)
		}
	})
	if !strings.Contains(out, "GLOBAL OPTIONS") || !strings.Contains(out, "--random") {
		t.Error("-h output missing expected help content")
	}
}

// With -s the shiny roll is skipped, so every --random title carries the
// "(shiny)" suffix deterministically.
func TestRandomShinyTitle(t *testing.T) {
	for i := 0; i < 10; i++ {
		out := capture(func() {
			if err := showRandomPokemon("2", true, true); err != nil {
				t.Fatal(err)
			}
		})
		if title := strings.SplitN(out, "\n", 2)[0]; !strings.HasSuffix(title, " (shiny)") {
			t.Fatalf("draw %d: title %q lacks ' (shiny)' suffix", i, title)
		}
	}
}

// --shiny + --form loads the form sprite from the shiny subdirectory, and the
// title reflects both.
func TestShinyFormOutput(t *testing.T) {
	want, err := assets.ReadFile("assets/colorscripts/shiny/pikachu-alola-cap")
	if err != nil {
		t.Fatal(err)
	}
	out := capture(func() {
		if err := showPokemonByName("pikachu", false, true, "alola-cap"); err != nil {
			t.Error(err)
		}
	})
	if out != string(want) {
		t.Errorf("shiny form sprite differs from embedded file (%d vs %d bytes)", len(out), len(want))
	}

	out = capture(func() {
		if err := showPokemonByName("pikachu", true, true, "alola-cap"); err != nil {
			t.Error(err)
		}
	})
	if first := strings.SplitN(out, "\n", 2)[0]; first != "pikachu-alola-cap (shiny)" {
		t.Errorf("title = %q, want %q", first, "pikachu-alola-cap (shiny)")
	}
}

// With the title enabled, output is exactly "<name>\n" followed by the sprite.
func TestTitleThenSprite(t *testing.T) {
	want, err := assets.ReadFile("assets/colorscripts/regular/pikachu")
	if err != nil {
		t.Fatal(err)
	}
	out := capture(func() {
		if err := showPokemonByName("pikachu", true, false, ""); err != nil {
			t.Error(err)
		}
	})
	if out != "pikachu\n"+string(want) {
		t.Errorf("output should be title + sprite (%d bytes, got %d)", len("pikachu\n")+len(want), len(out))
	}
}

// Every sprite is ANSI colorscript art: after leading indentation the output
// must start with an escape sequence (a title line would not).
func TestSpriteIsAnsiArt(t *testing.T) {
	out := capture(func() {
		if err := showPokemonByName("pikachu", false, false, ""); err != nil {
			t.Error(err)
		}
	})
	if !isSpriteOutput(out) {
		t.Errorf("sprite output is not ANSI art: %q", out[:min(20, len(out))])
	}
}

// --random --no-title must not emit a title; the raw sprite is ANSI art.
func TestRandomNoTitleOutput(t *testing.T) {
	out := capture(func() {
		if err := newTestApp().Run(context.Background(), []string{"pokego", "--random", "8", "--no-title"}); err != nil {
			t.Fatal(err)
		}
	})
	if !isSpriteOutput(out) {
		t.Errorf("--random 8 --no-title output should be ANSI art, got %q", out[:min(20, len(out))])
	}
}

// Dex order is the --random sampling assumption: dex N sits at index N-1.
func TestDexOrder(t *testing.T) {
	checks := []struct {
		idx  int
		name string
	}{
		{24, "pikachu"},   // dex 25
		{150, "mew"},      // dex 151
		{808, "melmetal"}, // dex 809 (last of gen 7)
		{809, "grookey"},  // dex 810 (first of gen 8)
		{897, "calyrex"},  // dex 898
	}
	for _, c := range checks {
		if allPokemon[c.idx].Name != c.name {
			t.Errorf("dex %d = %q, want %q", c.idx+1, allPokemon[c.idx].Name, c.name)
		}
	}
}

func TestErrorPrintingOnce(t *testing.T) {
	var out, errBuf bytes.Buffer
	app := newApp()
	app.Writer = &out
	app.ErrWriter = &errBuf

	// Flag-usage error: cli prints "Incorrect Usage: ..." itself; main must
	// not re-print it.
	err := app.Run(context.Background(), []string{"pokego", "--form"})
	if err == nil {
		t.Fatal("--form without a value: expected parse error")
	}
	if got := strings.Count(errBuf.String(), "flag needs an argument"); got != 1 {
		t.Errorf("usage error printed %d times, want 1:\n%s", got, errBuf.String())
	}
	// The help for a usage error goes to stdout, not stderr.
	if !strings.Contains(out.String(), "GLOBAL OPTIONS") {
		t.Errorf("usage-error help missing from stdout:\n%s", out.String())
	}

	// Action error: printed once by the ExitErrHandler.
	errBuf.Reset()
	err = app.Run(context.Background(), []string{"pokego", "--name", "missingmon"})
	if err == nil {
		t.Fatal("--name missingmon: expected error")
	}
	if got := strings.Count(errBuf.String(), "invalid pokemon"); got != 1 {
		t.Errorf("action error printed %d times, want 1:\n%s", got, errBuf.String())
	}
}

// --- Additional tests ---

// buildIndex must map every name to the correct Pokemon entry.
func TestBuildIndex(t *testing.T) {
	for i, p := range allPokemon {
		got, ok := pokemonIndex[p.Name]
		if !ok {
			t.Errorf("pokemonIndex missing %q", p.Name)
			continue
		}
		if got != &allPokemon[i] {
			t.Errorf("pokemonIndex[%q] points to wrong entry", p.Name)
		}
	}
}

// Every pokemon with a form other than "regular" must have a corresponding
// sprite file in both regular and shiny directories (for the form suffix).
func TestFormSpriteFilesExist(t *testing.T) {
	for _, p := range allPokemon {
		for _, f := range p.Forms {
			if f == "regular" {
				continue
			}
			name := p.Name + "-" + f
			for _, subdir := range []string{regularSubdir, shinySubdir} {
				assetPath := path.Join(rootDir, colorscriptsDir, subdir, name)
				if _, err := assets.ReadFile(assetPath); err != nil {
					t.Errorf("missing sprite file %s for %s %s: %v", assetPath, p.Name, f, err)
				}
			}
		}
	}
}

// Every pokemon with just "regular" form must have regular and shiny sprite
// files named after the pokemon (no form suffix).
func TestRegularFormSpriteFilesExist(t *testing.T) {
	for _, p := range allPokemon {
		if len(p.Forms) != 1 || p.Forms[0] != "regular" {
			continue
		}
		for _, subdir := range []string{regularSubdir, shinySubdir} {
			assetPath := path.Join(rootDir, colorscriptsDir, subdir, p.Name)
			if _, err := assets.ReadFile(assetPath); err != nil {
				t.Errorf("missing sprite file %s for %s: %v", assetPath, p.Name, err)
			}
		}
	}
}

// showPokemonByName must accept every valid form for pokemon that have them.
func TestShowPokemonAllForms(t *testing.T) {
	for _, p := range allPokemon {
		for _, f := range p.Forms {
			formArg := f
			if formArg == "regular" {
				formArg = "" // regular is the no-op
			}
			silent(func() {
				if err := showPokemonByName(p.Name, false, false, formArg); err != nil {
					t.Errorf("showPokemonByName(%q, form=%q): %v", p.Name, f, err)
				}
			})
		}
	}
}

// Range "1-3" must only sample pokemon from gens 1, 2, and 3.
func TestRandomRange1to3Coverage(t *testing.T) {
	expected := make(map[string]bool)
	for _, g := range []string{"1", "2", "3"} {
		r := generations[g]
		for i := r[0]; i <= r[1]; i++ {
			expected[allPokemon[i-1].Name] = true
		}
	}
	for i := 0; i < 100; i++ {
		out := capture(func() {
			if err := showRandomPokemon("1-3", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 1-3 sampled %q outside gens 1-3", title)
		}
	}
}

// Range "5-7" must only sample pokemon from gens 5, 6, and 7.
func TestRandomRange5to7Coverage(t *testing.T) {
	expected := make(map[string]bool)
	for _, g := range []string{"5", "6", "7"} {
		r := generations[g]
		for i := r[0]; i <= r[1]; i++ {
			expected[allPokemon[i-1].Name] = true
		}
	}
	for i := 0; i < 100; i++ {
		out := capture(func() {
			if err := showRandomPokemon("5-7", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 5-7 sampled %q outside gens 5-7", title)
		}
	}
}

// Each generation boundary: first and last pokemon of each gen.
func TestGenerationBoundaries(t *testing.T) {
	boundaries := []struct {
		gen   string
		first string
		last  string
	}{
		{"1", "bulbasaur", "mew"},
		{"2", "chikorita", "celebi"},
		{"3", "treecko", "deoxys"},
		{"4", "turtwig", "arceus"},
		{"5", "victini", "genesect"},
		{"6", "chespin", "volcanion"},
		{"7", "rowlet", "melmetal"},
		{"8", "grookey", "enamorus"},
	}
	for _, b := range boundaries {
		r := generations[b.gen]
		if got := allPokemon[r[0]-1].Name; got != b.first {
			t.Errorf("gen %s first = %q, want %q", b.gen, got, b.first)
		}
		if got := allPokemon[r[1]-1].Name; got != b.last {
			t.Errorf("gen %s last = %q, want %q", b.gen, got, b.last)
		}
	}
}

// Charizard has mega-x, mega-y, and gmax forms; all must resolve.
func TestCharizardForms(t *testing.T) {
	for _, form := range []string{"mega-x", "mega-y", "gmax"} {
		silent(func() {
			if err := showPokemonByName("charizard", false, false, form); err != nil {
				t.Errorf("charizard form %q: %v", form, err)
			}
		})
	}
}

// Case insensitive errors for missing pokemon.
func TestShowPokemonByNameCaseInsensitive(t *testing.T) {
	// These are all the same pokemon, just different cases.
	for _, name := range []string{"PIKACHU", "Pikachu", "pikaCHU"} {
		silent(func() {
			if err := showPokemonByName(name, false, false, ""); err != nil {
				t.Errorf("showPokemonByName(%q): %v", name, err)
			}
		})
	}
}

// Multiple comma-separated with trailing and leading commas.
func TestShowRandomPokemonCommaEdgeCases(t *testing.T) {
	tests := []struct {
		gen   string
		valid bool
	}{
		{"1,2", true},
		{" 1 , 2 ", true},
		{"1,,2", true},
		{",", false},
		{"  ", false},
	}
	for _, tc := range tests {
		err := showRandomPokemon(tc.gen, false, false)
		if tc.valid && err != nil {
			t.Errorf("--random %q: unexpected error %v", tc.gen, err)
		} else if !tc.valid && err == nil {
			t.Errorf("--random %q: expected error, got nil", tc.gen)
		}
	}
}

// Form trimming and case insensitivity through the full path.
func TestShowPokemonByNameFormTrimAndCase(t *testing.T) {
	forms := []struct {
		input string
		valid bool
	}{
		{" MEGA-X ", true},
		{"mega-x", true},
		{"MEGA-X", true},
		{" Mega-X ", true},
		{"mega_x", false},
	}
	for _, f := range forms {
		err := showPokemonByName("charizard", false, false, f.input)
		if f.valid && err != nil {
			t.Errorf("charizard form %q: unexpected error %v", f.input, err)
		} else if !f.valid && err == nil {
			t.Errorf("charizard form %q: expected error, got nil", f.input)
		}
	}
}

// The title for a pokemon with a form and shiny must include both.
func TestTitleFormShiny(t *testing.T) {
	out := capture(func() {
		if err := showPokemonByName("charizard", true, true, "mega-x"); err != nil {
			t.Fatal(err)
		}
	})
	title := strings.SplitN(out, "\n", 2)[0]
	if title != "charizard-mega-x (shiny)" {
		t.Errorf("title = %q, want %q", title, "charizard-mega-x (shiny)")
	}
}

// The title for a pokemon with a form (no shiny) must show name-form.
func TestTitleFormNoShiny(t *testing.T) {
	out := capture(func() {
		if err := showPokemonByName("charizard", true, false, "mega-y"); err != nil {
			t.Fatal(err)
		}
	})
	title := strings.SplitN(out, "\n", 2)[0]
	if title != "charizard-mega-y" {
		t.Errorf("title = %q, want %q", title, "charizard-mega-y")
	}
}

// Missing pokemon error must always say "invalid pokemon".
func TestErrorMessagesMissingPokemon(t *testing.T) {
	for _, name := range []string{"missingmon", "MISSINGMON", "missing"} {
		err := showPokemonByName(name, false, false, "")
		if err == nil {
			t.Errorf("%q: expected error", name)
			continue
		}
		if !strings.Contains(err.Error(), "invalid pokemon") {
			t.Errorf("%q: error %q missing 'invalid pokemon'", name, err.Error())
		}
	}
}

// Invalid form error must always say "invalid form".
func TestErrorMessagesInvalidForm(t *testing.T) {
	err := showPokemonByName("pikachu", false, false, "bogus")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid form 'bogus' for pokemon pikachu") {
		t.Errorf("error = %q", err.Error())
	}
}

// The alternates list for invalid form must exclude "regular".
func TestErrorAlternatesExcludesRegular(t *testing.T) {
	err := showPokemonByName("charizard", false, false, "bogus")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "available alternate forms are:") {
		t.Errorf("missing 'available alternate forms are:' in: %s", msg)
	}
	if strings.Contains(msg, "- regular") {
		t.Errorf("alternates must not include 'regular': %s", msg)
	}
	for _, want := range []string{"- gmax", "- mega-x", "- mega-y"} {
		if !strings.Contains(msg, want) {
			t.Errorf("alternates missing %q: %s", want, msg)
		}
	}
}

// --random with every single generation individually.
func TestRandomEachGeneration(t *testing.T) {
	for gen := range generations {
		silent(func() {
			if err := showRandomPokemon(gen, false, false); err != nil {
				t.Errorf("--random %s: %v", gen, err)
			}
		})
	}
}

// --list output must match allPokemon in dex order.
func TestListOutputDexOrder(t *testing.T) {
	out := capture(listPokemonNames)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != len(allPokemon) {
		t.Fatalf("list has %d lines, want %d", len(lines), len(allPokemon))
	}
	for i, p := range allPokemon {
		if lines[i] != p.Name {
			t.Errorf("line %d = %q, want %q (dex order)", i+1, lines[i], p.Name)
		}
	}
}

// printFile must read the exact bytes from the embedded asset.
func TestPrintFileExactBytes(t *testing.T) {
	want, err := assets.ReadFile("assets/colorscripts/regular/bulbasaur")
	if err != nil {
		t.Fatal(err)
	}
	out := capture(func() {
		if err := printFile("assets/colorscripts/regular/bulbasaur"); err != nil {
			t.Fatal(err)
		}
	})
	if out != string(want) {
		t.Errorf("printFile output differs: %d vs %d bytes", len(out), len(want))
	}
}

// printFile must error on nonexistent paths.
func TestPrintFileNonexistent(t *testing.T) {
	err := printFile("assets/colorscripts/regular/doesnotexist")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// Shiny + form sprite must match the embedded file byte-for-byte.
func TestShinyFormSpriteMatchesFile(t *testing.T) {
	want, err := assets.ReadFile("assets/colorscripts/shiny/charizard-mega-x")
	if err != nil {
		t.Fatal(err)
	}
	out := capture(func() {
		if err := showPokemonByName("charizard", false, true, "mega-x"); err != nil {
			t.Fatal(err)
		}
	})
	if out != string(want) {
		t.Errorf("shiny mega-x sprite differs (%d vs %d bytes)", len(out), len(want))
	}
}

// --random with --no-title must produce only ANSI art for every generation.
func TestRandomNoTitleAllGens(t *testing.T) {
	for gen := range generations {
		out := capture(func() {
			if err := newTestApp().Run(context.Background(), []string{"pokego", "--random", gen, "--no-title"}); err != nil {
				t.Fatal(err)
			}
		})
		if !isSpriteOutput(out) {
			t.Errorf("--random %s --no-title: not ANSI art", gen)
		}
	}
}

// --list line count must match allPokemon length exactly.
func TestListLineCountMatchesData(t *testing.T) {
	out := capture(listPokemonNames)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != len(allPokemon) {
		t.Errorf("list lines %d != allPokemon %d", len(lines), len(allPokemon))
	}
}

// Action with --random and --form must error.
func TestActionRandomWithFormError(t *testing.T) {
	err := newTestApp().Run(context.Background(), []string{"pokego", "--random", "3", "--form", "mega"})
	if err == nil {
		t.Error("--random with --form: expected error")
	}
}

// Action with --name and --random: name wins (tested but worth reinforcing).
func TestActionNameOverRandom(t *testing.T) {
	out, err := capture2(func() error {
		return newTestApp().Run(context.Background(), []string{"pokego", "--name", "bulbasaur", "--random", "1"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if first := strings.SplitN(out, "\n", 2)[0]; first != "bulbasaur" {
		t.Errorf("name should win: got title %q", first)
	}
}

// Action with --version and --name: version wins.
func TestActionVersionOverName(t *testing.T) {
	out, err := capture2(func() error {
		return newTestApp().Run(context.Background(), []string{"pokego", "--version", "--name", "pikachu"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "dev\n" {
		t.Errorf("version should win: got %q", out)
	}
}

// Action with --list and --name: list wins.
func TestActionListOverName(t *testing.T) {
	out, err := capture2(func() error {
		return newTestApp().Run(context.Background(), []string{"pokego", "--list", "--name", "pikachu"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out, "\n"); got != 905 {
		t.Errorf("list should win: got %d lines", got)
	}
}

// Every pokemon with a form file must have a valid sprite with nonzero size.
func TestFormSpritesNonzero(t *testing.T) {
	for _, p := range allPokemon {
		for _, f := range p.Forms {
			if f == "regular" {
				continue
			}
			name := p.Name + "-" + f
			for _, subdir := range []string{regularSubdir, shinySubdir} {
				data, err := assets.ReadFile(path.Join(rootDir, colorscriptsDir, subdir, name))
				if err != nil {
					t.Errorf("%s/%s: %v", subdir, name, err)
					continue
				}
				if len(data) == 0 {
					t.Errorf("%s/%s: sprite is empty", subdir, name)
				}
			}
		}
	}
}

// capture2 is like capture but also returns an error from the function.
func capture2(fn func() error) (string, error) {
	old := os.Stdout
	f, err := os.CreateTemp("", "pokego-out-*")
	if err != nil {
		panic(err)
	}
	os.Stdout = f
	defer func() {
		os.Stdout = old
		f.Close()
		os.Remove(f.Name())
	}()
	err = fn()
	f.Close()
	b, readErr := os.ReadFile(f.Name())
	if readErr != nil {
		panic(readErr)
	}
	return string(b), err
}

// --random range "1-1" is equivalent to "1".
func TestRandomRangeSameGen(t *testing.T) {
	expected := make(map[string]bool)
	r := generations["1"]
	for i := r[0]; i <= r[1]; i++ {
		expected[allPokemon[i-1].Name] = true
	}
	for i := 0; i < 50; i++ {
		out := capture(func() {
			if err := showRandomPokemon("1-1", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 1-1 sampled %q outside gen 1", title)
		}
	}
}

// --random 2 must only sample gen 2 pokemon (dex 152-251).
func TestRandomGen2Coverage(t *testing.T) {
	r := generations["2"]
	expected := make(map[string]bool)
	for i := r[0]; i <= r[1]; i++ {
		expected[allPokemon[i-1].Name] = true
	}
	seen := make(map[string]bool)
	for i := 0; i < 500; i++ {
		out := capture(func() {
			if err := showRandomPokemon("2", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 2 sampled %q outside gen 2", title)
		}
		seen[title] = true
	}
	// Verify coverage of both ends of gen 2.
	for _, name := range []string{"chikorita", "celebi"} {
		if !seen[name] {
			t.Errorf("%s never sampled in gen 2 draws", name)
		}
	}
}

// --random with shiny produces the (shiny) suffix in title.
func TestRandomShinyTitleSuffix(t *testing.T) {
	for i := 0; i < 5; i++ {
		out := capture(func() {
			if err := showRandomPokemon("4", true, true); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.SplitN(out, "\n", 2)[0]
		if !strings.HasSuffix(title, " (shiny)") {
			t.Fatalf("draw %d: title %q lacks ' (shiny)'", i, title)
		}
	}
}

// --random with no title and no shiny must be pure ANSI art.
func TestRandomNoTitleNoShinyPureArt(t *testing.T) {
	out := capture(func() {
		if err := newTestApp().Run(context.Background(), []string{"pokego", "--random", "3", "--no-title"}); err != nil {
			t.Fatal(err)
		}
	})
	if len(out) == 0 {
		t.Fatal("empty output")
	}
	if !isSpriteOutput(out) {
		t.Errorf("output is not ANSI art: %q", out[:min(20, len(out))])
	}
}

// The full title+sprite output for shiny form must start with title line then ANSI.
func TestTitleThenSpriteShinyForm(t *testing.T) {
	want, err := assets.ReadFile("assets/colorscripts/shiny/charizard-mega-x")
	if err != nil {
		t.Fatal(err)
	}
	out := capture(func() {
		if err := showPokemonByName("charizard", true, true, "mega-x"); err != nil {
			t.Fatal(err)
		}
	})
	expected := "charizard-mega-x (shiny)\n" + string(want)
	if out != expected {
		t.Errorf("output mismatch: got %d bytes, want %d", len(out), len(expected))
	}
}

// listPokemonNames must output exactly one newline per pokemon, no trailing blank.
func TestListNoTrailingBlank(t *testing.T) {
	out := capture(listPokemonNames)
	if strings.HasSuffix(out, "\n\n") {
		t.Error("list has trailing blank line")
	}
	if !strings.HasSuffix(out, "\n") {
		t.Error("list does not end with newline")
	}
}

// showError message for invalid form must not have trailing newline.
func TestInvalidFormErrorNoTrailingNewline(t *testing.T) {
	err := showPokemonByName("pikachu", false, false, "bogus")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.HasSuffix(msg, "\n") {
		t.Errorf("error has trailing newline: %q", msg)
	}
}

// showError message for missing pokemon must not have trailing newline.
func TestMissingPokemonErrorNoTrailingNewline(t *testing.T) {
	err := showPokemonByName("missingmon", false, false, "")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.HasSuffix(msg, "\n") {
		t.Errorf("error has trailing newline: %q", msg)
	}
}

// --random comma list "2,5,8" must only sample gens 2, 5, or 8.
func TestRandomCommaListCoverage(t *testing.T) {
	expected := make(map[string]bool)
	for _, g := range []string{"2", "5", "8"} {
		r := generations[g]
		for i := r[0]; i <= r[1]; i++ {
			expected[allPokemon[i-1].Name] = true
		}
	}
	for i := 0; i < 100; i++ {
		out := capture(func() {
			if err := showRandomPokemon("2,5,8", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 2,5,8 sampled %q outside gens 2/5/8", title)
		}
	}
}

// Every generation range "N-N" must sample correctly.
func TestRandomSameGenRangeAll(t *testing.T) {
	for gen := range generations {
		r := generations[gen]
		expected := make(map[string]bool)
		for i := r[0]; i <= r[1]; i++ {
			expected[allPokemon[i-1].Name] = true
		}
		for i := 0; i < 20; i++ {
			out := capture(func() {
				if err := showRandomPokemon(gen+"-"+gen, true, false); err != nil {
					t.Fatal(err)
				}
			})
			title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
			if !expected[title] {
				t.Fatalf("--random %s-%s sampled %q outside gen %s", gen, gen, title, gen)
			}
		}
	}
}

// Shiny title format must not have extra spaces.
func TestShinyTitleFormatNoExtraSpaces(t *testing.T) {
	out := capture(func() {
		if err := showRandomPokemon("1", true, true); err != nil {
			t.Fatal(err)
		}
	})
	title := strings.SplitN(out, "\n", 2)[0]
	if strings.Contains(title, "  ") {
		t.Errorf("title has double space: %q", title)
	}
	if !strings.HasSuffix(title, " (shiny)") {
		t.Errorf("title missing shiny suffix: %q", title)
	}
}

// --random range spanning across gen boundary "7-8".
func TestRandomRangeAcrossBoundary(t *testing.T) {
	expected := make(map[string]bool)
	for _, g := range []string{"7", "8"} {
		r := generations[g]
		for i := r[0]; i <= r[1]; i++ {
			expected[allPokemon[i-1].Name] = true
		}
	}
	seen := make(map[string]bool)
	for i := 0; i < 200; i++ {
		out := capture(func() {
			if err := showRandomPokemon("7-8", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 7-8 sampled %q outside gens 7-8", title)
		}
		seen[title] = true
	}
	// Both gen 7 and gen 8 pokemon should appear.
	gen7 := allPokemon[generations["7"][0]-1].Name
	gen8 := allPokemon[generations["8"][0]-1].Name
	if !seen[gen7] {
		t.Errorf("first gen 7 pokemon %q never sampled", gen7)
	}
	if !seen[gen8] {
		t.Errorf("first gen 8 pokemon %q never sampled", gen8)
	}
}

// Error for reversed range like "8-1".
func TestRandomReversedRange(t *testing.T) {
	err := showRandomPokemon("8-1", false, false)
	if err == nil {
		t.Error("--random 8-1: expected error")
	}
}

// Error for range with invalid start but valid end.
func TestRandomInvalidStartValidEnd(t *testing.T) {
	err := showRandomPokemon("99-8", false, false)
	if err == nil {
		t.Error("--random 99-8: expected error")
	}
}

// Error for valid start but invalid end.
func TestRandomValidStartInvalidEnd(t *testing.T) {
	err := showRandomPokemon("1-99", false, false)
	if err == nil {
		t.Error("--random 1-99: expected error")
	}
}

// --list must have no extra whitespace on lines.
func TestListLinesNoExtraWhitespace(t *testing.T) {
	out := capture(listPokemonNames)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	for i, line := range lines {
		if line != strings.TrimSpace(line) {
			t.Errorf("line %d has extra whitespace: %q", i+1, line)
		}
		if line != strings.ToLower(line) {
			t.Errorf("line %d is not lowercase: %q", i+1, line)
		}
	}
}

// Shiny sprite must also be valid ANSI art.
func TestShinySpriteIsAnsiArt(t *testing.T) {
	out := capture(func() {
		if err := showPokemonByName("pikachu", false, true, ""); err != nil {
			t.Fatal(err)
		}
	})
	if !isSpriteOutput(out) {
		t.Errorf("shiny sprite is not ANSI art")
	}
}

// Shiny form sprite must be valid ANSI art.
func TestShinyFormSpriteIsAnsiArt(t *testing.T) {
	out := capture(func() {
		if err := showPokemonByName("pikachu", false, true, "alola-cap"); err != nil {
			t.Fatal(err)
		}
	})
	if !isSpriteOutput(out) {
		t.Errorf("shiny form sprite is not ANSI art")
	}
}

// Form with regular name appended as suffix must produce valid output.
func TestRegularFormAppendedAsSuffix(t *testing.T) {
	out := capture(func() {
		if err := showPokemonByName("bulbasaur", true, false, "regular"); err != nil {
			t.Fatal(err)
		}
	})
	if first := strings.SplitN(out, "\n", 2)[0]; first != "bulbasaur" {
		t.Errorf("title = %q, want %q (regular should be no-op)", first, "bulbasaur")
	}
}

// --random 8 must always produce output for every generation range type.
func TestRandomGen8AlwaysOutput(t *testing.T) {
	for i := 0; i < 10; i++ {
		out := capture(func() {
			if err := newTestApp().Run(context.Background(), []string{"pokego", "--random", "8", "--no-title"}); err != nil {
				t.Fatal(err)
			}
		})
		if len(out) == 0 {
			t.Fatalf("draw %d: empty output", i)
		}
	}
}

// --random range "3-5" must only include gens 3, 4, 5.
func TestRandomRange3to5(t *testing.T) {
	expected := make(map[string]bool)
	for _, g := range []string{"3", "4", "5"} {
		r := generations[g]
		for i := r[0]; i <= r[1]; i++ {
			expected[allPokemon[i-1].Name] = true
		}
	}
	for i := 0; i < 80; i++ {
		out := capture(func() {
			if err := showRandomPokemon("3-5", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 3-5 sampled %q outside gens 3-5", title)
		}
	}
}

// Action with --random and multiple invalid generations.
func TestActionRandomMultipleInvalid(t *testing.T) {
	err := newTestApp().Run(context.Background(), []string{"pokego", "--random", "99,100"})
	if err == nil {
		t.Error("--random 99,100: expected error")
	}
}

// Action with --random range and --form must error.
func TestActionRandomRangeWithFormError(t *testing.T) {
	err := newTestApp().Run(context.Background(), []string{"pokego", "--random", "1-3", "--form", "mega"})
	if err == nil {
		t.Error("--random 1-3 with --form: expected error")
	}
}

// The version string must be settable and reflectable.
func TestVersionReflectsVar(t *testing.T) {
	old := version
	defer func() { version = old }()
	version = "9.9.9"
	out := capture(func() {
		if err := newApp().Run(context.Background(), []string{"pokego", "--version"}); err != nil {
			t.Fatal(err)
		}
	})
	if out != "9.9.9\n" {
		t.Errorf("version = %q, want %q", out, "9.9.9\n")
	}
}

// --random with every generation from 1-8 must not error.
func TestRandomAllGensNoError(t *testing.T) {
	gens := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	for _, g := range gens {
		silent(func() {
			if err := showRandomPokemon(g, false, false); err != nil {
				t.Errorf("--random %s: %v", g, err)
			}
		})
	}
}

// The form error alternates list must not include the bogus form itself.
func TestAlternatesNotIncludeBogus(t *testing.T) {
	err := showPokemonByName("pikachu", false, false, "bogus")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, "- bogus") {
		t.Errorf("alternates must not include bogus: %s", msg)
	}
}

// --random range "6-8" must not include gen 1-5 pokemon.
func TestRandomRange6to8(t *testing.T) {
	expected := make(map[string]bool)
	for _, g := range []string{"6", "7", "8"} {
		r := generations[g]
		for i := r[0]; i <= r[1]; i++ {
			expected[allPokemon[i-1].Name] = true
		}
	}
	for i := 0; i < 80; i++ {
		out := capture(func() {
			if err := showRandomPokemon("6-8", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 6-8 sampled %q outside gens 6-8", title)
		}
	}
}

// --random with whitespace-only string must error.
func TestRandomWhitespaceOnly(t *testing.T) {
	err := showRandomPokemon("   ", false, false)
	if err == nil {
		t.Error("--random '   ': expected error")
	}
}

// --random with empty string must error.
func TestRandomEmptyString(t *testing.T) {
	err := showRandomPokemon("", false, false)
	if err == nil {
		t.Error("--random '': expected error")
	}
}

// --random single generation list must only sample that gen.
func TestRandomSingleGenList(t *testing.T) {
	for _, gen := range []string{"1", "3", "5", "8"} {
		r := generations[gen]
		expected := make(map[string]bool)
		for i := r[0]; i <= r[1]; i++ {
			expected[allPokemon[i-1].Name] = true
		}
		for i := 0; i < 30; i++ {
			out := capture(func() {
				if err := showRandomPokemon(gen, true, false); err != nil {
					t.Fatal(err)
				}
			})
			title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
			if !expected[title] {
				t.Fatalf("--random %s sampled %q outside gen %s", gen, title, gen)
			}
		}
	}
}

// --list alias -l must produce the same output.
func TestListAliasOutput(t *testing.T) {
	out := capture(func() {
		if err := newTestApp().Run(context.Background(), []string{"pokego", "-l"}); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 905 {
		t.Errorf("-l printed %d lines, want 905", len(lines))
	}
}

// --shiny alias -s with --name must work.
func TestShinyAliasWithNames(t *testing.T) {
	silent(func() {
		if err := newTestApp().Run(context.Background(), []string{"pokego", "-n", "pikachu", "-s", "-nt"}); err != nil {
			t.Errorf("-n pikachu -s -nt: %v", err)
		}
	})
}

// --random alias -r with generation list.
func TestRandomAliasWithList(t *testing.T) {
	silent(func() {
		if err := newTestApp().Run(context.Background(), []string{"pokego", "-r", "1,3,6"}); err != nil {
			t.Errorf("-r 1,3,6: %v", err)
		}
	})
}

// --form alias -f with valid form.
func TestFormAliasWithValidForm(t *testing.T) {
	silent(func() {
		if err := newTestApp().Run(context.Background(), []string{"pokego", "-n", "pikachu", "-f", "alola-cap"}); err != nil {
			t.Errorf("-n pikachu -f alola-cap: %v", err)
		}
	})
}

// --version alias -v must work.
func TestVersionAlias(t *testing.T) {
	out := capture(func() {
		if err := newTestApp().Run(context.Background(), []string{"pokego", "-v"}); err != nil {
			t.Fatal(err)
		}
	})
	if out != "dev\n" {
		t.Errorf("-v output = %q, want %q", out, "dev\n")
	}
}

// --name alias -n with --form alias -f and --no-title alias -nt.
func TestMultipleAliases(t *testing.T) {
	silent(func() {
		if err := newTestApp().Run(context.Background(), []string{"pokego", "-n", "pikachu", "-f", "alola-cap", "-nt"}); err != nil {
			t.Errorf("multiple aliases: %v", err)
		}
	})
}

// --random with shiny and title must always have (shiny) suffix.
func TestRandomShinyAlwaysSuffix(t *testing.T) {
	for i := 0; i < 20; i++ {
		out := capture(func() {
			if err := showRandomPokemon("6", true, true); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.SplitN(out, "\n", 2)[0]
		if !strings.HasSuffix(title, " (shiny)") {
			t.Fatalf("draw %d: title %q lacks ' (shiny)'", i, title)
		}
	}
}

// Form with trailing spaces only must be treated as empty.
func TestFormSpacesOnlyIsNoop(t *testing.T) {
	silent(func() {
		if err := showPokemonByName("pikachu", false, false, "   "); err != nil {
			t.Errorf("form with only spaces: %v", err)
		}
	})
}

// Every generation's start must be 1 more than the previous gen's end.
func TestGenerationRangesContiguous(t *testing.T) {
	prevEnd := 0
	for _, gen := range []string{"1", "2", "3", "4", "5", "6", "7", "8"} {
		r, ok := generations[gen]
		if !ok {
			t.Fatalf("missing generation %s", gen)
		}
		if r[0] != prevEnd+1 {
			t.Errorf("gen %s start = %d, want %d", gen, r[0], prevEnd+1)
		}
		prevEnd = r[1]
	}
}

// --random 4 must cover all gen 4 dex entries (387-493).
func TestRandomGen4Coverage(t *testing.T) {
	r := generations["4"]
	expected := make(map[string]bool)
	for i := r[0]; i <= r[1]; i++ {
		expected[allPokemon[i-1].Name] = true
	}
	seen := make(map[string]bool)
	for i := 0; i < 500; i++ {
		out := capture(func() {
			if err := showRandomPokemon("4", true, false); err != nil {
				t.Fatal(err)
			}
		})
		title := strings.TrimSuffix(strings.SplitN(out, "\n", 2)[0], " (shiny)")
		if !expected[title] {
			t.Fatalf("--random 4 sampled %q outside gen 4", title)
		}
		seen[title] = true
	}
	// Verify both ends.
	for _, name := range []string{"turtwig", "arceus"} {
		if !seen[name] {
			t.Errorf("%s never sampled in gen 4 draws", name)
		}
	}
}
