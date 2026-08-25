package repl

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
)

// ==================== 辅助函数 ====================

// getOrbitLayerName 根据高度返回轨道层名称
func (r *Repl) getOrbitLayerName(body domain.CelestialBody, altitude float64) string {
	if len(body.OrbitalLayers) == 0 {
		return "未知轨道"
	}
	radius := body.Radius + altitude
	for _, layer := range body.OrbitalLayers {
		layerRadiusMin := body.Radius + layer.AltitudeMin
		layerRadiusMax := body.Radius + layer.AltitudeMax
		if radius >= layerRadiusMin && radius <= layerRadiusMax {
			return layer.Name
		}
	}
	if radius > body.Radius+body.OrbitalLayers[len(body.OrbitalLayers)-1].AltitudeMax {
		return "高轨道 (超限)"
	}
	return "低轨道 (超限)"
}

// getOrbitLayerIndex 返回当前所在轨道层的索引（0=最低）
func (r *Repl) getOrbitLayerIndex(body domain.CelestialBody, altitude float64) int {
	if len(body.OrbitalLayers) == 0 {
		return -1
	}
	radius := body.Radius + altitude
	for i, layer := range body.OrbitalLayers {
		layerRadiusMin := body.Radius + layer.AltitudeMin
		layerRadiusMax := body.Radius + layer.AltitudeMax
		if radius >= layerRadiusMin && radius <= layerRadiusMax {
			return i
		}
	}
	if radius > body.Radius+body.OrbitalLayers[len(body.OrbitalLayers)-1].AltitudeMax {
		return len(body.OrbitalLayers)
	}
	return -1
}

// getCargoBayByIndex 从航天器的货舱列表中根据序号查找货舱
func (r *Repl) getCargoBayByIndex(v *domain.Vessel, index int) (*domain.CargoBay, int) {
	for i := range v.CargoBays {
		if v.CargoBays[i].Index == index {
			return &v.CargoBays[i], i
		}
	}
	return nil, -1
}

// getCargoItemByIndex 从货舱中根据货物索引查找货物
func (r *Repl) getCargoItemByIndex(bay *domain.CargoBay, itemIndex int) (*domain.CargoItem, int) {
	if itemIndex < 1 || itemIndex > len(bay.Loaded) {
		return nil, -1
	}
	return &bay.Loaded[itemIndex-1], itemIndex - 1
}

// getCargoMass 计算货物质量
func (r *Repl) getCargoMass(item domain.CargoItem) float64 {
	switch item.Type {
	case "rocket":
		for _, d := range r.game.RocketDesigns {
			if d.ID == item.ID {
				return d.PayloadMass
			}
		}
	case "satellite":
		for _, d := range r.game.SatelliteDesigns {
			if d.ID == item.ID {
				return d.TotalMass
			}
		}
	case "part":
		if p, ok := r.game.PartsDB[item.ID]; ok {
			return p.MassDry
		}
	}
	return 0
}

// getCargoName 获取货物名称
func (r *Repl) getCargoName(item domain.CargoItem) string {
	switch item.Type {
	case "rocket":
		for _, d := range r.game.RocketDesigns {
			if d.ID == item.ID {
				return d.Name + " (火箭)"
			}
		}
	case "satellite":
		for _, d := range r.game.SatelliteDesigns {
			if d.ID == item.ID {
				return d.Name + " (卫星)"
			}
		}
	case "part":
		if p, ok := r.game.PartsDB[item.ID]; ok {
			return p.Name
		}
	}
	return "未知货物"
}

// ==================== 控制中心辅助 ====================



// ==================== 获取航天器日心状态 ====================

// getFlybyState 从航天器构建日心状态
func (r *Repl) getFlybyState(v *domain.Vessel) (*engine.FlybyState, error) {
	state := &engine.FlybyState{
		Pos:             v.HeliocentricPos,
		Vel:             v.HeliocentricVel,
		DeltaVRemaining: v.DeltaVRemaining,
		CurrentBodyID:   v.OrbitBodyID,
	}
	if state.Pos[0] == 0 && state.Pos[1] == 0 && v.OrbitBodyID != "deep_space" {
		body, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]
		if !ok {
			return nil, fmt.Errorf("无法获取天体数据")
		}
		px, py := engine.PositionAtTime(body, r.game.CurrentTime)
		vx, vy := engine.OrbitalVelocity(body, r.game.CurrentTime)
		state.Pos = [2]float64{px, py}
		state.Vel = [2]float64{vx, vy}
		state.CurrentBodyID = v.OrbitBodyID
		v.HeliocentricPos = state.Pos
		v.HeliocentricVel = state.Vel
	}
	return state, nil
}

