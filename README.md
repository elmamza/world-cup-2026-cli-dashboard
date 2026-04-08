![screenshot](screenshot.png)

# World Cup 2026 CLI Dashboard

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![MIT License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> ⚽ Dashboard CLI interactif pour suivre la Coupe du Monde 2026 depuis votre terminal.
> Fork du [world-cup-2022-cli-dashboard](https://github.com/cedricblondeau/world-cup-2022-cli-dashboard).

## Features

- ⚽ Matchs en direct (buts, cartons, substitutions)
- 🗒️ Composiciones de equipos
- 📅 Matchs passés et à venir
- 📒 Classements & bracket
- 📊 Stats joueurs (buts, cartons jaunes, cartons rouges)

## Install

### Method 1: Homebrew 🍺

Install:
```bash
brew tap elmamza/elmamza
brew install world-cup-2026-cli-dashboard
```

Run:
```bash
world-cup-2026-cli-dashboard
```

### Method 2: Docker 🐳

Build from the `main` branch:
```bash
docker build --no-cache https://github.com/elmamza/world-cup-2026-cli-dashboard.git#main -t world-cup-2026-cli-dashboard
```

Run it:
```bash
docker run -ti -e TZ=America/Toronto world-cup-2026-cli-dashboard
```

Replace `America/Toronto` with the desired timezone.

### Method 3: Go package

Requirements:
- Go 1.21+ (with `$PATH` properly set up)
- Git

```bash
go install github.com/elmamza/world-cup-2026-cli-dashboard@latest
world-cup-2026-cli-dashboard
```

### Method 4: Pre-compiled binaries

Pre-compiled binaries are available on the [releases page](https://github.com/elmamza/world-cup-2026-cli-dashboard/releases).

## UI

UI is powered by [bubbletea](https://github.com/charmbracelet/bubbletea) and [lipgloss](https://github.com/charmbracelet/lipgloss).

For optimal results, it's recommended to use a terminal with:
- True Color (24-bit) support;
- at least 160 columns and 50 rows.

## LICENSE

MIT