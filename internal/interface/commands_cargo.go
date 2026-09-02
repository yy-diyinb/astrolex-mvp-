package repl

import (
	"fmt"
	"strconv"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
	"astrolex/internal/i18n"
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
			fmt.Printf(i18n.T("error_design_not_found"), designID)
			return
		}
		if len(design.CargoBays) == 0 {
			fmt.Printf(i18n.T("cargo_info_no_cargo"), design.Name)
			return
		}
		fmt.Printf(i18n.T("cargo_info_rocket_title"), design.Name)
		for _, bay := range design.CargoBays {
			fmt.Printf(i18n.T("cargo_info_cargo_bay"), bay.Index)
			if len(bay.Loaded) == 0 {
				fmt.Println(i18n.T("cargo_info_empty"))
			} else {
				for i, item := range bay.Loaded {
					name := r.getCargoName(item)
					mass := r.getCargoMass(item)
					fmt.Printf(i18n.T("cargo_info_item"), i+1, name, item.Type, mass)
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
			fmt.Printf(i18n.T("error_vessel_not_found"), designID)
			return
		}
		if len(vessel.CargoBays) == 0 {
			fmt.Printf(i18n.T("cargo_info_no_cargo_vessel"), vessel.Name)
			return
		}
		fmt.Printf(i18n.T("cargo_info_vessel_title"), vessel.Name)
		for _, bay := range vessel.CargoBays {
			fmt.Printf(i18n.T("cargo_info_cargo_bay"), bay.Index)
			if len(bay.Loaded) == 0 {
				fmt.Println(i18n.T("cargo_info_empty"))
			} else {
				for i, item := range bay.Loaded {
					name := r.getCargoName(item)
					mass := r.getCargoMass(item)
					fmt.Printf(i18n.T("cargo_info_item"), i+1, name, item.Type, mass)
				}
			}
		}
	default:
		fmt.Printf(i18n.T("cargo_info_invalid_type"), designType)
	}
}

// ==================== 装载货物 ====================

func (r *Repl) cargoLoadCommand(designID string, bayIndexStr string, cargoType string, cargoID string) {
	// 解析货舱序号
	bayIndex, err := strconv.Atoi(bayIndexStr)
	if err != nil || bayIndex < 1 {
		fmt.Println(i18n.T("cargo_load_invalid_bay"))
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
		fmt.Printf(i18n.T("error_design_not_found"), designID)
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
		fmt.Printf(i18n.T("cargo_load_bay_not_found"), bayIndex)
		var bays []int
		for i, mod := range design.Modules {
			if mod.Type == domain.ModuleCargoBay {
				bays = append(bays, i)
			}
		}
		if len(bays) > 0 {
			fmt.Print(i18n.T("cargo_load_available_bays"))
			for i := range bays {
				fmt.Printf("%d ", i+1)
			}
			fmt.Println()
		} else {
			fmt.Println(i18n.T("cargo_load_no_bay"))
		}
		return
	}

	// 获取货舱零件的容量
	partID := design.Modules[cargoModIdx].PartID
	part, ok := r.game.PartsDB[partID]
	if !ok {
		fmt.Printf(i18n.T("cargo_load_part_not_found"), partID)
		return
	}
	if part.CargoMassCapacity <= 0 {
		fmt.Println(i18n.T("cargo_load_capacity_zero"))
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
			fmt.Printf(i18n.T("error_design_not_found"), cargoID)
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
			fmt.Printf(i18n.T("error_sat_design_not_found"), cargoID)
			return
		}
		cargoMass = subDesign.TotalMass
		cargoName = subDesign.Name + " (卫星)"
	case "part":
		part2, ok := r.game.PartsDB[cargoID]
		if !ok {
			fmt.Printf(i18n.T("cargo_load_part_not_found"), cargoID)
			return
		}
		cargoMass = part2.MassDry
		cargoName = part2.Name
	default:
		fmt.Printf(i18n.T("cargo_load_invalid_type"), cargoType)
		return
	}

	if cargoMass <= 0 {
		fmt.Println(i18n.T("cargo_load_zero_mass"))
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
		fmt.Println(i18n.T("cargo_load_failed_capacity"))
		fmt.Printf(i18n.T("cargo_load_capacity_info"), part.CargoMassCapacity)
		fmt.Printf(i18n.T("cargo_load_used"), usedMass)
		fmt.Printf(i18n.T("cargo_load_cargo_mass"), cargoMass)
		fmt.Printf(i18n.T("cargo_load_exceed"), usedMass+cargoMass-part.CargoMassCapacity)
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
		fmt.Printf(i18n.T("cargo_load_recalc_failed"), err)
	} else {
		design.DeltaV = engine.CalcDeltaV(stages, design.PayloadMass)
		design.MaxAccel = engine.CalcMaxAccel(stages, design.PayloadMass)
		design.Stages = len(stages)
	}

	fmt.Println(i18n.T("cargo_load_success"))
	fmt.Printf(i18n.T("cargo_load_item"), cargoName, cargoMass)
	fmt.Printf(i18n.T("cargo_load_bay"), bayIndex)
	fmt.Printf(i18n.T("cargo_load_usage"), usedMass+cargoMass, part.CargoMassCapacity)
	fmt.Printf(i18n.T("cargo_load_payload"), design.PayloadMass)
	fmt.Printf(i18n.T("cargo_load_delta_v"), design.DeltaV)
}