// updateVesselState 将日心状态更新回航天器
func (r *Repl) updateVesselState(v *domain.Vessel, state *engine.FlybyState) {
	v.HeliocentricPos = state.Pos
	v.HeliocentricVel = state.Vel
	v.DeltaVRemaining = state.DeltaVRemaining
	if state.CurrentBodyID != "" && state.CurrentBodyID != "deep_space" {
		v.OrbitBodyID = state.CurrentBodyID
		if body, ok := r.game.StarSystem.CelestialBodies[state.CurrentBodyID]; ok {
			px, py := engine.PositionAtTime(body, r.game.CurrentTime)
			dist := math.Sqrt((state.Pos[0]-px)*(state.Pos[0]-px) + (state.Pos[1]-py)*(state.Pos[1]-py))
			alt := dist - body.Radius
			if alt < 0 {
				alt = body.OrbitalLayers[len(body.OrbitalLayers)-1].TypicalAltitude
			}
			v.OrbitAltitude = alt
		}
	}
}

// ==================== orbit info 命令 ====================
func (r *Repl) orbitCommand(vesselID string) {
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
		fmt.Println("航天器已失效。")
		return
	}

	if v.OrbitBodyID == "deep_space" {
		fmt.Printf("=== 航天器 %s (%s) 深空状态 ===\n", v.Name, v.ID)
		fmt.Printf("类型: %s\n", v.Type)
		fmt.Printf("出发天体: %s\n", v.DepartureBody)
		fmt.Printf("日心位置: (%.0f, %.0f) km\n", v.HeliocentricPos[0], v.HeliocentricPos[1])
		fmt.Printf("日心速度: (%.2f, %.2f) km/s\n", v.HeliocentricVel[0], v.HeliocentricVel[1])
		fmt.Printf("剩余 Δv: %.0f m/s\n", v.DeltaVRemaining)
		fmt.Printf("数据: %.0f MB\n", v.DataStored)
		fmt.Printf("电力: %.0f Wh\n", v.Power)
		if !v.ArrivalTime.IsZero() {
			fmt.Printf("预计到达: %s\n", v.ArrivalTime.Format("2006-01-02"))
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

	body, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]
	if !ok {
		fmt.Println("错误: 无法获取天体数据")
		return
	}

	layerName := r.getOrbitLayerName(body, v.OrbitAltitude)
	radius := body.Radius + v.OrbitAltitude
	v_c := engine.CircularOrbitVelocity(body.GM, radius)
	v_esc := engine.EscapeVelocity(body.GM, radius)

	fmt.Printf("=== 航天器 %s 轨道信息 ===\n", v.Name)
	fmt.Printf("类型: %s\n", v.Type)
	fmt.Printf("绕转天体: %s\n", body.Name)
	fmt.Printf("轨道高度: %.0f km\n", v.OrbitAltitude)
	fmt.Printf("轨道层: %s\n", layerName)
	fmt.Printf("轨道半径: %.0f km\n", radius)
	fmt.Printf("圆轨道速度: %.2f km/s\n", v_c)
	fmt.Printf("逃逸速度: %.2f km/s\n", v_esc)
	fmt.Printf("剩余 Δv: %.0f m/s\n", v.DeltaVRemaining)
	fmt.Printf("日心位置: (%.0f, %.0f) km\n", v.HeliocentricPos[0], v.HeliocentricPos[1])
	fmt.Printf("日心速度: (%.2f, %.2f) km/s\n", v.HeliocentricVel[0], v.HeliocentricVel[1])
	if v.OrbitAltitude > 0 {
		fmt.Printf("逃逸所需 Δv: %.0f m/s\n", engine.EscapeFromOrbitDeltaV(body.GM, radius)*1000)
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
}

// ==================== orbit transfer 命令 ====================
func (r *Repl) transferCommand(vesselID string, targetLayer string) {
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
		fmt.Println("航天器已失效。")
		return
	}
	if v.OrbitBodyID == "deep_space" {
		fmt.Println("航天器在深空，无法变轨。使用 'orbit travel' 进行星际航行。")
		return
	}

	body, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]
	if !ok {
		fmt.Println("错误: 无法获取天体数据")
		return
	}
	if len(body.OrbitalLayers) == 0 {
		fmt.Println("该天体没有定义轨道层，无法转移。")
		return
	}

	var targetIndex int = -1
	if idx, err := strconv.Atoi(targetLayer); err == nil {
		if idx >= 0 && idx < len(body.OrbitalLayers) {
			targetIndex = idx
		}
	} else {
		for i, layer := range body.OrbitalLayers {
			if strings.Contains(strings.ToLower(layer.Name), strings.ToLower(targetLayer)) {
				targetIndex = i
				break
			}
		}
	}
	if targetIndex == -1 {
		fmt.Printf("错误: 未找到目标轨道层 '%s'\n", targetLayer)
		fmt.Printf("可用轨道层: ")
		for i, layer := range body.OrbitalLayers {
			fmt.Printf("%d:%s ", i, layer.Name)
		}
		fmt.Println()
		return
	}

	currentIndex := r.getOrbitLayerIndex(body, v.OrbitAltitude)
	if currentIndex == targetIndex {
		fmt.Printf("航天器已在目标轨道层 '%s'\n", body.OrbitalLayers[targetIndex].Name)
		return
	}

	currentRadius := body.Radius + v.OrbitAltitude
	targetRadius := body.Radius + body.OrbitalLayers[targetIndex].TypicalAltitude
	dv_kmps := engine.OrbitalLayerDeltaV(body.GM, currentRadius, targetRadius)
	dv_mps := dv_kmps * 1000

	fmt.Printf("从当前轨道 (%.0f km) 转移到 %s (%.0f km)\n",
		v.OrbitAltitude, body.OrbitalLayers[targetIndex].Name, body.OrbitalLayers[targetIndex].TypicalAltitude)
	fmt.Printf("需要 Δv: %.0f m/s\n", dv_mps)

	if v.DeltaVRemaining < dv_mps {
		fmt.Printf("❌ 剩余 Δv 不足! 需要 %.0f m/s, 当前剩余 %.0f m/s\n", dv_mps, v.DeltaVRemaining)
		return
	}
	if v.Power < 50 {
		fmt.Println("❌ 电力不足，无法执行变轨。需要至少 50 Wh。")
		return
	}

	v.DeltaVRemaining -= dv_mps
	v.Power -= 50
	v.OrbitAltitude = body.OrbitalLayers[targetIndex].TypicalAltitude
	fmt.Printf("✅ 变轨成功！当前轨道高度: %.0f km, 剩余 Δv: %.0f m/s\n", v.OrbitAltitude, v.DeltaVRemaining)
}

