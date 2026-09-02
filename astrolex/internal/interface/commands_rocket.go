package repl

import (
	"fmt"
	"strconv"
	"strings"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
)

// ==================== 零件列表 ====================
func (r *Repl) listParts() {
	fmt.Println("零件列表:")
	for id, p := range r.game.PartsDB {
		category := p.Category
		fmt.Printf("[%s] %s: %s (质量: %.0f kg, 成本: %d 信用点", id, category, p.Name, p.MassDry, p.Cost)
		if category == "Engine" {
			fmt.Printf(", 推力: %.0f kN, 比冲: %.0fs", p.Thrust, p.ISP)
		} else if category == "FuelTank" {
			fmt.Printf(", 容量: %.0f kg", p.MassFuelMax)
		} else if category == "Satellite" {
			fmt.Printf(", 卫星设计: %s", p.SatelliteID)
		} else if category == "CargoBay" {
			fmt.Printf(", 货舱容量: %.0f kg", p.CargoMassCapacity)
		}
		fmt.Println(")")
	}
}

// ==================== 火箭设计列表 ====================
func (r *Repl) listDesigns() {
	if len(r.game.RocketDesigns) == 0 {
		fmt.Println("暂无火箭设计，请使用 design 创建。")
		return
	}
	for _, d := range r.game.RocketDesigns {
		var parts []string
		for _, mod := range d.Modules {
			parts = append(parts, string(mod.Type))
		}
		diagram := strings.Join(parts, "-")
		hardwareCost := r.calcHardwareCost(d)
		fmt.Printf("ID: %s  名称: %s  级数: %d  Δv: %.0f m/s  载荷: %.0f kg  硬件成本: %d 信用点\n", d.ID, d.Name, d.Stages, d.DeltaV, d.PayloadMass, hardwareCost)
		fmt.Printf("  构造图: %s\n", diagram)
	}
}

// ==================== 硬件成本计算 ====================
func (r *Repl) calcHardwareCost(design domain.RocketDesign) int64 {
	var total int64
	for _, mod := range design.Modules {
		if mod.Part != nil {
			total += mod.Part.Cost
		}
	}
	return total
}

// ==================== 发射成本计算 ====================
func (r *Repl) calcLaunchCost(design domain.RocketDesign) int64 {
	hardware := r.calcHardwareCost(design)
	var fuelMass float64
	for _, mod := range design.Modules {
		if mod.Type == domain.ModuleFuelTank && mod.Part != nil {
			fuelMass += mod.Part.MassFuelMax * 0.5 // 估算加注一半
		}
	}
	fuelCost := int64(fuelMass * 0.05)
	padCost := int64(500)
	return hardware + fuelCost + padCost
}

// ==================== 示例模拟 ====================
func (r *Repl) simulateDemo() {
	enginePart, ok1 := r.game.PartsDB["kr-99"]
	tankPart, ok2 := r.game.PartsDB["t-4000"]
	fairingPart, ok3 := r.game.PartsDB["faring_6m"]
	avionicsPart, ok4 := r.game.PartsDB["av-1"]
	if !ok1 || !ok2 || !ok3 || !ok4 {
		fmt.Println("错误: 缺少示例零件")
		return
	}
	modules := []domain.Module{
		{Type: domain.ModuleFairing, PartID: "faring_6m", Part: &fairingPart},
		{Type: domain.ModuleAvionics, PartID: "av-1", Part: &avionicsPart},
		{Type: domain.ModuleFuelTank, PartID: "t-4000", Part: &tankPart},
		{Type: domain.ModuleEngine, PartID: "kr-99", Part: &enginePart},
		{Type: domain.ModuleFuelTank, PartID: "t-4000", Part: &tankPart},
		{Type: domain.ModuleEngine, PartID: "kr-99", Part: &enginePart},
	}
	dv, err := engine.CalcDeltaVFromModules(modules, 5000)
	if err != nil {
		fmt.Println("模拟错误:", err)
		return
	}
	accel, err := engine.CalcMaxAccelFromModules(modules, 5000)
	if err != nil {
		fmt.Println("模拟错误:", err)
		return
	}
	fmt.Printf("示例火箭 Δv = %.0f m/s, 最大加速度 = %.2f G\n", dv, accel)
}

