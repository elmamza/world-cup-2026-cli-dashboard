package main

import (
	"fmt"
	"os"

	"github.com/cedricblondeau/world-cup-2022-cli-dashboard/data/openfootball"
	"github.com/cedricblondeau/world-cup-2022-cli-dashboard/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	ofClient := openfootball.NewClient("2026--usa")
	dashboard := ui.NewDashboard(ofClient)
	p := tea.NewProgram(dashboard)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Oh no, there's been an error: %v", err)
		os.Exit(1)
	}
}