// ==================== orbit escape 命令 ====================
func (r *Repl) escapeCommand(vesselID string) {
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
		fmt.Println("航天器已失效。")
		return
	}
	if v.OrbitBodyID == "deep_space" {
		fmt.Println("航天器已在深空。")
		return
	}

	body, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]
	if !ok {
		fmt.Println("错误: 无法获取天体数据")
		return
	}

	currentRadius := body.Radius + v.OrbitAltitude
	if len(body.OrbitalLayers) > 0 {
		topLayer := body.OrbitalLayers[len(body.OrbitalLayers)-1]
		topRadius := body.Radius + topLayer.TypicalAltitude
		if currentRadius < topRadius-100 {
			fmt.Printf("⚠️ 当前不在最高轨道层！请先变轨到 %s。\n", topLayer.Name)
			fmt.Printf("  当前高度: %.0f km, 需要: %.0f km\n", v.OrbitAltitude, topLayer.TypicalAltitude)
			fmt.Println("  使用 'orbit transfer' 变轨到最高层。")
			return
		}
	}

	dv_kmps := engine.EscapeFromOrbitDeltaV(body.GM, currentRadius)
	dv_mps := dv_kmps * 1000

	fmt.Printf("从当前轨道 (%.0f km) 弹射离开 %s\n", v.OrbitAltitude, body.Name)
	fmt.Printf("需要 Δv: %.0f m/s\n", dv_mps)

	if v.DeltaVRemaining < dv_mps {
		fmt.Printf("❌ 剩余 Δv 不足! 需要 %.0f m/s, 当前剩余 %.0f m/s\n", dv_mps, v.DeltaVRemaining)
		return
	}
	if v.Power < 100 {
		fmt.Println("❌ 电力不足，无法弹射。需要至少 100 Wh。")
		return
	}

	v.DeltaVRemaining -= dv_mps
	v.Power -= 100
	v.DepartureBody = body.ID
	v.OrbitBodyID = "deep_space"
	v.OrbitAltitude = 0
	px, py := engine.PositionAtTime(body, r.game.CurrentTime)
	v.HeliocentricPos = [2]float64{px, py}
	v.HeliocentricVel = [2]float64{v.HeliocentricVel[0], v.HeliocentricVel[1]}
	fmt.Printf("✅ 弹射成功！航天器 '%s' 已进入深空，离开 %s 引力范围。\n", v.Name, body.Name)
	fmt.Printf("   剩余 Δv: %.0f m/s\n", v.DeltaVRemaining)
	fmt.Println("🚀 现在可以使用 'orbit travel' 进行星际航行。")
}

