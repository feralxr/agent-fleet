package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/feralxr/agent-fleet/internal/config"
	"github.com/feralxr/agent-fleet/internal/ui"
)

// version is set via -ldflags at build time.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	initConfig := flag.Bool("init-config", false, "write example fleet.toml to ~/.config/agent-fleet/")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if *initConfig {
		dir, err := config.ConfigDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config dir: %v\n", err)
			os.Exit(1)
		}
		path := filepath.Join(dir, "fleet.toml")
		if err := config.WriteExampleFleetToml(path); err != nil {
			fmt.Fprintf(os.Stderr, "init-config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
		os.Exit(0)
	}

	m := ui.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
