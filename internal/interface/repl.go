package repl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"astrolex/internal/config"
	"astrolex/internal/domain"
	"astrolex/internal/i18n"
)

type Repl struct {
	game            *domain.Game
	cfg             *config.GameConfig
	reader          *bufio.Reader
	activeContractID string
	pendingPrograms map[string]*domain.ECCLProgram
}

func NewRepl(g *domain.Game, cfg *config.GameConfig) *Repl {
	return &Repl{
		game:            g,
		cfg:             cfg,
		reader:          bufio.NewReader(os.Stdin),
		activeContractID: "",
		pendingPrograms: make(map[string]*domain.ECCLProgram),
	}
}

func (r *Repl) Run() {
	fmt.Println(i18n.T("welcome"))
	for {
		fmt.Print("> ")
		input, _ := r.reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			break
		}
		r.handleCommand(input)
	}
}

func (r *Repl) handleCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}
	cmd := parts[0]
	switch cmd {
	case "help":
		fmt.Println(i18n.T("help_available"))
		fmt.Println(i18n.T("help_design"))
		fmt.Println(i18n.T("help_edit"))
		fmt.Println(i18n.T("help_list"))
		fmt.Println(i18n.T("help_sat"))
		fmt.Println(i18n.T("help_orbit"))
		fmt.Println(i18n.T("help_orbit_flyby_plan"))
		fmt.Println(i18n.T("help_orbit_flyby_execute"))
		fmt.Println(i18n.T("help_launch"))
		fmt.Println(i18n.T("help_contract"))
		fmt.Println(i18n.T("help_window"))
		fmt.Println(i18n.T("help_date"))
		fmt.Println(i18n.T("help_cargo"))
		fmt.Println(i18n.T("help_cargo_load"))
		fmt.Println(i18n.T("help_program"))
		fmt.Println(i18n.T("help_program_list"))
		fmt.Println(i18n.T("help_program_edit"))
		fmt.Println(i18n.T("help_program_upload"))
		fmt.Println(i18n.T("help_program_run"))
		fmt.Println(i18n.T("help_program_stop"))
		fmt.Println(i18n.T("help_program_logs"))
		fmt.Println(i18n.T("help_program_status"))
		fmt.Println(i18n.T("help_assembly"))
		fmt.Println(i18n.T("help_assembly_create"))
		fmt.Println(i18n.T("help_assembly_add"))
		fmt.Println(i18n.T("help_assembly_list"))
		fmt.Println(i18n.T("help_assembly_status"))
		fmt.Println(i18n.T("help_assembly_start"))
		fmt.Println(i18n.T("help_assembly_step"))
		fmt.Println(i18n.T("help_assembly_pause"))
		fmt.Println(i18n.T("help_assembly_resume"))
		fmt.Println(i18n.T("help_assembly_delete"))
		fmt.Println(i18n.T("help_assembly_program"))
		fmt.Println(i18n.T("help_save"))
		fmt.Println(i18n.T("help_load"))
		fmt.Println(i18n.T("help_list_saves"))
		fmt.Println(i18n.T("help_exit"))

	case "list":
		if len(parts) < 2 {
			fmt.Println(i18n.T("usage_list"))
			return
		}
		sub := parts[1]
		switch sub {
		case "parts":
			r.listParts()
		case "designs":
			r.listDesigns()
		case "satellites":
			r.listSatellites()
		case "saves":
			r.listSaves()
		default:
			fmt.Printf(i18n.T("unknown_list_type")+"\n", sub)
		}

	case "simulate":
		r.simulateDemo()

	case "design":
		if len(parts) >= 2 && parts[1] == "satellite" {
			r.designSatellite()
		} else if len(parts) >= 2 && parts[1] == "list" {
			r.listDesigns()
		} else {
			r.designRocket()
		}

	case "edit":
		if len(parts) < 3 {
			fmt.Println(i18n.T("invalid_usage") + " edit <design|satellite> <ID>")
			return
		}
		editType := parts[1]
		editID := parts[2]
		if editType == "design" {
			r.editDesign(editID)
		} else if editType == "satellite" {
			r.editSatellite(editID)
		} else {
			fmt.Printf(i18n.T("unknown_command")+"\n", editType)
		}

	case "contract":
		r.generateContract()

	case "accept":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " accept <合约ID>")
			return
		}
		r.acceptContract(parts[1])

	case "launch":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " launch <设计ID> [目标天体ID] [轨道层]")
			fmt.Println(i18n.T("help_launch"))
			return
		}
		designID := parts[1]
		targetID := ""
		orbitLayer := ""
		if len(parts) >= 3 {
			targetID = parts[2]
		}
		if len(parts) >= 4 {
			orbitLayer = parts[3]
		}
		r.launchRocket(designID, targetID, orbitLayer)

	case "done":
		r.doneContract()

	case "budget":
		r.showBudget()

	case "window":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " window <目标天体ID>")
			return
		}
		r.windowCommand(parts[1])

	case "date":
		r.showDate()

	case "tick":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " tick <天数>")
			return
		}
		days, err := strconv.Atoi(parts[1])
		if err != nil || days <= 0 {
			fmt.Println(i18n.T("tick_invalid_days"))
			return
		}
		r.advanceTime(days)

	case "sat":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " sat <子命令> [参数]")
			fmt.Println(i18n.T("help_sat"))
			return
		}
		subCmd := parts[1]
		switch subCmd {
		case "list":
			r.vesselList()
		case "status":
			if len(parts) < 3 {
				fmt.Println(i18n.T("invalid_usage") + " sat status <航天器ID>")
				return
			}
			r.vesselStatus(parts[2])
		case "measure":
			if len(parts) < 3 {
				fmt.Println(i18n.T("invalid_usage") + " sat measure <航天器ID>")
				return
			}
			r.vesselMeasure(parts[2])
		case "send":
			if len(parts) < 3 {
				fmt.Println(i18n.T("invalid_usage") + " sat send <航天器ID>")
				return
			}
			r.vesselSend(parts[2])
		case "point":
			if len(parts) < 4 {
				fmt.Println(i18n.T("invalid_usage") + " sat point <航天器ID> <目标>")
				return
			}
			r.vesselPoint(parts[2], parts[3])
		default:
			fmt.Printf(i18n.T("unknown_command")+"\n", subCmd)
		}

	case "orbit":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " orbit <子命令> [参数]")
			fmt.Println(i18n.T("help_orbit"))
			return
		}
		r.handleOrbitCommand(parts[1], parts[2:])

	case "cargo":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " cargo <子命令> [参数]")
			fmt.Println(i18n.T("help_cargo"))
			return
		}
		subCmd := parts[1]
		switch subCmd {
		case "info":
			if len(parts) < 4 {
				fmt.Println(i18n.T("invalid_usage") + " cargo info <rocket|satellite> <ID>")
				return
			}
			r.cargoInfoCommand(parts[2], parts[3])
		case "load":
			if len(parts) < 6 {
				fmt.Println(i18n.T("invalid_usage") + " cargo load <火箭设计ID> <货舱序号> <货物类型> <货物ID>")
				return
			}
			r.cargoLoadCommand(parts[2], parts[3], parts[4], parts[5])
		default:
			fmt.Printf(i18n.T("unknown_command")+"\n", subCmd)
		}

	case "program":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " program <子命令> [参数]")
			fmt.Println(i18n.T("help_program"))
			return
		}
		r.handleProgramCommand(parts[1], parts[2:])

	case "assembly":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " assembly <子命令> [参数]")
			fmt.Println(i18n.T("help_assembly"))
			return
		}
		r.handleAssemblyCommand(parts[1], parts[2:])

	case "save":
		slot := "default"
		if len(parts) > 1 {
			slot = parts[1]
		}
		r.saveGame(slot)

	case "load":
		if len(parts) < 2 {
			fmt.Println(i18n.T("invalid_usage") + " load <槽位名>")
			return
		}
		if err := r.loadGame(parts[1]); err != nil {
			fmt.Printf(i18n.T("load_failed")+"\n", err)
		}

	case "exit", "quit":
		fmt.Println(i18n.T("exit"))
		os.Exit(0)

	default:
		fmt.Println(i18n.T("unknown_command"))
	}
}

