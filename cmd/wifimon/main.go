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
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "WiFiMon watches your Wi-Fi connection in real time.\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  wifimon [flags]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
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

	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		panic(err)
	}
}
