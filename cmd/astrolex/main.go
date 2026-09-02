package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"astrolex/internal/config"
	"astrolex/internal/domain"
	"astrolex/internal/i18n"
	"astrolex/internal/repository"
	iface "astrolex/internal/interface"
)

func main() {
	// 设置随机种子（Go 1.20+ 可省略，但保留兼容）
	rand.Seed(time.Now().UnixNano())

	// ==================== 语言选择 ====================
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("=== Astrolex ===")
	fmt.Println("please choose your language")
	fmt.Println("请选择您的语言")
	fmt.Print("c/e: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	var lang string
	switch input {
	case "c", "chinese", "zh":
		lang = "zh"
	case "e", "english", "en":
		lang = "en"
	default:
		fmt.Println("Invalid choice, default to English")
		lang = "en"
	}

	if err := i18n.Load(lang); err != nil {
		log.Fatal("加载语言文件失败:", err)
	}
	fmt.Println(i18n.T("language_changed"))

	// ==================== 加载配置 ====================
	cfg, err := config.LoadConfig("config/game_config.json")
	if err != nil {
		log.Fatal(i18n.T("load_config_failed"), err)
	}
	initialTime, err := time.Parse(time.RFC3339, cfg.InitialTime)
	if err != nil {
		log.Fatal(i18n.T("parse_time_failed"), err)
	}

	// ==================== 加载静态数据 ====================
	starSys, err := repository.LoadStarCatalog("data/star_catalog.json")
	if err != nil {
		log.Fatal(i18n.T("load_star_catalog_failed"), err)
	}
	parts, err := repository.LoadParts("data/parts_library.json")
	if err != nil {
		log.Fatal(i18n.T("load_parts_failed"), err)
	}
	suppliers, err := repository.LoadSuppliers("data/suppliers.json")
	if err != nil {
		log.Fatal(i18n.T("load_suppliers_failed"), err)
	}
	bases, err := repository.LoadBases("data/bases.json")
	if err != nil {
		log.Fatal(i18n.T("load_bases_failed"), err)
	}

	// ==================== 加载或创建存档 ====================
	game := &domain.Game{}
	savesDir := "saves"
	defaultSave := filepath.Join(savesDir, "slot_default.json")
	if _, err := os.Stat(defaultSave); err == nil {
		data, err := os.ReadFile(defaultSave)
		if err != nil {
			log.Fatal(i18n.T("read_save_failed"), err)
		}
		if err := json.Unmarshal(data, game); err != nil {
			log.Fatal(i18n.T("parse_save_failed"), err)
		}
		fmt.Println(i18n.T("loaded_default_save"))
	} else {
		game = &domain.Game{
			Version:     "0.2.0",
			CurrentTime: initialTime,
			Player: domain.Player{
				Credits: cfg.InitialCredits,
				Reputation: domain.Reputation{
					Safety:  cfg.Reputation.Safety,
					Speed:   cfg.Reputation.Speed,
					Politic: make(map[string]int),
				},
			},
			StarSystem:        starSys,
			PartsDB:           parts,
			SuppliersDB:       suppliers,
			BasesDB:           bases,
			Contracts:         []domain.Contract{},
			Launches:          []domain.LaunchMission{},
			LogBook:           []domain.LogEntry{},
			Config:            domain.GameConfig{EnableSandbox: false, EnableECCL: false},
			RocketDesigns:     []domain.RocketDesign{},
			SatelliteDesigns:  []domain.SatelliteDesign{},
			Vessels:           []domain.Vessel{},
			AssemblyProjects:  []domain.AssemblyProject{},
		}
		fmt.Println(i18n.T("new_game_created"))
	}

	// ==================== 启动 REPL ====================
	repl := iface.NewRepl(game, cfg)
	repl.Run()
}
