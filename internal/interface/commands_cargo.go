package repl

import (
	"fmt"
	"strconv"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
)

// ==================== 货舱信息查询 ====================

func (r *Repl) cargoInfoCommand(designType string, designID string) {
	switch designType {
	case "rocket":
		var design *domain.RocketDesign
		for i := range r.game.RocketDesigns {
			if r.game.RocketDesigns[i].ID == designID {
				design = &r.game.RocketDesigns[i]
				break
			}
		}
		if design == nil {
			fmt.Printf("错误: 未找到火箭设计 '%s'\n", designID)
			return
		}
		if len(design.CargoBays) == 0 {
			fmt.Printf("火箭设计 '%s' 没有装载任何货物。\n", design.Name)
			return
		}
		fmt.Printf("火箭设计 '%s' 货舱装载情况:\n", design.Name)
		for _, bay := range design.CargoBays {
			fmt.Printf("  货舱 %d:\n", bay.Index)
			if len(bay.Loaded) == 0 {
				fmt.Printf("    (空)\n")
			} else {
				for i, item := range bay.Loaded {
					name := r.getCargoName(item)
					mass := r.getCargoMass(item)
					fmt.Printf("    [%d] %s (%s, 质量: %.0f kg)\n", i+1, name, item.Type, mass)
				}
			}
		}
	case "satellite":
		var vessel *domain.Vessel
		for i := range r.game.Vessels {
			if r.game.Vessels[i].ID == designID {
				vessel = &r.game.Vessels[i]
				break
			}
		}
		if vessel == nil {
			fmt.Printf("错误: 未找到在轨航天器 '%s'\n", designID)
			return
		}
		if len(vessel.CargoBays) == 0 {
			fmt.Printf("航天器 '%s' 没有货舱或没有装载货物。\n", vessel.Name)
			return
		}
		fmt.Printf("航天器 '%s' 货舱装载情况:\n", vessel.Name)
		for _, bay := range vessel.CargoBays {
			fmt.Printf("  货舱 %d:\n", bay.Index)
			if len(bay.Loaded) == 0 {
				fmt.Printf("    (空)\n")
			} else {
				for i, item := range bay.Loaded {
					name := r.getCargoName(item)
					mass := r.getCargoMass(item)
					fmt.Printf("    [%d] %s (%s, 质量: %.0f kg)\n", i+1, name, item.Type, mass)
				}
			}
		}
	default:
		fmt.Printf("错误: 不支持的查看类型 '%s'，可用: rocket, satellite\n", designType)
	}
}

// ==================== 装载货物 ====================

