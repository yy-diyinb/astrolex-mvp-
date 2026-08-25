🚀 Astrolex
https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go
https://img.shields.io/badge/License-MIT-yellow.svg
https://img.shields.io/github/actions/workflow/status/yy-diyinb/astrolex-mvp-/go.yml?branch=main
Hardcore space engineering simulator · Terminal MUD
Astrolex puts you in the cockpit of your own space agency. Design rockets and satellites, calculate orbital transfers, automate missions with ECCL scripting, and manage complex assembly projects—all from the Linux terminal.

✨ Features

    🔧 Rocket & Satellite Design – Modular assembly with engines, fuel tanks, avionics, cargo bays, and scientific instruments. Real‑time Δv and acceleration computation.

    🌍 Realistic Orbital Mechanics – Hohmann transfers, launch windows, gravity assists, and multi‑layer orbit transitions (LEO/MEO/GEO). Track your vessels in heliocentric space.

    🛰️ In‑Orbit Operations – Dock any two vessels, merge resources, release cargo, collect and transmit science data.

    📟 ECCL Programming – Built‑in assembly‑like language with 15+ instructions (SET, WAIT, IF, CALL, MEASURE, FIRE, ON EVENT…). Upload, run, and debug automation scripts on active spacecraft.

    🧩 Assembly Project Management – Plan multi‑step constructions (space stations, interplanetary arks). Each step can represent a separate launch & docking, with progress tracking and ECCL integration.

    💰 Economy & Contracts – Accept missions, manage budgets and overruns, earn credits, and face penalties. Hardware, fuel, and pad fees are deducted per launch.

    💾 Save/Load – JSON‑based multi‑slot persistence.

    🖥️ ASCII Terminal UI – Static rocket renderings with telemetry panels and a scrolling event log for full immersion.

🚀 Quick Start
Prerequisites

  Go 1.20 or later

  Linux / macOS / WSL2

Clone & Run
bash

git clone https://github.com/yy-diyinb/astrolex-mvp-.git
cd astrolex-mvp-/astrolex
go mod download
go run cmd/astrolex/main.go

When the game starts, type help to see all available commands.
📸 Demo

https://via.placeholder.com/800x400?text=ASCII+Terminal+Demo
(Replace with an asciinema recording or a screenshot)
Example session (terminal):
text

> design satellite
> sol_panel_large camera antenna done
> design
> faring_6m av-1 cargo_bay_small t-4000 kr-99 t-4000 kr-99 done
> cargo load design_1 1 satellite sat_1
> launch design_1 earth 2
> orbit release main_1 1 1
> sat measure cargo_1
> sat send cargo_1

📖 Command Overview
Category	Commands
Design	design, edit, list parts/designs/satellites
Launch	launch, window, tick, date
Orbit	orbit info/transfer/escape/dock/travel/release/flyby
Satellite	sat list/status/measure/send/point
Cargo	cargo info/load
Programming	program edit/list/upload/run/stop/logs/status
Assembly	assembly create/add/list/status/start/step/pause/resume/delete/program
Economy	contract, accept, done, budget
System	save, load, list saves, exit
📁 Project Structure
text

astrolex/
├── cmd/astrolex/           # Main entry point
├── internal/
│   ├── config/             # Configuration loading
│   ├── domain/             # Core data models
│   ├── engine/             # Physics, ECCL VM, orbital mechanics
│   ├── interface/          # REPL, commands, ASCII renderer
│   └── repository/         # Data persistence (JSON)
├── data/                   # Static data: star catalog, parts, satellite modules
├── config/                 # Game configuration
├── saves/                  # Player save slots (ignored by Git)
├── docs/                   # Design documents
├── go.mod
├── go.sum
└── README.md

🧪 Development
Build
bash

go build -o astrolex cmd/astrolex/main.go

Run tests
bash

go test ./...

Lint (if you have golangci-lint installed)
bash

golangci-lint run

🤝 Contributing

Contributions, issues, and feature requests are welcome!
Please check the CONTRIBUTING.md for guidelines.

  Fork the repository

  Create your feature branch (git checkout -b feature/AmazingFeature)

  Commit your changes (git commit -m 'Add some AmazingFeature')

  Push to the branch (git push origin feature/AmazingFeature)

  Open a Pull Request

📄 License

Distributed under the MIT License. See LICENSE for more information.
🗺️ Roadmap

  Sandbox expedition mode (generational ships, life support)
  
  Supplier reputation & political dynamics
  
  In‑game encyclopedia (dynamic manual)
  
  ECCL full event system with interrupts
  
  Multi‑base launch sites (Moon, Mars, etc.)

🙏 Acknowledgements

  Inspired by Kerbal Space Program, Dwarf Fortress, and classic MUDs.

  Built with Go and a love for text interfaces.

Made with ❤️ for terminal lovers.