// ==================== 存档系统 ====================

func (r *Repl) saveGame(slot string) {
	if slot == "" {
		slot = "default"
	}
	savesDir := "saves"
	if err := os.MkdirAll(savesDir, 0755); err != nil {
		fmt.Printf(i18n.T("save_failed")+"\n", err)
		return
	}
	filename := filepath.Join(savesDir, fmt.Sprintf("slot_%s.json", slot))
	data, err := json.MarshalIndent(r.game, "", "  ")
	if err != nil {
		fmt.Printf(i18n.T("save_failed")+"\n", err)
		return
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf(i18n.T("save_failed")+"\n", err)
		return
	}
	fmt.Printf(i18n.T("save_success")+"\n", slot)
}

func (r *Repl) loadGame(slot string) error {
	if slot == "" {
		slot = "default"
	}
	filename := filepath.Join("saves", fmt.Sprintf("slot_%s.json", slot))
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	var newGame domain.Game
	if err := json.Unmarshal(data, &newGame); err != nil {
		return err
	}
	*r.game = newGame
	if r.activeContractID != "" {
		found := false
		for _, c := range r.game.Contracts {
			if c.ID == r.activeContractID && c.Status == "Accepted" {
				found = true
				break
			}
		}
		if !found {
			r.activeContractID = ""
		}
	}
	fmt.Printf(i18n.T("load_success")+"\n", slot)
	return nil
}

func (r *Repl) listSaves() {
	savesDir := "saves"
	files, err := os.ReadDir(savesDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(i18n.T("no_saves"))
		} else {
			fmt.Printf(i18n.T("read_saves_failed")+"\n", err)
		}
		return
	}
	fmt.Println(i18n.T("available_save_slots"))
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "slot_") && strings.HasSuffix(f.Name(), ".json") {
			slot := strings.TrimPrefix(f.Name(), "slot_")
			slot = strings.TrimSuffix(slot, ".json")
			fmt.Printf("  %s\n", slot)
		}
	}
}
