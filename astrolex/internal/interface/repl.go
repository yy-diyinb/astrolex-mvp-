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
)

type Repl struct {
	game            *domain.Game
	cfg             *config.GameConfig
	reader          *bufio.Reader
	activeContractID string
	pendingPrograms map[string]*domain.ECCLProgram // 待上传的ECCL程序
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
	fmt.Println("Astrolex MVP - 输入 help 查看命令")
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
		fmt.Println("可用命令: design, edit, simulate, list, contract, accept, launch, done, budget, window, date, tick, sat, orbit, cargo, program, assembly, save, load, exit")
		fmt.Println("  design [satellite]  - 设计火箭或卫星")
		fmt.Println("  edit <design|satellite> <ID> - 编辑已保存的设计")
		fmt.Println("  list [parts|designs|satellites|saves] - 列出零件、火箭设计、卫星设计或存档")
		fmt.Println("  sat [list|status|measure|send|point] - 在轨航天器控制")
		fmt.Println("  orbit [info|transfer|escape|dock|travel|release|flyby] - 航天器轨道操作")
		fmt.Println("    orbit flyby plan <航天器ID>          - 推荐下一个飞掠目标")
		fmt.Println("    orbit flyby execute <航天器ID> <行星ID> [高度] - 执行飞掠")
		fmt.Println("  launch <设计ID> [目标天体ID] [轨道层] - 执行发射（轨道层: low/medium/high）")
		fmt.Println("  contract / accept / done / budget - 合约系统")
		fmt.Println("  window <目标天体ID> - 查看发射窗口")
		fmt.Println("  date / tick <天数> - 时间管理")
		fmt.Println("  cargo info <rocket|satellite> <ID> - 查看货舱装载信息")
		fmt.Println("  cargo load <火箭设计ID> <货舱序号> <货物类型> <货物ID> - 装载货物到火箭货舱")
		fmt.Println("  program list                    - 列出所有程序")
		fmt.Println("  program edit <名称>             - 创建或编辑程序")
		fmt.Println("  program upload <程序ID> <航天器ID> - 上传程序到航天器")
		fmt.Println("  program run <程序ID> <航天器ID>   - 在航天器上运行程序")
		fmt.Println("  program stop <程序ID> <航天器ID>  - 停止运行中的程序")
		fmt.Println("  program logs <程序ID> <航天器ID>  - 查看程序日志")
		fmt.Println("  program status <程序ID> <航天器ID> - 查看程序状态")
		fmt.Println("  assembly create <名称>      - 创建组装项目")
		fmt.Println("  assembly add <项目ID> <模块ID> <数量> - 添加步骤")
		fmt.Println("  assembly list              - 列出所有项目")
		fmt.Println("  assembly status <项目ID>   - 查看项目状态")
		fmt.Println("  assembly start <项目ID>    - 开始组装")
		fmt.Println("  assembly step <项目ID>     - 推进到下一步")
		fmt.Println("  assembly pause <项目ID>    - 暂停组装")
		fmt.Println("  assembly resume <项目ID>   - 恢复组装")
		fmt.Println("  assembly delete <项目ID>   - 删除项目")
		fmt.Println("  assembly program <项目ID> <程序ID> - 关联ECCL程序")
		fmt.Println("  save [槽位名] - 保存游戏（默认 default）")
		fmt.Println("  load <槽位名> - 加载存档")
		fmt.Println("  list saves - 列出所有存档槽位")

	case "list":
		if len(parts) < 2 {
			fmt.Println("用法: list [parts|designs|satellites|saves]")
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
			fmt.Printf("未知列表类型 '%s'\n", sub)
		}

	case "simulate":
		r.simulateDemo()

	case "design":
		if len(parts) >= 2 && parts[1] == "satellite" {
			r.designSatellite()
		} else {
			r.designRocket()
		}

	case "edit":
		if len(parts) < 3 {
			fmt.Println("用法: edit <design|satellite> <ID>")
			return
		}
		editType := parts[1]
		editID := parts[2]
		if editType == "design" {
			r.editDesign(editID)
		} else if editType == "satellite" {
			r.editSatellite(editID)
		} else {
			fmt.Printf("未知编辑类型 '%s'\n", editType)
		}

	case "contract":
		r.generateContract()

	case "accept":
		if len(parts) < 2 {
			fmt.Println("用法: accept <合约ID>")
			return
		}
		r.acceptContract(parts[1])

	case "launch":
		if len(parts) < 2 {
			fmt.Println("用法: launch <设计ID> [目标天体ID] [轨道层]")
			fmt.Println("  轨道层: low, medium, high (或 0,1,2)，默认为 high")
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
			fmt.Println("用法: window <目标天体ID>")
			return
		}
		r.windowCommand(parts[1])

	case "date":
		r.showDate()

	case "tick":
		if len(parts) < 2 {
			fmt.Println("用法: tick <天数>")
			return
		}
		days, err := strconv.Atoi(parts[1])
		if err != nil || days <= 0 {
			fmt.Println("请输入正整数的天数")
			return
		}
		r.advanceTime(days)

	case "sat":
		if len(parts) < 2 {
			fmt.Println("用法: sat <子命令> [参数]")
			fmt.Println("  子命令: list, status <航天器ID>, measure <航天器ID>, send <航天器ID>, point <航天器ID> <目标>")
			return
		}
		subCmd := parts[1]
		switch subCmd {
		case "list":
			r.vesselList()
		case "status":
			if len(parts) < 3 {
				fmt.Println("用法: sat status <航天器ID>")
				return
			}
			r.vesselStatus(parts[2])
		case "measure":
			if len(parts) < 3 {
				fmt.Println("用法: sat measure <航天器ID>")
				return
			}
			r.vesselMeasure(parts[2])
		case "send":
			if len(parts) < 3 {
				fmt.Println("用法: sat send <航天器ID>")
				return
			}
			r.vesselSend(parts[2])
		case "point":
			if len(parts) < 4 {
				fmt.Println("用法: sat point <航天器ID> <目标>")
				return
			}
			r.vesselPoint(parts[2], parts[3])
		default:
			fmt.Printf("未知卫星子命令 '%s'\n", subCmd)
		}

	case "orbit":
		if len(parts) < 2 {
			fmt.Println("用法: orbit <子命令> [参数]")
			fmt.Println("  子命令: info <航天器ID>, transfer <航天器ID> <层>, escape <航天器ID>, dock <主航天器ID> <从航天器ID>, travel <航天器ID> <目标天体ID>, release <航天器ID> <货舱序号> <货物索引>, flyby plan <航天器ID>, flyby execute <航天器ID> <行星ID> [高度]")
			return
		}
		r.handleOrbitCommand(parts[1], parts[2:])

	case "cargo":
		if len(parts) < 2 {
			fmt.Println("用法: cargo <子命令> [参数]")
			fmt.Println("  子命令: info <rocket|satellite> <ID> - 查看货舱信息")
			fmt.Println("          load <火箭设计ID> <货舱序号> <货物类型> <货物ID> - 装载货物")
			fmt.Println("          货物类型: rocket, satellite, part")
			return
		}
		subCmd := parts[1]
		switch subCmd {
		case "info":
			if len(parts) < 4 {
				fmt.Println("用法: cargo info <rocket|satellite> <ID>")
				return
			}
			r.cargoInfoCommand(parts[2], parts[3])
		case "load":
			if len(parts) < 6 {
				fmt.Println("用法: cargo load <火箭设计ID> <货舱序号> <货物类型> <货物ID>")
				return
			}
			r.cargoLoadCommand(parts[2], parts[3], parts[4], parts[5])
		default:
			fmt.Printf("未知 cargo 子命令 '%s'\n", subCmd)
		}

	case "program":
		if len(parts) < 2 {
			fmt.Println("用法: program <子命令> [参数]")
			fmt.Println("  program list                    - 列出所有程序")
			fmt.Println("  program edit <名称>             - 创建或编辑程序")
			fmt.Println("  program upload <程序ID> <航天器ID> - 上传程序到航天器")
			fmt.Println("  program run <程序ID> <航天器ID>   - 在航天器上运行程序")
			fmt.Println("  program stop <程序ID> <航天器ID>  - 停止运行中的程序")
			fmt.Println("  program logs <程序ID> <航天器ID>  - 查看程序日志")
			fmt.Println("  program status <程序ID> <航天器ID> - 查看程序状态")
			return
		}
		r.handleProgramCommand(parts[1], parts[2:])

	case "assembly":
		if len(parts) < 2 {
			fmt.Println("用法: assembly <子命令> [参数]")
			fmt.Println("  create <名称>           - 创建组装项目")
			fmt.Println("  add <项目ID> <模块ID> <数量> - 添加步骤")
			fmt.Println("  list                   - 列出所有项目")
			fmt.Println("  status <项目ID>        - 查看项目状态")
			fmt.Println("  start <项目ID>         - 开始组装")
			fmt.Println("  step <项目ID>          - 推进到下一步")
			fmt.Println("  pause <项目ID>         - 暂停组装")
			fmt.Println("  resume <项目ID>        - 恢复组装")
			fmt.Println("  delete <项目ID>        - 删除项目")
			fmt.Println("  program <项目ID> <程序ID> - 关联ECCL程序")
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
			fmt.Println("用法: load <槽位名>")
			return
		}
		if err := r.loadGame(parts[1]); err != nil {
			fmt.Printf("加载存档失败: %v\n", err)
		}

	case "exit", "quit":
		fmt.Println("退出游戏。")
		os.Exit(0)

	default:
		fmt.Println("未知命令")
	}
}