// ==================== orbit dock 命令 ====================
func (r *Repl) dockCommand(masterID, slaveID string) {
	var master, slave *domain.Vessel
	for i := range r.game.Vessels {
		if r.game.Vessels[i].ID == masterID {
			master = &r.game.Vessels[i]
		}
		if r.game.Vessels[i].ID == slaveID {
			slave = &r.game.Vessels[i]
		}
	}
	if master == nil || slave == nil {
		fmt.Println("错误: 未找到指定的航天器")
		return
	}
	if !master.IsActive || !slave.IsActive {
		fmt.Println("错误: 航天器已失效")
		return
	}
	if master.ID == slave.ID {
		fmt.Println("错误: 不能与自身对接")
		return
	}
	if master.OrbitBodyID == "deep_space" || slave.OrbitBodyID == "deep_space" {
		fmt.Println("错误: 深空航天器无法对接（需要轨道上对接）")
		return
	}
	if master.OrbitBodyID != slave.OrbitBodyID {
		fmt.Printf("错误: 两个航天器不在同一轨道 (%s vs %s)\n", master.OrbitBodyID, slave.OrbitBodyID)
		return
	}
	if master.OrbitAltitude < 0 || slave.OrbitAltitude < 0 {
		fmt.Println("错误: 无效轨道高度")
		return
	}
	if master.OrbitAltitude > slave.OrbitAltitude+100 || master.OrbitAltitude < slave.OrbitAltitude-100 {
		fmt.Printf("错误: 高度差超过100km (%.0f vs %.0f)\n", master.OrbitAltitude, slave.OrbitAltitude)
		return
	}
	if master.IsAssembly && contains(master.DockedWith, slave.ID) {
		fmt.Printf("航天器 %s 已经与 %s 对接\n", master.Name, slave.Name)
		return
	}
	if slave.IsAssembly && contains(slave.DockedWith, master.ID) {
		fmt.Printf("航天器 %s 已经与 %s 对接\n", slave.Name, master.Name)
		return
	}
	// 允许任意组合体作为从星，合并后统一为组合体

	// 控制中心与固件逻辑
	masterHasControl := master.HasControlCenter
	slaveHasControl := slave.HasControlCenter
	if slaveHasControl && !masterHasControl {
		master.HasControlCenter = true
	}
	if !slaveHasControl {
		master.Firmware = append(master.Firmware, slave.ID)
		fmt.Printf("🔌 从星 '%s' 已作为固件模块连接至控制中心\n", slave.Name)
	} else {
		master.Firmware = append(master.Firmware, slave.Firmware...)
		fmt.Printf("🔌 控制中心已合并，固件列表扩展至 %d 个\n", len(master.Firmware))
	}

	// 合并数据
	master.Modules = append(master.Modules, slave.Modules...)
	master.MaxPower += slave.MaxPower
	master.Power += slave.Power
	master.DataStored += slave.DataStored
	master.DataRate += slave.DataRate
	master.DeltaVRemaining += slave.DeltaVRemaining
	master.IsAssembly = true
	master.DockedWith = append(master.DockedWith, slave.ID)
	master.CargoBays = append(master.CargoBays, slave.CargoBays...)

	// 合并固件列表（如果从星有固件，也合并）
	master.Firmware = append(master.Firmware, slave.Firmware...)
	// 去重（可选）

	// 标记从星为已对接
	slave.IsActive = false
	slave.OrbitBodyID = "docked"

	fmt.Printf("✅ 对接成功！%s 和 %s 合并为组合体 %s\n", master.Name, slave.Name, master.Name)
	fmt.Printf("   总电力: %.0f / %.0f Wh\n", master.Power, master.MaxPower)
	fmt.Printf("   总数据率: %.0f kbps\n", master.DataRate)
	fmt.Printf("   剩余 Δv: %.0f m/s\n", master.DeltaVRemaining)
	fmt.Printf("   对接列表: %s\n", strings.Join(master.DockedWith, ", "))
	if len(slave.CargoBays) > 0 {
		fmt.Printf("   货舱已合并: %d 个货舱\n", len(slave.CargoBays))
	}
	fmt.Printf("   控制中心: %s\n", map[bool]string{true: "✅ 是", false: "❌ 否"}[master.HasControlCenter])
	if len(master.Firmware) > 0 {
		fmt.Printf("   已连接固件: %d 个\n", len(master.Firmware))
	}
}

