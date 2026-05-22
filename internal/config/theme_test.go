package config

import (
	"os"
	"path/filepath"
	"testing"
)

// ── Default ──────────────────────────────────────────────────────────────────

func TestDefaultIsComplete(t *testing.T) {
	d := Default()
	fields := map[string]string{
		"Title": d.Title, "Subtitle": d.Subtitle,
		"Label": d.Label, "Value": d.Value,
		"CardBorder": d.CardBorder,
		"TabActiveFG": d.TabActiveFG, "TabActiveBG": d.TabActiveBG,
		"TabFG": d.TabFG, "TabBG": d.TabBG,
		"Footer": d.Footer, "Status": d.Status,
		"Ok": d.Ok, "Warn": d.Warn, "Err": d.Err,
		"SignalGreat": d.SignalGreat, "SignalGood": d.SignalGood,
		"SignalFair":  d.SignalFair,  "SignalPoor": d.SignalPoor,
		"SignalNone":  d.SignalNone,
		"LatencyGreat": d.LatencyGreat, "LatencyGood": d.LatencyGood,
		"LatencyFair":  d.LatencyFair,  "LatencyBad":  d.LatencyBad,
		"LossGreat": d.LossGreat, "LossFair": d.LossFair, "LossBad": d.LossBad,
	}
	for name, val := range fields {
		if val == "" {
			t.Errorf("Default().%s is empty", name)
		}
	}
}

// ── Load – no file ────────────────────────────────────────────────────────────

func TestLoadNoFileReturnsDefaults(t *testing.T) {
	got  := Load("/nonexistent/path/wifimon.ini")
	want := Default()
	if got.Title != want.Title || got.SignalGreat != want.SignalGreat {
		t.Errorf("expected defaults when file missing, got %+v", got)
	}
}

// ── Load – full override ──────────────────────────────────────────────────────

func TestLoadOverridesAllKeys(t *testing.T) {
	ini := `
[theme]
title          = 99
subtitle       = 88
label          = 77
value          = 66
card_border    = 55
tab_active_fg  = 44
tab_active_bg  = 33
tab_fg         = 22
tab_bg         = 11
footer         = 10
status         = 9
ok             = 8
warn           = 7
err            = 6
signal_great   = 5
signal_good    = 4
signal_fair    = 3
signal_poor    = 2
signal_none    = 1
latency_great  = 51
latency_good   = 52
latency_fair   = 53
latency_bad    = 54
loss_great     = 61
loss_fair      = 62
loss_bad       = 63
`
	path := writeTempIni(t, ini)
	got  := Load(path)

	check := func(name, want, got string) {
		t.Helper()
		if got != want {
			t.Errorf("%s: want %q, got %q", name, want, got)
		}
	}

	check("Title",       "99", got.Title)
	check("Subtitle",    "88", got.Subtitle)
	check("Label",       "77", got.Label)
	check("Value",       "66", got.Value)
	check("CardBorder",  "55", got.CardBorder)
	check("TabActiveFG", "44", got.TabActiveFG)
	check("TabActiveBG", "33", got.TabActiveBG)
	check("TabFG",       "22", got.TabFG)
	check("TabBG",       "11", got.TabBG)
	check("Footer",      "10", got.Footer)
	check("Status",       "9", got.Status)
	check("Ok",           "8", got.Ok)
	check("Warn",         "7", got.Warn)
	check("Err",          "6", got.Err)
	check("SignalGreat",  "5", got.SignalGreat)
	check("SignalGood",   "4", got.SignalGood)
	check("SignalFair",   "3", got.SignalFair)
	check("SignalPoor",   "2", got.SignalPoor)
	check("SignalNone",   "1", got.SignalNone)
	check("LatencyGreat","51", got.LatencyGreat)
	check("LatencyGood", "52", got.LatencyGood)
	check("LatencyFair", "53", got.LatencyFair)
	check("LatencyBad",  "54", got.LatencyBad)
	check("LossGreat",   "61", got.LossGreat)
	check("LossFair",    "62", got.LossFair)
	check("LossBad",     "63", got.LossBad)
}

// ── Load – partial override keeps defaults ────────────────────────────────────

func TestLoadPartialKeepsDefaults(t *testing.T) {
	ini := `
[theme]
title = #ff0000
`
	path := writeTempIni(t, ini)
	got  := Load(path)
	def  := Default()

	if got.Title != "#ff0000" {
		t.Errorf("title: want #ff0000, got %q", got.Title)
	}
	// everything else must equal the default
	if got.SignalGreat != def.SignalGreat {
		t.Errorf("signal_great should be default %q, got %q", def.SignalGreat, got.SignalGreat)
	}
	if got.CardBorder != def.CardBorder {
		t.Errorf("card_border should be default %q, got %q", def.CardBorder, got.CardBorder)
	}
}

// ── Load – comments and blank lines are ignored ───────────────────────────────

func TestLoadIgnoresCommentsAndBlanks(t *testing.T) {
	ini := `
# this whole file is comments and blanks

; another comment style
[theme]
; title = should not apply
# label = should not apply either

`
	path := writeTempIni(t, ini)
	got  := Load(path)
	def  := Default()

	if got.Title != def.Title {
		t.Errorf("title should be default, got %q", got.Title)
	}
}

// ── Load – inline comments stripped ──────────────────────────────────────────

func TestLoadStripsInlineComments(t *testing.T) {
	ini := `
[theme]
title = 75   # nice blue
label = 109  ; another comment
`
	path := writeTempIni(t, ini)
	got  := Load(path)

	if got.Title != "75" {
		t.Errorf("title: want 75, got %q", got.Title)
	}
	if got.Label != "109" {
		t.Errorf("label: want 109, got %q", got.Label)
	}
}