// ==================== 存档系统 ====================

func (r *Repl) saveGame(slot string) {
	if slot == "" {
		slot = "default"
	}
	savesDir := "saves"
	if err := os.MkdirAll(savesDir, 0755); err != nil {
		fmt.Printf("创建存档目录失败: %v\n", err)
		return
	}
	filename := filepath.Join(savesDir, fmt.Sprintf("slot_%s.json", slot))
	data, err := json.MarshalIndent(r.game, "", "  ")
	if err != nil {
		fmt.Printf("序列化存档失败: %v\n", err)
		return
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("写入存档失败: %v\n", err)
		return
	}
	fmt.Printf("游戏已保存到槽位 '%s'\n", slot)
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
	fmt.Printf("游戏已从槽位 '%s' 加载\n", slot)
	return nil
}

func (r *Repl) listSaves() {
	savesDir := "saves"
	files, err := os.ReadDir(savesDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("没有存档。")
		} else {
			fmt.Printf("读取存档目录失败: %v\n", err)
		}
		return
	}
	fmt.Println("可用的存档槽位:")
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "slot_") && strings.HasSuffix(f.Name(), ".json") {
			slot := strings.TrimPrefix(f.Name(), "slot_")
			slot = strings.TrimSuffix(slot, ".json")
			fmt.Printf("  %s\n", slot)
		}
	}
}
