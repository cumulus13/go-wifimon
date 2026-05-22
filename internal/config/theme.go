// Package config loads and exposes the wifimon colour theme.
//
// Config file format (INI-style, no external dependencies):
//
//	[theme]
//	title          = 39
//	subtitle       = 241
//	label          = 244
//	value          = 252
//	card_border    = 62
//	tab_active_fg  = 230
//	tab_active_bg  = 31
//	tab_fg         = 252
//	tab_bg         = 238
//	footer         = 241
//	status         = 241
//
//	# state badge colours
//	ok             = 42
//	warn           = 214
//	err            = 196
//
//	# signal thresholds  (value colour by signal %)
//	signal_great   = 42    # >= 85 %
//	signal_good    = 83    # >= 65 %
//	signal_fair    = 214   # >= 45 %
//	signal_poor    = 208   # >= 20 %
//	signal_none    = 196   # < 20 %
//
//	# latency thresholds (ms)
//	latency_great  = 42    # <= 20 ms
//	latency_good   = 83    # <= 60 ms
//	latency_fair   = 214   # <= 120 ms
//	latency_bad    = 196   # > 120 ms  / unreachable
//
//	# packet-loss thresholds (%)
//	loss_great     = 42    # == 0 %
//	loss_fair      = 214   # <= 5 %
//	loss_bad       = 196   # > 5 %
//
// Colour values are xterm-256 palette indices (0-255) or hex (#rrggbb / #rgb).
// Lines starting with # or ; are comments. Blank lines are ignored.
// Keys outside a [theme] section are also accepted for convenience.
// Unrecognised keys are silently ignored.
//
// Config file search order (first found wins):
//  1. Path given via --config flag (handled by caller)
//  2. $WIFIMON_CONFIG environment variable
//  3. <exe-dir>/wifimon.ini  (same folder as the binary)
//  4. ./wifimon.ini           (current working directory)
//  5. ~/.config/wifimon/wifimon.ini
//  6. Built-in defaults (no file required)
package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Theme holds every colour token used by the TUI.
// All values are lipgloss-compatible colour strings:
// xterm-256 index ("42"), hex ("#00ff87"), or ANSI name ("white").
type Theme struct {
	// Chrome
	Title       string
	Subtitle    string
	Label       string
	Value       string
	CardBorder  string
	TabActiveFG string
	TabActiveBG string
	TabFG       string
	TabBG       string
	Footer      string
	Status      string

	// State badges
	Ok   string
	Warn string
	Err  string

	// Signal thresholds (foreground colour per quality tier)
	SignalGreat string // >= 85 %
	SignalGood  string // >= 65 %
	SignalFair  string // >= 45 %
	SignalPoor  string // >= 20 %
	SignalNone  string // <  20 %

	// Latency thresholds (foreground colour per quality tier)
	LatencyGreat string // <= 20 ms
	LatencyGood  string // <= 60 ms
	LatencyFair  string // <= 120 ms
	LatencyBad   string // >  120 ms or unreachable

	// Packet-loss thresholds
	LossGreat string // == 0 %
	LossFair  string // <= 5 %
	LossBad   string // >  5 %
}

// Default returns the built-in theme. All callers should start here and then
// apply any user overrides on top.
func Default() Theme {
	return Theme{
		// Chrome
		Title:       "39",  // bright cyan-blue
		Subtitle:    "241", // mid-grey
		Label:       "244", // light grey
		Value:       "252", // near-white
		CardBorder:  "62",  // slate-blue
		TabActiveFG: "230", // bright yellow-white
		TabActiveBG: "31",  // medium blue
		TabFG:       "252",
		TabBG:       "238", // dark grey
		Footer:      "241",
		Status:      "241",

		// State badges
		Ok:   "42",  // green
		Warn: "214", // orange
		Err:  "196", // red

		// Signal quality gradient  (green → yellow-green → orange → red)
		SignalGreat: "42",
		SignalGood:  "83",
		SignalFair:  "214",
		SignalPoor:  "208",
		SignalNone:  "196",

		// Latency quality gradient
		LatencyGreat: "42",
		LatencyGood:  "83",
		LatencyFair:  "214",
		LatencyBad:   "196",

		// Packet-loss gradient
		LossGreat: "42",
		LossFair:  "214",
		LossBad:   "196",
	}
}

