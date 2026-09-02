package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"astrolex/internal/config"
	"astrolex/internal/domain"
	"astrolex/internal/repository"
	iface "astrolex/internal/interface"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// 加载配置
	cfg, err := config.LoadConfig("config/game_config.json")
	if err != nil {
		log.Fatal("加载配置失败:", err)
	}
	initialTime, err := time.Parse(time.RFC3339, cfg.InitialTime)
	if err != nil {
		log.Fatal("解析初始时间失败:", err)
	}

	// 加载静态数据
	starSys, err := repository.LoadStarCatalog("data/star_catalog.json")
	if err != nil {
		log.Fatal("加载星表失败:", err)
	}
	parts, err := repository.LoadParts("data/parts_library.json")
	if err != nil {
		log.Fatal("加载零件失败:", err)
	}
	suppliers, err := repository.LoadSuppliers("data/suppliers.json")
	if err != nil {
		log.Fatal("加载供应商失败:", err)
	}
	bases, err := repository.LoadBases("data/bases.json")
	if err != nil {
		log.Fatal("加载基地失败:", err)
	}

	// 尝试加载默认存档
	game := &domain.Game{}
	savesDir := "saves"
	defaultSave := filepath.Join(savesDir, "slot_default.json")
	if _, err := os.Stat(defaultSave); err == nil {
		data, err := os.ReadFile(defaultSave)
		if err != nil {
			log.Fatal("读取存档失败:", err)
		}
		if err := json.Unmarshal(data, game); err != nil {
			log.Fatal("解析存档失败:", err)
		}
		fmt.Println("已加载默认存档。")
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
			Vessels:           []domain.Vessel{}, // 修正：替换 ActiveSatellites
		}
		fmt.Println("新建游戏。")
	}

	// 启动 REPL
	repl := iface.NewRepl(game, cfg)
	repl.Run()
}