// ==================== orbit travel 命令 ====================
func (r *Repl) travelCommand(vesselID, targetID string) {
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
		fmt.Println("航天器已失效")
		return
	}

	state, err := r.getFlybyState(v)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	if v.OrbitBodyID != "deep_space" {
		fmt.Printf("错误: 航天器当前不在深空，位于 %s，请先执行 'orbit escape'\n", v.OrbitBodyID)
		return
	}
	if v.DepartureBody == "" {
		fmt.Println("错误: 缺少出发天体信息，无法计算转移")
		return
	}

	target, ok := r.game.StarSystem.CelestialBodies[targetID]
	if !ok {
		fmt.Printf("错误: 未找到目标天体 '%s'\n", targetID)
		return
	}
	fromBody, ok := r.game.StarSystem.CelestialBodies[v.DepartureBody]
	if !ok {
		fmt.Println("错误: 出发天体数据缺失")
		return
	}

	newState, dv, flightDays, err := engine.ApplyHohmannTransfer(state, target, r.game.CurrentTime)
	if err != nil {
		fmt.Printf("转移计算失败: %v\n", err)
		return
	}

	windowStart, windowEnd, waitDays, _, err := engine.NextWindow(fromBody, target, r.game.CurrentTime, r.cfg.Physics.WindowSearchDays)
	if err != nil {
		fmt.Printf("计算窗口失败: %v\n", err)
		return
	}
	current := r.game.CurrentTime
	if current.Before(windowStart) || current.After(windowEnd) {
		fmt.Printf("\n❌ 当前时间不在发射窗口内！\n")
		fmt.Printf("下一个窗口开始: %s\n", windowStart.Format("2006-01-02"))
		fmt.Printf("需要等待: %d 天\n", waitDays)
		fmt.Printf("使用 'tick %d' 推进时间到窗口期。\n", waitDays)
		return
	}

	r.updateVesselState(v, newState)
	v.DepartureBody = targetID
	arrivalTime := r.game.CurrentTime.AddDate(0, 0, int(flightDays)+1)
	v.ArrivalTime = arrivalTime

	fmt.Printf("🚀 星际航行成功！\n")
	fmt.Printf("   航天器 %s 已到达 %s 高轨道 (高度 %.0f km)\n", v.Name, target.Name, v.OrbitAltitude)
	fmt.Printf("   消耗 Δv: %.0f m/s, 剩余: %.0f m/s\n", dv, v.DeltaVRemaining)
	fmt.Printf("   预计到达时间: %s (飞行约 %.0f 天)\n", arrivalTime.Format("2006-01-02"), flightDays)
}

