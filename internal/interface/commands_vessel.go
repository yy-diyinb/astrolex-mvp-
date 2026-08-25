package repl

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
)

// ==================== 卫星设计列表 ====================

// listSatellites 显示所有已设计的卫星（非在轨）
func (r *Repl) listSatellites() {
	if len(r.game.SatelliteDesigns) == 0 {
		fmt.Println("暂无卫星设计，请使用 design satellite 创建。")
		return
	}
	for _, s := range r.game.SatelliteDesigns {
		fmt.Printf("ID: %s  名称: %s  质量: %.0f kg  功耗: %.0f W  数据率: %.0f kbps  成本: %d 信用点\n",
			s.ID, s.Name, s.TotalMass, s.TotalPower, s.TotalDataRate, s.TotalCost)
		var mods []string
		for _, m := range s.Modules {
			mods = append(mods, m.Name)
		}
		fmt.Printf("  模块: %s\n", strings.Join(mods, ", "))
	}
}

// loadSatelliteModules 从JSON文件加载卫星模块列表
func loadSatelliteModules(path string) ([]domain.SatelliteModule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var modules []domain.SatelliteModule
	err = json.Unmarshal(data, &modules)
	return modules, err
}

// ==================== 卫星设计（新建与编辑） ====================

func (r *Repl) designSatellite() {
	r.designSatelliteCommon(nil)
}

func (r *Repl) editSatellite(satID string) {
	var sat *domain.SatelliteDesign
	for i := range r.game.SatelliteDesigns {
		if r.game.SatelliteDesigns[i].ID == satID {
			sat = &r.game.SatelliteDesigns[i]
			break
		}
	}
	if sat == nil {
		fmt.Printf("错误: 未找到卫星设计 ID '%s'\n", satID)
		return
	}
	fmt.Printf("编辑卫星设计: %s (%s)\n", sat.Name, sat.ID)
	r.designSatelliteCommon(sat)
}