// ── Load – keys outside [theme] section still applied ────────────────────────

func TestLoadAcceptsKeysOutsideSection(t *testing.T) {
	ini := `title = 200
label = 201
`
	path := writeTempIni(t, ini)
	got  := Load(path)

	if got.Title != "200" {
		t.Errorf("title outside section: want 200, got %q", got.Title)
	}
	if got.Label != "201" {
		t.Errorf("label outside section: want 201, got %q", got.Label)
	}
}

// ── Load – unknown keys are silently ignored ──────────────────────────────────

func TestLoadIgnoresUnknownKeys(t *testing.T) {
	ini := `
[theme]
title           = 55
unknown_key     = 999
another_unknown = foo
`
	path := writeTempIni(t, ini)
	// Should not panic.
	got := Load(path)
	if got.Title != "55" {
		t.Errorf("title: want 55, got %q", got.Title)
	}
}

// ── Load – hex colour values are preserved as-is ─────────────────────────────

func TestLoadHexColors(t *testing.T) {
	ini := `
[theme]
title = #1a2b3c
ok    = #00ff87
err   = #ff005f
`
	path := writeTempIni(t, ini)
	got  := Load(path)

	if got.Title != "#1a2b3c" {
		t.Errorf("title hex: want #1a2b3c, got %q", got.Title)
	}
	if got.Ok != "#00ff87" {
		t.Errorf("ok hex: want #00ff87, got %q", got.Ok)
	}
}

// ── WIFIMON_CONFIG env var ────────────────────────────────────────────────────

func TestLoadEnvVar(t *testing.T) {
	ini := "[theme]\ntitle = 123\n"
	path := writeTempIni(t, ini)

	t.Setenv("WIFIMON_CONFIG", path)
	got := Load("") // no explicit path — should pick up env var
	if got.Title != "123" {
		t.Errorf("env var load: want 123, got %q", got.Title)
	}
}

// ── SignalColor ───────────────────────────────────────────────────────────────

func TestSignalColor(t *testing.T) {
	d := Default()
	cases := []struct{ pct int; want string }{
		{100, d.SignalGreat},
		{85,  d.SignalGreat},
		{84,  d.SignalGood},
		{65,  d.SignalGood},
		{64,  d.SignalFair},
		{45,  d.SignalFair},
		{44,  d.SignalPoor},
		{20,  d.SignalPoor},
		{19,  d.SignalNone},
		{0,   d.SignalNone},
	}
	for _, c := range cases {
		got := d.SignalColor(c.pct)
		if got != c.want {
			t.Errorf("SignalColor(%d): want %q, got %q", c.pct, c.want, got)
		}
	}
}

// ── LatencyColor ──────────────────────────────────────────────────────────────

func TestLatencyColor(t *testing.T) {
	d := Default()
	cases := []struct {
		ms        int
		reachable bool
		want      string
	}{
		{0,   false, d.LatencyBad},
		{5,   true,  d.LatencyGreat},
		{20,  true,  d.LatencyGreat},
		{21,  true,  d.LatencyGood},
		{60,  true,  d.LatencyGood},
		{61,  true,  d.LatencyFair},
		{120, true,  d.LatencyFair},
		{121, true,  d.LatencyBad},
		{999, true,  d.LatencyBad},
	}
	for _, c := range cases {
		got := d.LatencyColor(c.ms, c.reachable)
		if got != c.want {
			t.Errorf("LatencyColor(%d, %v): want %q, got %q",
				c.ms, c.reachable, c.want, got)
		}
	}
}

// ── LossColor ─────────────────────────────────────────────────────────────────

func TestLossColor(t *testing.T) {
	d := Default()
	cases := []struct {
		pct  float64
		want string
	}{
		{0,    d.LossGreat},
		{0.1,  d.LossFair},
		{5,    d.LossFair},
		{5.1,  d.LossBad},
		{100,  d.LossBad},
	}
	for _, c := range cases {
		got := d.LossColor(c.pct)
		if got != c.want {
			t.Errorf("LossColor(%.1f): want %q, got %q", c.pct, c.want, got)
		}
	}
}

// ── helper ────────────────────────────────────────────────────────────────────

func writeTempIni(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "wifimon.ini")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeTempIni: %v", err)
	}
	return path
}

// ── exe-dir search ───────────────────────────────────────────────────────────
// We can't control os.Executable() in tests, but we can verify that a file
// next to the test binary (which IS the exe during go test) is found when
// the explicit path and env var are both empty.
func TestLoadFindsExeDir(t *testing.T) {
	// Resolve where the current test binary lives.
	exe, err := os.Executable()
	if err != nil {
		t.Skip("os.Executable not available:", err)
	}
	if real, err2 := filepath.EvalSymlinks(exe); err2 == nil {
		exe = real
	}
	exeDir := filepath.Dir(exe)

	// Write a config right next to the test binary.
	iniPath := filepath.Join(exeDir, "wifimon.ini")
	if err := os.WriteFile(iniPath, []byte("[theme]\ntitle = 77\n"), 0644); err != nil {
		t.Skipf("cannot write to exe dir %s: %v", exeDir, err)
	}
	defer os.Remove(iniPath)

	// Unset env var so it doesn't interfere.
	t.Setenv("WIFIMON_CONFIG", "")

	got := Load("") // no explicit path
	if got.Title != "77" {
		t.Errorf("exe-dir load: want title=77, got %q", got.Title)
	}
}