// ==================== orbit release 命令 ====================
func (r *Repl) releaseCommand(vesselID string, bayIndexStr string, itemIndexStr string) {
	bayIndex, err := strconv.Atoi(bayIndexStr)
	if err != nil || bayIndex < 1 {
		fmt.Println("错误: 无效的货舱序号，请输入正整数")
		return
	}
	itemIndex, err := strconv.Atoi(itemIndexStr)
	if err != nil || itemIndex < 1 {
		fmt.Println("错误: 无效的货物索引，请输入正整数")
		return
	}

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
		fmt.Println("错误: 航天器已失效")
		return
	}

	bay, _ := r.getCargoBayByIndex(v, bayIndex)
	if bay == nil {
		fmt.Printf("错误: 未找到货舱 %d\n", bayIndex)
		fmt.Println("可用货舱:")
		for _, b := range v.CargoBays {
			fmt.Printf("  货舱 %d: %d 件货物\n", b.Index, len(b.Loaded))
		}
		return
	}
	if len(bay.Loaded) == 0 {
		fmt.Printf("货舱 %d 已空\n", bayIndex)
		return
	}

	item, itemIdx := r.getCargoItemByIndex(bay, itemIndex)
	if item == nil {
		fmt.Printf("错误: 未找到货物 %d\n", itemIndex)
		fmt.Printf("货舱 %d 中的货物:\n", bayIndex)
		for i, cargo := range bay.Loaded {
			fmt.Printf("  [%d] %s (%s)\n", i+1, r.getCargoName(cargo), cargo.Type)
		}
		return
	}

	cargoName := r.getCargoName(*item)
	cargoMass := r.getCargoMass(*item)

	bay.Loaded = append(bay.Loaded[:itemIdx], bay.Loaded[itemIdx+1:]...)

	newVesselID := fmt.Sprintf("cargo_%d", len(r.game.Vessels)+1)
	newVessel := domain.Vessel{
		ID:              newVesselID,
		Name:            cargoName,
		Type:            domain.VesselCargo,
		DesignID:        "",
		OrbitBodyID:     v.OrbitBodyID,
		OrbitAltitude:   v.OrbitAltitude,
		LaunchTime:      v.LaunchTime,
		Power:           500,
		MaxPower:        500,
		DataStored:      0,
		DataRate:        0,
		DeltaVRemaining: 0,
		IsActive:        true,
		IsAssembly:      false,
		DockedWith:      []string{},
		Modules:         []domain.SatelliteModule{},
		DepartureBody:   v.DepartureBody,
		CargoBays:       []domain.CargoBay{},
		HasControlCenter: false,
		Firmware:        []string{},
		HeliocentricPos: v.HeliocentricPos,
		HeliocentricVel: v.HeliocentricVel,
	}

	switch item.Type {
	case "rocket":
		for _, d := range r.game.RocketDesigns {
			if d.ID == item.ID {
				newVessel.Name = d.Name + " (火箭载荷)"
				newVessel.Modules = append(newVessel.Modules, domain.SatelliteModule{
					ID:           "rocket_payload",
					Name:         "火箭载荷: " + d.Name,
					Type:         "Structure",
					Mass:         d.PayloadMass,
					PowerConsume: 0,
					DataRate:     0,
					Cost:         0,
				})
				break
			}
		}
	case "satellite":
		for _, d := range r.game.SatelliteDesigns {
			if d.ID == item.ID {
				newVessel.Name = d.Name + " (卫星)"
				newVessel.Modules = d.Modules
				newVessel.DataRate = d.TotalDataRate
				newVessel.MaxPower = 500
				break
			}
		}
	case "part":
		if p, ok := r.game.PartsDB[item.ID]; ok {
			newVessel.Name = p.Name + " (零件)"
			newVessel.Modules = append(newVessel.Modules, domain.SatelliteModule{
				ID:           p.ID,
				Name:         p.Name,
				Type:         "Part",
				Mass:         p.MassDry,
				PowerConsume: 0,
				DataRate:     0,
				Cost:         p.Cost,
			})
		}
	default:
		fmt.Printf("警告: 未知货物类型 '%s'，释放后可能无法正常使用\n", item.Type)
	}

	if r.hasControlChip(&newVessel) {
		newVessel.HasControlCenter = true
		fmt.Printf("🔌 释放的货物包含航电，已激活控制中心\n")
	}

	r.game.Vessels = append(r.game.Vessels, newVessel)

	fmt.Printf("✅ 货物释放成功！\n")
	fmt.Printf("   货物: %s\n", cargoName)
	fmt.Printf("   质量: %.0f kg\n", cargoMass)
	fmt.Printf("   新实体 ID: %s\n", newVesselID)
	fmt.Printf("   位置: %s 轨道 (高度 %.0f km)\n",
		r.game.StarSystem.CelestialBodies[v.OrbitBodyID].Name, v.OrbitAltitude)
	fmt.Println("   可使用 'sat list' 查看新实体。")
	fmt.Println("\n💡 提示: 释放后的货物可使用 'sat list' 查看。")

	if len(bay.Loaded) == 0 {
		fmt.Printf("货舱 %d 已空。\n", bayIndex)
	}
}