func (r *Repl) designSatelliteCommon(preset *domain.SatelliteDesign) {
	isEdit := preset != nil
	var name string

	if isEdit {
		fmt.Printf("当前卫星名称: %s (回车保留)\n", preset.Name)
		fmt.Print("请输入卫星名称: ")
		nameInput, _ := r.reader.ReadString('\n')
		nameInput = strings.TrimSpace(nameInput)
		if nameInput == "" {
			name = preset.Name
		} else {
			name = nameInput
		}
	} else {
		fmt.Print("请输入卫星名称: ")
		name, _ = r.reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name == "" {
			name = "未命名卫星"
		}
	}

	moduleCatalog, err := loadSatelliteModules("data/satellite_modules.json")
	if err != nil {
		fmt.Println("加载卫星模块失败:", err)
		return
	}

	fmt.Println("\n可用卫星模块列表:")
	for _, m := range moduleCatalog {
		fmt.Printf("[%s] %s (%s) 质量:%.0fkg 功耗:%.0fW 数据率:%.0fkbps 成本:%d\n",
			m.ID, m.Name, m.Type, m.Mass, m.PowerConsume, m.DataRate, m.Cost)
	}

	if isEdit {
		var currentMods []string
		for _, m := range preset.Modules {
			currentMods = append(currentMods, m.ID)
		}
		if len(currentMods) > 0 {
			fmt.Printf("当前模块序列: %s\n", strings.Join(currentMods, " "))
		}
	}

	var selectedModules []domain.SatelliteModule
	fmt.Println("\n请按顺序输入模块ID（用空格分隔），输入 'done' 结束：")
	for {
		fmt.Print("> ")
		line, _ := r.reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "done") {
			partsBeforeDone := strings.SplitN(line, "done", 2)[0]
			if strings.TrimSpace(partsBeforeDone) == "" {
				break
			}
			tokens := strings.Fields(partsBeforeDone)
			for _, token := range tokens {
				if token == "" {
					continue
				}
				var found *domain.SatelliteModule
				for _, m := range moduleCatalog {
					if m.ID == token {
						found = &m
						break
					}
				}
				if found == nil {
					fmt.Printf("警告: 未找到模块 '%s'\n", token)
					continue
				}
				selectedModules = append(selectedModules, *found)
			}
			break
		}
		tokens := strings.Fields(line)
		for _, token := range tokens {
			if token == "" {
				continue
			}
			var found *domain.SatelliteModule
			for _, m := range moduleCatalog {
				if m.ID == token {
					found = &m
					break
				}
			}
			if found == nil {
				fmt.Printf("警告: 未找到模块 '%s'\n", token)
				continue
			}
			selectedModules = append(selectedModules, *found)
		}
	}

	if len(selectedModules) == 0 {
		fmt.Println("未选择任何模块，取消设计。")
		return
	}

	var totalMass, totalPower, totalDataRate float64
	var totalCost int64
	for _, m := range selectedModules {
		totalMass += m.Mass
		totalPower += m.PowerConsume
		totalDataRate += m.DataRate
		totalCost += m.Cost
	}

	satID := fmt.Sprintf("sat_%d", len(r.game.SatelliteDesigns)+1)
	if isEdit {
		satID = preset.ID
	}

	design := domain.SatelliteDesign{
		ID:            satID,
		Name:          name,
		Modules:       selectedModules,
		TotalMass:     totalMass,
		TotalPower:    totalPower,
		TotalDataRate: totalDataRate,
		TotalCost:     totalCost,
		CreatedAt:     time.Now(),
	}

	if isEdit {
		*preset = design
		if part, ok := r.game.PartsDB[satID]; ok {
			part.Name = name + " (卫星)"
			part.MassDry = totalMass
			part.Cost = totalCost
			part.SatelliteID = satID
			r.game.PartsDB[satID] = part
		}
		fmt.Printf("\n✅ 卫星设计已更新！\n")
	} else {
		r.game.SatelliteDesigns = append(r.game.SatelliteDesigns, design)
		satellitePart := domain.Part{
			ID:           satID,
			Name:         name + " (卫星)",
			Category:     "Satellite",
			SupplierID:   "player",
			MassDry:      totalMass,
			MassFuelMax:  0,
			Thrust:       0,
			ISP:          0,
			FailureRate:  0.001,
			Diameter:     0,
			DeliveryTime: 0,
			InterfaceTag: "SAT",
			Cost:         totalCost,
			SatelliteID:  satID,
			IsControlChip: false,
		}
		r.game.PartsDB[satID] = satellitePart
		fmt.Printf("\n✅ 卫星设计完成！\n")
	}

	fmt.Printf("名称: %s\n", name)
	fmt.Printf("ID: %s\n", satID)
	fmt.Printf("总质量: %.0f kg\n", totalMass)
	fmt.Printf("总功耗: %.0f W\n", totalPower)
	fmt.Printf("总数据率: %.0f kbps\n", totalDataRate)
	fmt.Printf("总成本: %d 信用点\n", totalCost)
	fmt.Printf("已添加到零件列表，可在火箭设计时作为载荷选择 (ID: %s)\n", satID)
}

// ==================== 在轨航天器命令 ====================

// vesselList 列出所有在轨航天器（Vessel）
func (r *Repl) vesselList() {
	if len(r.game.Vessels) == 0 {
		fmt.Println("当前没有在轨航天器。")
		return
	}
	fmt.Println("在轨航天器列表:")
	for _, v := range r.game.Vessels {
		if !v.IsActive {
			continue
		}
		bodyName := "深空"
		if v.OrbitBodyID != "deep_space" && v.OrbitBodyID != "docked" {
			if b, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]; ok {
				bodyName = b.Name
			}
		} else if v.OrbitBodyID == "deep_space" {
			bodyName = "深空"
		} else {
			bodyName = "已对接"
		}
		ccMark := ""
		if v.HasControlCenter {
			ccMark = " [控制中心]"
		}
		typeMark := ""
		switch v.Type {
		case domain.VesselSingle:
			typeMark = " [单体]"
		case domain.VesselComposite:
			typeMark = " [组合体]"
		case domain.VesselCargo:
			typeMark = " [货物]"
		case domain.VesselAircraft:
			typeMark = " [飞机]"
		}
		fmt.Printf("ID: %s  名称: %s%s%s  位置: %s  电力: %.0f Wh  数据: %.0f MB  剩余Δv: %.0f m/s  状态: %s\n",
			v.ID, v.Name, ccMark, typeMark, bodyName, v.Power, v.DataStored, v.DeltaVRemaining, map[bool]string{true: "正常", false: "失效"}[v.IsActive])
	}
}