func (r *Repl) cargoLoadCommand(designID string, bayIndexStr string, cargoType string, cargoID string) {
	// 解析货舱序号
	bayIndex, err := strconv.Atoi(bayIndexStr)
	if err != nil || bayIndex < 1 {
		fmt.Println("错误: 无效的货舱序号，请输入正整数")
		return
	}

	// 查找火箭设计（使用索引，避免值拷贝）
	var design *domain.RocketDesign
	for i := range r.game.RocketDesigns {
		if r.game.RocketDesigns[i].ID == designID {
			design = &r.game.RocketDesigns[i]
			break
		}
	}
	if design == nil {
		fmt.Printf("错误: 未找到火箭设计 '%s'\n", designID)
		return
	}

	// 验证货舱序号是否存在
	var cargoModIdx int = -1
	cargoCount := 0
	for i, mod := range design.Modules {
		if mod.Type == domain.ModuleCargoBay {
			cargoCount++
			if cargoCount == bayIndex {
				cargoModIdx = i
				break
			}
		}
	}
	if cargoModIdx == -1 {
		fmt.Printf("错误: 火箭设计中未找到货舱 %d\n", bayIndex)
		var bays []int
		for i, mod := range design.Modules {
			if mod.Type == domain.ModuleCargoBay {
				bays = append(bays, i)
			}
		}
		if len(bays) > 0 {
			fmt.Printf("可用货舱序号: ")
			for i := range bays {
				fmt.Printf("%d ", i+1)
			}
			fmt.Println()
		} else {
			fmt.Println("火箭设计中没有货舱零件。")
		}
		return
	}

	// 获取货舱零件的容量
	partID := design.Modules[cargoModIdx].PartID
	part, ok := r.game.PartsDB[partID]
	if !ok {
		fmt.Printf("错误: 货舱零件 '%s' 不存在\n", partID)
		return
	}
	if part.CargoMassCapacity <= 0 {
		fmt.Println("错误: 货舱容量为0，无法装载货物")
		return
	}

	// 获取货物信息
	var cargoMass float64
	var cargoName string
	switch cargoType {
	case "rocket":
		var subDesign *domain.RocketDesign
		for i := range r.game.RocketDesigns {
			if r.game.RocketDesigns[i].ID == cargoID {
				subDesign = &r.game.RocketDesigns[i]
				break
			}
		}
		if subDesign == nil {
			fmt.Printf("错误: 未找到火箭设计 '%s'\n", cargoID)
			return
		}
		cargoMass = subDesign.PayloadMass
		cargoName = subDesign.Name + " (火箭)"
	case "satellite":
		var subDesign *domain.SatelliteDesign
		for i := range r.game.SatelliteDesigns {
			if r.game.SatelliteDesigns[i].ID == cargoID {
				subDesign = &r.game.SatelliteDesigns[i]
				break
			}
		}
		if subDesign == nil {
			fmt.Printf("错误: 未找到卫星设计 '%s'\n", cargoID)
			return
		}
		cargoMass = subDesign.TotalMass
		cargoName = subDesign.Name + " (卫星)"
	case "part":
		part2, ok := r.game.PartsDB[cargoID]
		if !ok {
			fmt.Printf("错误: 未找到零件 '%s'\n", cargoID)
			return
		}
		cargoMass = part2.MassDry
		cargoName = part2.Name
	default:
		fmt.Printf("错误: 不支持的货物类型 '%s'，可用: rocket, satellite, part\n", cargoType)
		return
	}

	if cargoMass <= 0 {
		fmt.Println("警告: 货物质量为0，仍可装载")
	}

	// 查找或创建货舱装载记录
	var bay *domain.CargoBay
	for i := range design.CargoBays {
		if design.CargoBays[i].Index == bayIndex {
			bay = &design.CargoBays[i]
			break
		}
	}
	if bay == nil {
		bay = &domain.CargoBay{
			Index:  bayIndex,
			Loaded: []domain.CargoItem{},
		}
		design.CargoBays = append(design.CargoBays, *bay)
		for i := range design.CargoBays {
			if design.CargoBays[i].Index == bayIndex {
				bay = &design.CargoBays[i]
				break
			}
		}
	}

	// 计算已用质量
	usedMass := 0.0
	for _, item := range bay.Loaded {
		usedMass += r.getCargoMass(item)
	}

	if usedMass+cargoMass > part.CargoMassCapacity {
		fmt.Printf("❌ 装载失败！货舱容量不足。\n")
		fmt.Printf("   货舱容量: %.0f kg\n", part.CargoMassCapacity)
		fmt.Printf("   已使用: %.0f kg\n", usedMass)
		fmt.Printf("   本次货物质量: %.0f kg\n", cargoMass)
		fmt.Printf("   超出: %.0f kg\n", usedMass+cargoMass-part.CargoMassCapacity)
		return
	}

	cargoItem := domain.CargoItem{
		Type: cargoType,
		ID:   cargoID,
	}
	bay.Loaded = append(bay.Loaded, cargoItem)

	design.PayloadMass += cargoMass

	// 重新计算 Δv
	stages, err := engine.PairModules(design.Modules)
	if err != nil {
		fmt.Printf("警告: 重新计算物理参数失败: %v\n", err)
	} else {
		design.DeltaV = engine.CalcDeltaV(stages, design.PayloadMass)
		design.MaxAccel = engine.CalcMaxAccel(stages, design.PayloadMass)
		design.Stages = len(stages)
	}

	fmt.Printf("✅ 装载成功！\n")
	fmt.Printf("   货物: %s (质量: %.0f kg)\n", cargoName, cargoMass)
	fmt.Printf("   已装入货舱 %d\n", bayIndex)
	fmt.Printf("   当前货舱使用: %.0f / %.0f kg\n", usedMass+cargoMass, part.CargoMassCapacity)
	fmt.Printf("   火箭总载荷更新: %.0f kg\n", design.PayloadMass)
	fmt.Printf("   更新后 Δv: %.0f m/s\n", design.DeltaV)
}