// ==================== 获取卫星总质量 ====================
func (r *Repl) getSatelliteMass(design domain.RocketDesign) float64 {
	var mass float64
	for _, mod := range design.Modules {
		if mod.Part != nil && mod.Part.Category == "Satellite" {
			mass += mod.Part.MassDry
		}
	}
	return mass
}

// ==================== 火箭设计（新建） ====================
func (r *Repl) designRocket() {
	r.designRocketCommon(nil)
}

// ==================== 编辑火箭设计 ====================
func (r *Repl) editDesign(designID string) {
	var design *domain.RocketDesign
	for i := range r.game.RocketDesigns {
		if r.game.RocketDesigns[i].ID == designID {
			design = &r.game.RocketDesigns[i]
			break
		}
	}
	if design == nil {
		fmt.Printf("错误: 未找到设计 ID '%s'\n", designID)
		return
	}
	fmt.Printf("编辑火箭设计: %s (%s)\n", design.Name, design.ID)
	r.designRocketCommon(design)
}

// ==================== 火箭设计核心逻辑 ====================
func (r *Repl) designRocketCommon(preset *domain.RocketDesign) {
	isEdit := preset != nil
	var name string
	var basePayload float64

	if isEdit {
		fmt.Printf("当前火箭名称: %s (回车保留)\n", preset.Name)
		fmt.Print("请输入火箭名称: ")
		nameInput, _ := r.reader.ReadString('\n')
		nameInput = strings.TrimSpace(nameInput)
		if nameInput == "" {
			name = preset.Name
		} else {
			name = nameInput
		}

		satMass := r.getSatelliteMass(*preset)
		currentBasePayload := preset.PayloadMass - satMass
		if currentBasePayload < 0 {
			currentBasePayload = 0
		}
		fmt.Printf("当前基础载荷质量: %.0f kg (回车保留)\n", currentBasePayload)
		fmt.Print("请输入基础载荷质量 (kg) [不含卫星质量]: ")
		payloadStr, _ := r.reader.ReadString('\n')
		payloadStr = strings.TrimSpace(payloadStr)
		if payloadStr == "" {
			basePayload = currentBasePayload
		} else {
			val, err := strconv.ParseFloat(payloadStr, 64)
			if err != nil || val <= 0 {
				fmt.Println("载荷无效，使用当前值")
				basePayload = currentBasePayload
			} else {
				basePayload = val
			}
		}
	} else {
		fmt.Print("请输入火箭名称: ")
		name, _ = r.reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name == "" {
			name = "未命名火箭"
		}

		fmt.Print("请输入基础载荷质量 (kg) [不含卫星质量]: ")
		payloadStr, _ := r.reader.ReadString('\n')
		payloadStr = strings.TrimSpace(payloadStr)
		val, err := strconv.ParseFloat(payloadStr, 64)
		if err != nil || val <= 0 {
			fmt.Println("载荷无效，默认 5000 kg")
			basePayload = 5000
		} else {
			basePayload = val
		}
	}

	r.listParts()
	fmt.Println("\n请按从顶部到底部的顺序输入零件ID，用空格分隔，输入 'done' 结束：")
	fmt.Println("提示：引擎会自动与上方最近的燃料箱配对。卫星零件将自动计入有效载荷质量。")

	if isEdit {
		var currentParts []string
		for _, mod := range preset.Modules {
			currentParts = append(currentParts, mod.PartID)
		}
		if len(currentParts) > 0 {
			fmt.Printf("当前零件序列: %s\n", strings.Join(currentParts, " "))
		}
	}

	var inputIDs []string
	for {
		fmt.Print("> ")
		line, _ := r.reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		tokens := strings.Fields(line)
		doneFound := false
		for _, token := range tokens {
			if strings.EqualFold(token, "done") {
				doneFound = true
				break
			}
			if token != "" {
				inputIDs = append(inputIDs, token)
			}
		}
		if doneFound {
			break
		}
	}

	if len(inputIDs) == 0 {
		fmt.Println("未输入任何零件，取消设计。")
		return
	}

	var modules []domain.Module
	satelliteMass := 0.0
	cargoBayCount := 0 // 用于记录货舱序号（从1开始）
	for _, id := range inputIDs {
		part, ok := r.game.PartsDB[id]
		if !ok {
			fmt.Printf("警告: 零件ID '%s' 不存在，跳过\n", id)
			continue
		}
		if part.Category == "Satellite" {
			satelliteMass += part.MassDry
			fmt.Printf("✅ 已加入卫星 '%s'，质量 %.0f kg\n", part.Name, part.MassDry)
			p := part
			mod := domain.Module{
				Type:   domain.ModuleSatellite,
				PartID: id,
				Part:   &p,
			}
			modules = append(modules, mod)
			continue
		}
		if part.Category == "CargoBay" {
			cargoBayCount++
			fmt.Printf("✅ 已加入货舱 '%s' (序号 %d)，容量 %.0f kg\n", part.Name, cargoBayCount, part.CargoMassCapacity)
			p := part
			mod := domain.Module{
				Type:   domain.ModuleCargoBay,
				PartID: id,
				Part:   &p,
			}
			modules = append(modules, mod)
			continue
		}
		var modType domain.ModuleType
		switch part.Category {
		case "Engine":
			modType = domain.ModuleEngine
		case "FuelTank":
			modType = domain.ModuleFuelTank
		case "Avionics":
			modType = domain.ModuleAvionics
		case "AeroShell":
			modType = domain.ModuleFairing
		default:
			fmt.Printf("警告: 零件 '%s' 类别 '%s' 不支持，跳过\n", part.Name, part.Category)
			continue
		}
		p := part
		mod := domain.Module{
			Type:   modType,
			PartID: id,
			Part:   &p,
		}
		modules = append(modules, mod)
	}

	if len(modules) == 0 {
		fmt.Println("没有有效零件，取消设计。")
		return
	}

	totalPayload := basePayload + satelliteMass

	// 计算并保存货舱信息（用于后续的 load 命令）
	// 这里我们只需在设计中记录货舱数量，具体装载由 load 命令管理
	// 设计本身不存储货舱内容，但我们需要确保 CargoBays 字段被正确初始化
	// 由于设计可能被编辑，我们重置 CargoBays 为空列表，因为 load 命令会重新填充
	if isEdit {
		// 保留已有货舱装载？编辑时清空，因为重新设计了零件序列
		preset.CargoBays = []domain.CargoBay{}
	} else {
		// 新设计初始化为空
	}

	dv, err := engine.CalcDeltaVFromModules(modules, totalPayload)
	if err != nil {
		fmt.Println("设计错误:", err)
		return
	}
	accel, err := engine.CalcMaxAccelFromModules(modules, totalPayload)
	if err != nil {
		fmt.Println("设计错误:", err)
		return
	}
	stages, err := engine.PairModules(modules)
	if err != nil {
		fmt.Println("配对错误:", err)
		return
	}
	numStages := len(stages)

	var typeNames []string
	for _, mod := range modules {
		typeNames = append(typeNames, string(mod.Type))
	}
	diagram := strings.Join(typeNames, "-")

	fmt.Printf("\n=== 设计结果 ===\n")
	fmt.Printf("名称: %s\n", name)
	fmt.Printf("构造图: %s\n", diagram)
	fmt.Printf("级数: %d\n", numStages)
	fmt.Printf("基础载荷: %.0f kg\n", basePayload)
	fmt.Printf("卫星载荷: %.0f kg\n", satelliteMass)
	fmt.Printf("总载荷: %.0f kg\n", totalPayload)
	fmt.Printf("总 Δv: %.0f m/s\n", dv)
	fmt.Printf("最大加速度: %.2f G\n", accel)

	if isEdit {
		preset.Name = name
		preset.Modules = modules
		preset.PayloadMass = totalPayload
		preset.DeltaV = dv
		preset.MaxAccel = accel
		preset.Stages = numStages
		fmt.Printf("设计已更新，ID: %s\n", preset.ID)
	} else {
		design := domain.RocketDesign{
			ID:          fmt.Sprintf("design_%d", len(r.game.RocketDesigns)+1),
			Name:        name,
			ContractID:  "",
			Modules:     modules,
			PayloadMass: totalPayload,
			TotalMass:   0,
			DeltaV:      dv,
			MaxAccel:    accel,
			Stages:      numStages,
			CargoBays:   []domain.CargoBay{}, // 初始为空，由 load 命令填充
		}
		r.game.RocketDesigns = append(r.game.RocketDesigns, design)
		fmt.Printf("设计已保存，ID: %s\n", design.ID)
	}
}