// vesselStatus 显示航天器详细信息
func (r *Repl) vesselStatus(vesselID string) {
	var v *domain.Vessel
	for _, vv := range r.game.Vessels {
		if vv.ID == vesselID {
			v = &vv
			break
		}
	}
	if v == nil {
		fmt.Printf("错误: 未找到航天器 '%s'\n", vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println("航天器已失效。")
		return
	}

	if v.OrbitBodyID == "deep_space" {
		fmt.Printf("=== 航天器 %s (%s) 深空状态 ===\n", v.Name, v.ID)
		fmt.Printf("类型: %s\n", v.Type)
		fmt.Printf("出发天体: %s\n", v.DepartureBody)
		fmt.Printf("剩余 Δv: %.0f m/s\n", v.DeltaVRemaining)
		fmt.Printf("电力: %.0f / %.0f Wh\n", v.Power, v.MaxPower)
		fmt.Printf("数据: %.0f MB 已存储\n", v.DataStored)
		fmt.Printf("数据率: %.0f kbps\n", v.DataRate)
		fmt.Printf("状态: %s\n", map[bool]string{true: "正常运行", false: "失效"}[v.IsActive])
		if !v.ArrivalTime.IsZero() {
			fmt.Printf("预计到达: %s\n", v.ArrivalTime.Format("2006-01-02"))
		}
		if v.IsAssembly {
			fmt.Printf("组合体状态: 已对接 %d 个航天器\n", len(v.DockedWith))
			fmt.Printf("对接列表: %s\n", strings.Join(v.DockedWith, ", "))
		}
		if len(v.CargoBays) > 0 {
			fmt.Println("货舱信息:")
			for _, bay := range v.CargoBays {
				fmt.Printf("  货舱 %d: %d 件货物\n", bay.Index, len(bay.Loaded))
				for i, item := range bay.Loaded {
					fmt.Printf("    [%d] %s (%s)\n", i+1, r.getCargoName(item), item.Type)
				}
			}
		}
		fmt.Printf("控制中心: %s\n", map[bool]string{true: "✅ 是", false: "❌ 否"}[v.HasControlCenter])
		if len(v.Firmware) > 0 {
			fmt.Printf("已连接固件: %d 个\n", len(v.Firmware))
			for _, fw := range v.Firmware {
				for _, vv := range r.game.Vessels {
					if vv.ID == fw {
						fmt.Printf("  - %s\n", vv.Name)
						break
					}
				}
			}
		}
		return
	}

	if v.OrbitBodyID == "docked" {
		fmt.Printf("航天器 %s 已与组合体对接，状态: 已合并\n", v.Name)
		return
	}

	body, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]
	if !ok {
		fmt.Println("错误: 无法获取天体数据")
		return
	}

	layerName := r.getOrbitLayerName(body, v.OrbitAltitude)
	radius := body.Radius + v.OrbitAltitude
	v_c := engine.CircularOrbitVelocity(body.GM, radius)
	v_esc := engine.EscapeVelocity(body.GM, radius)

	fmt.Printf("=== 航天器 %s (%s) ===\n", v.Name, v.ID)
	fmt.Printf("类型: %s\n", v.Type)
	fmt.Printf("位置: %s 轨道\n", body.Name)
	fmt.Printf("轨道层: %s (高度 %.0f km)\n", layerName, v.OrbitAltitude)
	fmt.Printf("轨道半径: %.0f km\n", radius)
	fmt.Printf("圆轨道速度: %.2f km/s\n", v_c)
	fmt.Printf("逃逸速度: %.2f km/s\n", v_esc)
	fmt.Printf("剩余 Δv: %.0f m/s\n", v.DeltaVRemaining)
	fmt.Printf("发射时间: %s\n", v.LaunchTime.Format("2006-01-02"))
	fmt.Printf("电力: %.0f / %.0f Wh\n", v.Power, v.MaxPower)
	fmt.Printf("数据: %.0f MB 已存储\n", v.DataStored)
	fmt.Printf("数据率: %.0f kbps\n", v.DataRate)
	fmt.Printf("状态: %s\n", map[bool]string{true: "正常运行", false: "失效"}[v.IsActive])
	if v.IsAssembly {
		fmt.Printf("组合体状态: 已对接 %d 个航天器\n", len(v.DockedWith))
		fmt.Printf("对接列表: %s\n", strings.Join(v.DockedWith, ", "))
	}
	if len(v.CargoBays) > 0 {
		fmt.Println("货舱信息:")
		for _, bay := range v.CargoBays {
			fmt.Printf("  货舱 %d: %d 件货物\n", bay.Index, len(bay.Loaded))
			for i, item := range bay.Loaded {
				fmt.Printf("    [%d] %s (%s)\n", i+1, r.getCargoName(item), item.Type)
			}
		}
	}
	fmt.Printf("控制中心: %s\n", map[bool]string{true: "✅ 是", false: "❌ 否"}[v.HasControlCenter])
	if len(v.Firmware) > 0 {
		fmt.Printf("已连接固件: %d 个\n", len(v.Firmware))
		for _, fw := range v.Firmware {
			for _, vv := range r.game.Vessels {
				if vv.ID == fw {
					fmt.Printf("  - %s\n", vv.Name)
					break
				}
			}
		}
	}
	fmt.Println("模块列表:")
	for _, m := range v.Modules {
		fmt.Printf("  - %s (%s): 功耗 %.0fW, 数据率 %.0f kbps\n", m.Name, m.Type, m.PowerConsume, m.DataRate)
	}
}

