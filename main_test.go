package main

import (
	"bytes"
	"context"
	"io"
	"os"
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
		if !strings.Contains(out, "pokego") {
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