// Load reads the first config file found in the search path and applies any
// [theme] overrides on top of the built-in defaults.
// If no file is found, or a key is missing, the default value is kept.
// Errors (unreadable file, parse issues) are silently ignored so the TUI
// always starts even with a broken config.
func Load(explicit string) Theme {
	t := Default()

	path := findConfig(explicit)
	if path == "" {
		return t
	}

	f, err := os.Open(path)
	if err != nil {
		return t
	}
	defer f.Close()

	inTheme := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip blank lines and comments
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}

		// section header
		if line[0] == '[' {
			section := strings.ToLower(strings.Trim(line, "[]"))
			inTheme = section == "theme"
			continue
		}

		// accept keys both inside and outside a [theme] section
		if !inTheme && strings.Contains(line, "[") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		// Strip inline comment — only when # or ; is preceded by whitespace,
		// so hex colours like "#ff0000" at the start of a value are preserved.
		for _, marker := range []string{" #", "\t#", " ;", "\t;"} {
			if idx := strings.Index(val, marker); idx >= 0 {
				val = strings.TrimSpace(val[:idx])
				break
			}
		}

		if val == "" {
			continue
		}

		applyKey(&t, key, val)
	}

	return t
}

// findConfig returns the first config file path that exists.
// Search order:
//  1. explicit path (from --config flag)
//  2. $WIFIMON_CONFIG env var
//  3. <exe-dir>/wifimon.ini   ← same folder as the binary
//  4. ./wifimon.ini            (current working directory)
//  5. ~/.config/wifimon/wifimon.ini
func findConfig(explicit string) string {
	var candidates []string

	if explicit != "" {
		candidates = append(candidates, explicit)
	}

	if env := os.Getenv("WIFIMON_CONFIG"); env != "" {
		candidates = append(candidates, env)
	}

	// Exe-directory — most natural location for a Windows portable app.
	if exe, err := os.Executable(); err == nil {
		// Follow any symlinks so we get the real binary directory.
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "wifimon.ini"))
	}

	// Current working directory.
	candidates = append(candidates, "wifimon.ini")

	// XDG / home fallback.
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".config", "wifimon", "wifimon.ini"))
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// applyKey maps a config key string to the corresponding Theme field.
func applyKey(t *Theme, key, val string) {
	switch key {
	case "title":
		t.Title = val
	case "subtitle":
		t.Subtitle = val
	case "label":
		t.Label = val
	case "value":
		t.Value = val
	case "card_border":
		t.CardBorder = val
	case "tab_active_fg":
		t.TabActiveFG = val
	case "tab_active_bg":
		t.TabActiveBG = val
	case "tab_fg":
		t.TabFG = val
	case "tab_bg":
		t.TabBG = val
	case "footer":
		t.Footer = val
	case "status":
		t.Status = val
	case "ok":
		t.Ok = val
	case "warn":
		t.Warn = val
	case "err":
		t.Err = val
	case "signal_great":
		t.SignalGreat = val
	case "signal_good":
		t.SignalGood = val
	case "signal_fair":
		t.SignalFair = val
	case "signal_poor":
		t.SignalPoor = val
	case "signal_none":
		t.SignalNone = val
	case "latency_great":
		t.LatencyGreat = val
	case "latency_good":
		t.LatencyGood = val
	case "latency_fair":
		t.LatencyFair = val
	case "latency_bad":
		t.LatencyBad = val
	case "loss_great":
		t.LossGreat = val
	case "loss_fair":
		t.LossFair = val
	case "loss_bad":
		t.LossBad = val
	}
}

// SignalColor returns the theme colour for a given signal percentage.
func (t Theme) SignalColor(pct int) string {
	switch {
	case pct >= 85:
		return t.SignalGreat
	case pct >= 65:
		return t.SignalGood
	case pct >= 45:
		return t.SignalFair
	case pct >= 20:
		return t.SignalPoor
	default:
		return t.SignalNone
	}
}

// LatencyColor returns the theme colour for a given latency in milliseconds.
// Pass -1 (or any negative) to indicate unreachable.
func (t Theme) LatencyColor(ms int, reachable bool) string {
	if !reachable {
		return t.LatencyBad
	}
	switch {
	case ms <= 20:
		return t.LatencyGreat
	case ms <= 60:
		return t.LatencyGood
	case ms <= 120:
		return t.LatencyFair
	default:
		return t.LatencyBad
	}
}

// LossColor returns the theme colour for a given packet-loss percentage.
func (t Theme) LossColor(pct float64) string {
	switch {
	case pct == 0:
		return t.LossGreat
	case pct <= 5:
		return t.LossFair
	default:
		return t.LossBad
	}
}