// ==================== 数据采集与发送 ====================

func (r *Repl) vesselMeasure(vesselID string) {
	var v *domain.Vessel
	for i := range r.game.Vessels {
		if r.game.Vessels[i].ID == vesselID {
			v = &r.game.Vessels[i]
			break
		}
	}
	if v == nil {
		fmt.Printf("错误: 未找到航天器 '%s'\n", vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println("航天器已失效，无法采集数据。")
		return
	}
	if v.OrbitBodyID == "docked" {
		fmt.Println("航天器已对接为组合体的一部分，请使用组合体的ID进行操作。")
		return
	}
	hasSensor := false
	for _, m := range v.Modules {
		if m.Type == "Sensor" {
			hasSensor = true
			break
		}
	}
	if !hasSensor {
		fmt.Println("此航天器没有传感器模块，无法采集数据。")
		return
	}
	if v.Power < r.cfg.Satellite.MeasurePowerCost {
		fmt.Println("电力不足，无法采集数据。")
		return
	}
	v.Power -= r.cfg.Satellite.MeasurePowerCost
	dataAmount := 0.0
	for _, m := range v.Modules {
		if m.Type == "Sensor" {
			dataAmount += m.DataRate * 10 / 8 / 1024
		}
	}
	v.DataStored += dataAmount
	fmt.Printf("数据采集完成，获得 %.2f MB 数据。当前电力: %.0f Wh，数据: %.0f MB\n", dataAmount, v.Power, v.DataStored)
}

func (r *Repl) vesselSend(vesselID string) {
	var v *domain.Vessel
	for i := range r.game.Vessels {
		if r.game.Vessels[i].ID == vesselID {
			v = &r.game.Vessels[i]
			break
		}
	}
	if v == nil {
		fmt.Printf("错误: 未找到航天器 '%s'\n", vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println("航天器已失效，无法发送数据。")
		return
	}
	if v.OrbitBodyID == "docked" {
		fmt.Println("航天器已对接为组合体的一部分，请使用组合体的ID进行操作。")
		return
	}
	if v.DataStored == 0 {
		fmt.Println("没有数据可发送。")
		return
	}
	hasComms := false
	for _, m := range v.Modules {
		if m.Type == "Comms" {
			hasComms = true
			break
		}
	}
	if !hasComms {
		fmt.Println("此航天器没有通信模块，无法发送数据。")
		return
	}
	if v.Power < r.cfg.Satellite.SendPowerCost {
		fmt.Println("电力不足，无法发送数据。")
		return
	}
	v.Power -= r.cfg.Satellite.SendPowerCost
	reward := int64(v.DataStored * float64(r.cfg.Satellite.DataRewardPerMB))
	r.game.Player.Credits += reward
	fmt.Printf("数据发送成功！获得 %d 信用点。当前电力: %.0f Wh\n", reward, v.Power)
	v.DataStored = 0
}

func (r *Repl) vesselPoint(vesselID string, target string) {
	var v *domain.Vessel
	for i := range r.game.Vessels {
		if r.game.Vessels[i].ID == vesselID {
			v = &r.game.Vessels[i]
			break
		}
	}
	if v == nil {
		fmt.Printf("错误: 未找到航天器 '%s'\n", vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println("航天器已失效，无法调整姿态。")
		return
	}
	if v.OrbitBodyID == "docked" {
		fmt.Println("航天器已对接为组合体的一部分，请使用组合体的ID进行操作。")
		return
	}
	fmt.Printf("航天器 %s 已将姿态指向 %s (模拟操作)\n", v.Name, target)
}

// ==================== sat 命令别名（兼容旧命令） ====================

// 为了方便，保留 sat 子命令作为 vessel 的别名
// 在 repl.go 中，sat 命令将调用这些函数