// ==================== orbit flyby 命令 ====================

// flybyPlan 推荐下一个最佳飞掠目标
func (r *Repl) flybyPlan(vesselID string) {
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
		fmt.Println("航天器已失效")
		return
	}

	state, err := r.getFlybyState(v)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	if state.CurrentBodyID == "deep_space" {
		fmt.Println("航天器在深空，无法进行引力辅助（需要位于行星附近）")
		return
	}

	bestPlanet, bestDV, bestState, err := engine.PlanNextFlyby(state, r.game.StarSystem, r.game.CurrentTime, 0)
	if err != nil {
		fmt.Printf("规划失败: %v\n", err)
		return
	}
	planet := r.game.StarSystem.CelestialBodies[bestPlanet]
	fmt.Printf("推荐飞掠目标: %s\n", planet.Name)
	fmt.Printf("预计消耗 Δv: %.0f m/s\n", bestDV)
	fmt.Printf("飞掠后剩余 Δv: %.0f m/s\n", bestState.DeltaVRemaining)
	fmt.Printf("飞掠后位置: %s 附近\n", planet.Name)
	fmt.Println("\n使用 'flyby execute <航天器ID> <目标> [近心点高度]' 执行飞掠")
}

// flybyExecute 执行一次引力辅助
func (r *Repl) flybyExecute(vesselID, planetID string, periapsisStr string) {
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
		fmt.Println("航天器已失效")
		return
	}

	planet, ok := r.game.StarSystem.CelestialBodies[planetID]
	if !ok {
		fmt.Printf("错误: 未找到行星 '%s'\n", planetID)
		return
	}
	if planetID == v.OrbitBodyID {
		fmt.Println("错误: 不能对当前所在天体进行引力辅助")
		return
	}

	periapsis := planet.Radius + 500.0
	if periapsisStr != "" {
		p, err := strconv.ParseFloat(periapsisStr, 64)
		if err == nil && p > 0 {
			periapsis = p
			if periapsis < planet.Radius {
				fmt.Printf("警告: 近心点高度低于行星半径，自动调整为 %.0f km\n", planet.Radius+100)
				periapsis = planet.Radius + 100
			}
		} else {
			fmt.Println("警告: 无效的近心点高度，使用默认值")
		}
	}

	state, err := r.getFlybyState(v)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	newState, dv, err := engine.ComputeFlybyState(state, planet, periapsis, r.game.CurrentTime)
	if err != nil {
		fmt.Printf("飞掠计算失败: %v\n", err)
		return
	}

	r.updateVesselState(v, newState)
	v.FlybyHistory = append(v.FlybyHistory, planetID)

	fmt.Printf("✅ 引力辅助成功！\n")
	fmt.Printf("   行星: %s\n", planet.Name)
	fmt.Printf("   近心点高度: %.0f km\n", periapsis)
	fmt.Printf("   消耗 Δv: %.0f m/s\n", dv)
	fmt.Printf("   剩余 Δv: %.0f m/s\n", v.DeltaVRemaining)
	fmt.Printf("   新日心速度: (%.2f, %.2f) km/s\n", newState.Vel[0], newState.Vel[1])
	fmt.Printf("   当前位置: %s 附近\n", planet.Name)
	fmt.Println("\n💡 提示: 可以继续使用 'flyby plan' 规划下一次飞掠，或使用 'travel' 前往目标。")
}

