package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cumulus13/go-wifimon/internal/tui"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	configPath := flag.String("config", "", "path to wifimon.ini config file")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "WiFiMon watches your Wi-Fi connection in real time.\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  wifimon [flags]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nConfig file search order (first found wins):\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  1. --config <path>\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  2. $WIFIMON_CONFIG env var\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  3. <exe-dir>/wifimon.ini  (same folder as the binary)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  4. ./wifimon.ini           (current working directory)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  5. ~/.config/wifimon/wifimon.ini\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  6. Built-in defaults\n")
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}
	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(2)
	}

	p := tea.NewProgram(tui.NewModelWithConfig(*configPath), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		panic(err)
	}
}