// ==================== orbit 命令分发 ====================
func (r *Repl) handleOrbitCommand(subCmd string, args []string) {
	switch subCmd {
	case "info":
		if len(args) < 1 {
			fmt.Println("用法: orbit info <航天器ID>")
			return
		}
		r.orbitCommand(args[0])
	case "transfer":
		if len(args) < 2 {
			fmt.Println("用法: orbit transfer <航天器ID> <目标层索引或名称>")
			fmt.Println("  例如: orbit transfer sat_1 0 (转移到最低层)")
			fmt.Println("  例如: orbit transfer sat_1 高 (转移到最高层)")
			return
		}
		r.transferCommand(args[0], args[1])
	case "escape":
		if len(args) < 1 {
			fmt.Println("用法: orbit escape <航天器ID>")
			return
		}
		r.escapeCommand(args[0])
	case "dock":
		if len(args) < 2 {
			fmt.Println("用法: orbit dock <主航天器ID> <从航天器ID>")
			fmt.Println("  将两个航天器在轨道上对接，合并为组合体")
			return
		}
		r.dockCommand(args[0], args[1])
	case "travel":
		if len(args) < 2 {
			fmt.Println("用法: orbit travel <航天器ID> <目标天体ID>")
			fmt.Println("  从深空飞向目标天体（使用霍曼转移）")
			return
		}
		r.travelCommand(args[0], args[1])
	case "release":
		if len(args) < 3 {
			fmt.Println("用法: orbit release <航天器/组合体ID> <货舱序号> <货物索引>")
			fmt.Println("  从货舱释放货物，生成新的在轨实体")
			fmt.Println("  货物索引可通过 'orbit info <航天器ID>' 查看")
			return
		}
		r.releaseCommand(args[0], args[1], args[2])
	case "flyby":
		if len(args) < 2 {
			fmt.Println("用法: orbit flyby <子命令> [参数]")
			fmt.Println("  子命令: plan <航天器ID> - 推荐下一个最佳飞掠目标")
			fmt.Println("          execute <航天器ID> <行星ID> [近心点高度km] - 执行飞掠")
			return
		}
		switch args[0] {
		case "plan":
			if len(args) < 2 {
				fmt.Println("用法: orbit flyby plan <航天器ID>")
				return
			}
			r.flybyPlan(args[1])
		case "execute":
			if len(args) < 3 {
				fmt.Println("用法: orbit flyby execute <航天器ID> <行星ID> [近心点高度km]")
				return
			}
			alt := ""
			if len(args) >= 4 {
				alt = args[3]
			}
			r.flybyExecute(args[1], args[2], alt)
		default:
			fmt.Printf("未知 flyby 子命令 '%s'\n", args[0])
		}
	default:
		fmt.Println("未知轨道子命令")
		fmt.Println("  orbit info <航天器ID>     - 显示轨道信息（含日心状态）")
		fmt.Println("  orbit transfer <航天器ID> <目标层> - 变轨到指定轨道层")
		fmt.Println("  orbit escape <航天器ID>   - 弹射离开当前天体")
		fmt.Println("  orbit dock <主航天器ID> <从航天器ID> - 在轨对接（自动识别固件）")
		fmt.Println("  orbit travel <航天器ID> <目标天体ID> - 星际航行（霍曼转移）")
		fmt.Println("  orbit release <航天器ID> <货舱序号> <货物索引> - 从货舱释放货物")
		fmt.Println("  orbit flyby plan <航天器ID>          - 推荐下一个飞掠目标")
		fmt.Println("  orbit flyby execute <航天器ID> <行星ID> [高度] - 执行飞掠")
	}
}
