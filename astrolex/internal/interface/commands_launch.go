package repl

import (
	"fmt"
	"strings"
	"time"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
)

// getHighestOrbitRadius 返回目标天体的最高轨道层典型半径
func (r *Repl) getHighestOrbitRadius(body domain.CelestialBody) float64 {
	if len(body.OrbitalLayers) == 0 {
		return body.Radius + body.HillSphereRadius*0.1
	}
	topLayer := body.OrbitalLayers[len(body.OrbitalLayers)-1]
	return body.Radius + topLayer.TypicalAltitude
}

// getOrbitLayerIndexByName 根据名称获取轨道层索引
func (r *Repl) getOrbitLayerIndexByName(body domain.CelestialBody, name string) int {
	name = strings.ToLower(name)
	for i, layer := range body.OrbitalLayers {
		lowerName := strings.ToLower(layer.Name)
		if strings.Contains(lowerName, name) || strings.Contains(name, lowerName) {
			return i
		}
	}
	if name == "0" {
		return 0
	} else if name == "1" {
		return 1
	} else if name == "2" {
		return 2
	}
	return -1
}

// getDeltaVToOrbitLayer 计算从地表到达指定轨道层所需的总 Δv (m/s)
func (r *Repl) getDeltaVToOrbitLayer(body domain.CelestialBody, layerIndex int) float64 {
	if len(body.OrbitalLayers) == 0 {
		return body.DeltaVToLowOrbit
	}
	if layerIndex < 0 {
		layerIndex = 0
	}
	if layerIndex >= len(body.OrbitalLayers) {
		layerIndex = len(body.OrbitalLayers) - 1
	}
	layers := body.OrbitalLayers
	var totalDeltaV float64 = body.DeltaVToLowOrbit
	prevRadius := body.Radius + layers[0].TypicalAltitude
	for i := 1; i <= layerIndex; i++ {
		curRadius := body.Radius + layers[i].TypicalAltitude
		dv_kmps := engine.OrbitalLayerDeltaV(body.GM, prevRadius, curRadius)
		totalDeltaV += dv_kmps * 1000
		prevRadius = curRadius
	}
	return totalDeltaV
}

// getParentBody 返回天体的母行星（如果是卫星）
func (r *Repl) getParentBody(body domain.CelestialBody) domain.CelestialBody {
	if body.ParentID != "" && body.ParentID != "sol" {
		if parent, ok := r.game.StarSystem.CelestialBodies[body.ParentID]; ok {
			return parent
		}
	}
	return body
}

// hasControlChipInDesign 检查火箭设计中是否包含航电（控制芯片）
func (r *Repl) hasControlChipInDesign(design *domain.RocketDesign) bool {
	for _, mod := range design.Modules {
		if mod.Part != nil && mod.Part.Category == "Avionics" {
			if part, ok := r.game.PartsDB[mod.PartID]; ok && part.IsControlChip {
				return true
			}
		}
	}
	return false
}

// hasControlChip 检查航天器是否包含航电（控制芯片）
func (r *Repl) hasControlChip(v *domain.Vessel) bool {
	for _, mod := range v.Modules {
		if mod.Type == "Avionics" {
			if part, ok := r.game.PartsDB[mod.ID]; ok && part.IsControlChip {
				return true
			}
		}
	}
	return false
}

// simulateFlightWithASCII 发射模拟：静态火箭画面 + 逐行打印日志
func (r *Repl) simulateFlightWithASCII(design *domain.RocketDesign, stages []domain.Stage, requiredDV float64, targetName string) bool {
	// 1. 生成静态火箭画面（含初始遥测数据）
	renderer := NewASCIIRenderer(design, len(stages))
	renderer.Update(0, 0, 0, 0, 0) // 初始状态（高度、速度、加速度、时间、Δv）
	renderer.SetStage(1)
	renderer.SetTailFlame(0) // 静态画面无尾焰
	rocketFrame := renderer.Render()

	// 打印火箭画面
	fmt.Println(rocketFrame)
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println("📋 飞行日志:")

	// 2. 预计算每级燃烧时间和事件时间线
	burnTimes := make([]float64, len(stages))
	for i, s := range stages {
		burnTimes[i] = engine.CalcStageBurnTime(s)
		if burnTimes[i] <= 0 {
			burnTimes[i] = 5 // 默认5秒
		}
	}

	type Event struct {
		time float64
		msg  string
	}
	var events []Event
	currentTime := 0.0

	// 点火
	events = append(events, Event{time: currentTime, msg: "🚀 引擎点火，火箭升空！"})

	for i := 0; i < len(stages); i++ {
		currentTime += burnTimes[i]
		if i < len(stages)-1 {
			// 级分离
			events = append(events, Event{time: currentTime, msg: fmt.Sprintf("⏹ 第 %d 级燃料耗尽，关机分离", i+1)})
			currentTime += 1 // 间隔1秒
			events = append(events, Event{time: currentTime, msg: fmt.Sprintf("🔥 第 %d 级引擎点火", i+2)})
		} else {
			// 最后一级关机入轨
			currentTime += 1
			events = append(events, Event{time: currentTime, msg: "✅ 载荷成功入轨！"})
		}
	}

	// 3. 按顺序打印事件，每行间休眠
	lastTime := 0.0
	for _, evt := range events {
		wait := evt.time - lastTime
		if wait > 0 {
			time.Sleep(time.Duration(wait * float64(time.Second)))
		}
		fmt.Printf("[T+%.0fs] %s\n", evt.time, evt.msg)
		lastTime = evt.time
	}

	fmt.Println("\n🚀 任务完成！")
	return true
}

// launchRocket 执行火箭发射，支持指定目标天体、轨道层
func (r *Repl) launchRocket(designID string, targetParam string, orbitLayerParam string) {
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

	var targetID string
	var contract *domain.Contract
	if r.activeContractID != "" {
		for i := range r.game.Contracts {
			if r.game.Contracts[i].ID == r.activeContractID {
				contract = &r.game.Contracts[i]
				break
			}
		}
		if contract == nil {
			fmt.Println("错误: 当前合约不存在或已结束")
			return
		}
		if contract.Status != "Accepted" {
			fmt.Printf("合约 %s 状态为 '%s'，无法发射\n", contract.ID, contract.Status)
			return
		}
		targetID = contract.TargetBodyID
		if len(contract.ForbiddenSuppliers) > 0 {
			for _, mod := range design.Modules {
				if mod.Part != nil && contains(contract.ForbiddenSuppliers, mod.Part.SupplierID) {
					fmt.Printf("错误: 零件 %s 来自禁用供应商 %s\n", mod.Part.Name, mod.Part.SupplierID)
					return
				}
			}
		}
		if contract.MaxAccelLimit > 0 && design.MaxAccel > contract.MaxAccelLimit {
			fmt.Printf("错误: 加速度 %.2fG 超过限制 %.2fG\n", design.MaxAccel, contract.MaxAccelLimit)
			return
		}
		fmt.Printf("当前合约: %s (目标: %s, 已送达载荷: %.0f kg / %.0f kg)\n", contract.ID, r.game.StarSystem.CelestialBodies[targetID].Name, contract.TargetPayloadDelivered, contract.PayloadMass)
	} else {
		if targetParam != "" {
			targetID = targetParam
		} else {
			targetID = "mars"
		}
		fmt.Printf("无合约，测试发射目标: %s\n", r.game.StarSystem.CelestialBodies[targetID].Name)
	}

	targetBody, ok := r.game.StarSystem.CelestialBodies[targetID]
	if !ok {
		fmt.Printf("错误: 未找到目标天体 '%s'\n", targetID)
		return
	}

	targetLayerIndex := -1
	if orbitLayerParam != "" {
		targetLayerIndex = r.getOrbitLayerIndexByName(targetBody, orbitLayerParam)
		if targetLayerIndex == -1 {
			fmt.Printf("错误: 未知轨道层 '%s'，可用: low, medium, high (或 0,1,2)\n", orbitLayerParam)
			return
		}
	} else {
		targetLayerIndex = len(targetBody.OrbitalLayers) - 1
		if targetLayerIndex < 0 {
			targetLayerIndex = 0
		}
	}
	targetLayer := targetBody.OrbitalLayers[targetLayerIndex]
	targetAltitude := targetLayer.TypicalAltitude

	earth, ok := r.game.StarSystem.CelestialBodies["earth"]
	if !ok {
		fmt.Println("错误: 未找到地球数据")
		return
	}

	earthHighIndex := len(earth.OrbitalLayers) - 1
	earthLaunchDV := r.getDeltaVToOrbitLayer(earth, earthHighIndex)

	var transferDV float64
	var captureDV float64
	var localTransferDV float64
	isSatellite := (targetBody.ParentID != "" && targetBody.ParentID != "sol")

	if isSatellite {
		parentBody := r.getParentBody(targetBody)
		earthOrbitRadius := earth.SemiMajorAxis
		parentOrbitRadius := parentBody.SemiMajorAxis
		transferDV_kmps := engine.OrbitalLayerDeltaV(engine.SunGM, earthOrbitRadius, parentOrbitRadius)
		transferDV = transferDV_kmps * 1000

		parentHighRadius := r.getHighestOrbitRadius(parentBody)
		captureDV_kmps := engine.OrbitalLayerDeltaV(parentBody.GM, parentOrbitRadius, parentHighRadius)
		captureDV = captureDV_kmps * 1000

		satHighRadius := r.getHighestOrbitRadius(targetBody)
		localDV_kmps := engine.OrbitalLayerDeltaV(parentBody.GM, parentHighRadius, satHighRadius)
		localTransferDV = localDV_kmps * 1000
	} else {
		earthOrbitRadius := earth.SemiMajorAxis
		targetOrbitRadius := targetBody.SemiMajorAxis
		transferDV_kmps := engine.OrbitalLayerDeltaV(engine.SunGM, earthOrbitRadius, targetOrbitRadius)
		transferDV = transferDV_kmps * 1000

		targetHighRadius := r.getHighestOrbitRadius(targetBody)
		captureDV_kmps := engine.OrbitalLayerDeltaV(targetBody.GM, targetOrbitRadius, targetHighRadius)
		captureDV = captureDV_kmps * 1000
	}

	requiredDV := earthLaunchDV + transferDV + captureDV + localTransferDV

	isEarthTarget := (targetID == "earth" || targetID == "leo")
	if !isEarthTarget {
		windowBody := targetBody
		if isSatellite {
			windowBody = r.getParentBody(targetBody)
		}
		windowStart, windowEnd, waitDays, _, err := engine.NextWindow(earth, windowBody, r.game.CurrentTime, r.cfg.Physics.WindowSearchDays)
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
		fmt.Println("✅ 当前时间处于发射窗口内。")
	}

	fmt.Printf("目标: %s 的 %s (高度 ~%.0f km)\n", targetBody.Name, targetLayer.Name, targetAltitude)
	fmt.Printf("所需 Δv 分解: 地球发射 %.0f m/s + 星际转移 %.0f m/s + 捕获 %.0f m/s",
		earthLaunchDV, transferDV, captureDV)
	if isSatellite {
		fmt.Printf(" + 本地转移 %.0f m/s", localTransferDV)
	}
	fmt.Println()

	if design.DeltaV < requiredDV {
		fmt.Printf("❌ 火箭 Δv 不足: 设计 %.0f m/s, 需要 %.0f m/s\n", design.DeltaV, requiredDV)
		return
	}
	fmt.Printf("✅ 火箭 Δv 充足: %.0f m/s (需要 %.0f m/s)\n", design.DeltaV, requiredDV)

	hardwareCost := r.calcHardwareCost(*design)
	var fuelMass float64
	for _, mod := range design.Modules {
		if mod.Type == domain.ModuleFuelTank && mod.Part != nil {
			fuelMass += mod.Part.MassFuelMax * 0.5
		}
	}
	fuelCost := int64(fuelMass * r.cfg.Physics.FuelPricePerKg)
	padCost := r.cfg.Physics.LaunchPadFee
	totalCost := hardwareCost + fuelCost + padCost

	if contract != nil {
		remainingBudget := contract.Budget - contract.BudgetUsed
		var budgetPortion int64
		var playerPortion int64
		var budgetHardwarePortion int64

		if totalCost <= remainingBudget {
			budgetPortion = totalCost
			playerPortion = 0
			budgetHardwarePortion = hardwareCost
		} else {
			budgetPortion = remainingBudget
			playerPortion = totalCost - remainingBudget
			if r.game.Player.Credits < playerPortion {
				fmt.Printf("❌ 预算不足，且玩家信用点不足以支付差额！\n")
				fmt.Printf("   本次发射总成本: %d 信用点\n", totalCost)
				fmt.Printf("   剩余预算: %d 信用点\n", remainingBudget)
				fmt.Printf("   需要玩家支付: %d 信用点, 当前仅有: %d 信用点\n", playerPortion, r.game.Player.Credits)
				return
			}
			if totalCost > 0 {
				ratio := float64(budgetPortion) / float64(totalCost)
				budgetHardwarePortion = int64(float64(hardwareCost) * ratio)
			} else {
				budgetHardwarePortion = 0
			}
			fmt.Printf("⚠️ 预算不足，需要超支 %d 信用点\n", playerPortion)
			fmt.Printf("   本次发射总成本: %d 信用点 (硬件 %d + 燃料 %d + 场地 %d)\n", totalCost, hardwareCost, fuelCost, padCost)
			fmt.Printf("   剩余预算: %d 信用点\n", remainingBudget)
			fmt.Print("   确认继续发射？(y/n): ")
			confirm, _ := r.reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			if confirm != "y" && confirm != "yes" {
				fmt.Println("发射已取消。")
				return
			}
		}

		contract.BudgetUsed += budgetPortion
		contract.BudgetHardwareCost += budgetHardwarePortion
		contract.TotalCost += totalCost
		if playerPortion > 0 {
			contract.PlayerPaid += playerPortion
			r.game.Player.Credits -= playerPortion
			fmt.Printf("从个人账户扣除 %d 信用点 (超支部分)\n", playerPortion)
		}
		fmt.Printf("本次发射成本: %d 信用点 (硬件 %d + 燃料 %d + 场地 %d)\n", totalCost, hardwareCost, fuelCost, padCost)
		fmt.Printf("预算支付: %d, 个人支付: %d\n", budgetPortion, playerPortion)
		fmt.Printf("剩余预算: %d 信用点\n", contract.Budget-contract.BudgetUsed)
		fmt.Printf("当前个人信用点: %d\n", r.game.Player.Credits)
	} else {
		fmt.Printf("测试发射成本: %d 信用点 (硬件 %d + 燃料 %d + 场地 %d)\n", totalCost, hardwareCost, fuelCost, padCost)
		if r.game.Player.Credits < totalCost {
			fmt.Printf("❌ 信用点不足! 需要 %d, 当前 %d\n", totalCost, r.game.Player.Credits)
			return
		}
		r.game.Player.Credits -= totalCost
		fmt.Printf("扣除 %d 信用点，剩余 %d\n", totalCost, r.game.Player.Credits)
	}

	stages, err := engine.PairModules(design.Modules)
	if err != nil {
		fmt.Printf("错误: 无法解析设计: %v\n", err)
		return
	}
	failed := engine.SimulateLaunchFailure(stages)

	if failed {
		fmt.Println("\n💥 发射失败！零件故障导致任务中止。")
		mission := domain.LaunchMission{
			ID:           fmt.Sprintf("mission_%d", len(r.game.Launches)+1),
			ContractID:   r.activeContractID,
			RocketDesign: *design,
			BaseID:       "",
			LaunchTime:   r.game.CurrentTime,
			Success:      false,
			FinalDeltaV:  design.DeltaV,
			Redundancies: []string{},
		}
		mission.FailureReason = "零件故障导致发射失败"
		r.game.Launches = append(r.game.Launches, mission)
		if contract != nil {
			contract.Launches = append(contract.Launches, mission)
			r.game.Player.Credits -= contract.PenaltyCredits
			if r.game.Player.Credits < 0 {
				r.game.Player.Credits = 0
			}
			contract.Status = "Failed"
			fmt.Printf("扣除 %d 信用点罚金\n", contract.PenaltyCredits)
		}
		fmt.Printf("当前信用点: %d\n", r.game.Player.Credits)
		return
	}

	success := r.simulateFlightWithASCII(design, stages, requiredDV, targetBody.Name)

	mission := domain.LaunchMission{
		ID:           fmt.Sprintf("mission_%d", len(r.game.Launches)+1),
		ContractID:   r.activeContractID,
		RocketDesign: *design,
		BaseID:       "",
		LaunchTime:   r.game.CurrentTime,
		Success:      success,
		FinalDeltaV:  design.DeltaV,
		Redundancies: []string{},
	}
	if !success {
		mission.FailureReason = "发射失败"
	}
	r.game.Launches = append(r.game.Launches, mission)

	if contract != nil {
		contract.Launches = append(contract.Launches, mission)
		if success {
			contract.TargetPayloadDelivered += design.PayloadMass
			fmt.Printf("本次发射成功，已送达载荷 %.0f kg\n", design.PayloadMass)
		} else {
			fmt.Println("本次发射失败，未增加送达载荷")
		}
	} else {
		if success {
			fmt.Println("测试发射成功")
			r.game.Player.Credits += 10000
			fmt.Printf("测试发射奖励：+10000 信用点（当前信用点: %d）\n", r.game.Player.Credits)
		} else {
			fmt.Println("测试发射失败")
		}
	}

	if success {
		leftoverDV := design.DeltaV - requiredDV
		if leftoverDV < 0 {
			leftoverDV = 0
		}

		targetPosX, targetPosY := engine.PositionAtTime(targetBody, r.game.CurrentTime)
		targetVelX, targetVelY := engine.OrbitalVelocity(targetBody, r.game.CurrentTime)

		for _, mod := range design.Modules {
			if mod.Part != nil && mod.Part.Category == "Satellite" && mod.Part.SatelliteID != "" {
				var satDesign *domain.SatelliteDesign
				for _, sd := range r.game.SatelliteDesigns {
					if sd.ID == mod.Part.SatelliteID {
						satDesign = &sd
						break
					}
				}
				if satDesign != nil {
					newVessel := domain.Vessel{
						ID:              fmt.Sprintf("vessel_%d", len(r.game.Vessels)+1),
						Name:            satDesign.Name,
						Type:            domain.VesselSingle,
						DesignID:        satDesign.ID,
						OrbitBodyID:     targetID,
						OrbitAltitude:   targetAltitude,
						LaunchTime:      r.game.CurrentTime,
						Power:           r.cfg.Satellite.InitialPowerWh,
						MaxPower:        r.cfg.Satellite.MaxPowerWh,
						DataStored:      0,
						DataRate:        satDesign.TotalDataRate,
						DeltaVRemaining: leftoverDV,
						IsActive:        true,
						IsAssembly:      false,
						DockedWith:      []string{},
						Modules:         satDesign.Modules,
						DepartureBody:   "",
						CargoBays:       []domain.CargoBay{},
						HasControlCenter: false,
						Firmware:        []string{},
						ECCLPrograms:    []domain.ECCLProgram{},
						ActiveProgramID: "",
						HeliocentricPos: [2]float64{targetPosX, targetPosY},
						HeliocentricVel: [2]float64{targetVelX, targetVelY},
						FlybyHistory:    []string{},
					}
					if r.hasControlChip(&newVessel) {
						newVessel.HasControlCenter = true
					}
					r.game.Vessels = append(r.game.Vessels, newVessel)
					fmt.Printf("🛰️  航天器 '%s' 已成功部署到 %s 的 %s (高度 %.0f km)\n",
						satDesign.Name, targetBody.Name, targetLayer.Name, newVessel.OrbitAltitude)
					fmt.Printf("   剩余 Δv: %.0f m/s\n", leftoverDV)
				}
			}
		}

		if len(design.CargoBays) > 0 {
			cargoBaysCopy := make([]domain.CargoBay, len(design.CargoBays))
			for i, bay := range design.CargoBays {
				cargoBaysCopy[i] = domain.CargoBay{
					Index:  bay.Index,
					Loaded: bay.Loaded,
				}
			}
			hasControl := r.hasControlChipInDesign(design)

			mainVessel := domain.Vessel{
				ID:              fmt.Sprintf("main_%d", len(r.game.Vessels)+1),
				Name:            design.Name + " (组合体)",
				Type:            domain.VesselComposite,
				DesignID:        "",
				OrbitBodyID:     targetID,
				OrbitAltitude:   targetAltitude,
				LaunchTime:      r.game.CurrentTime,
				Power:           1000,
				MaxPower:        1000,
				DataStored:      0,
				DataRate:        0,
				DeltaVRemaining: leftoverDV,
				IsActive:        true,
				IsAssembly:      true,
				DockedWith:      []string{},
				Modules:         []domain.SatelliteModule{},
				DepartureBody:   "",
				CargoBays:       cargoBaysCopy,
				HasControlCenter: hasControl,
				Firmware:        []string{},
				ECCLPrograms:    []domain.ECCLProgram{},
				ActiveProgramID: "",
				HeliocentricPos: [2]float64{targetPosX, targetPosY},
				HeliocentricVel: [2]float64{targetVelX, targetVelY},
				FlybyHistory:    []string{},
			}
			r.game.Vessels = append(r.game.Vessels, mainVessel)
			fmt.Printf("🛰️  组合体 '%s' 已成功部署到 %s 的 %s (高度 %.0f km)，携带 %d 个货舱\n",
				design.Name, targetBody.Name, targetLayer.Name, targetAltitude, len(design.CargoBays))
			fmt.Printf("   剩余 Δv: %.0f m/s\n", leftoverDV)
			if hasControl {
				fmt.Println("   ✅ 控制中心已激活（包含航电）")
			} else {
				fmt.Println("   ❌ 无控制中心（缺少航电）")
			}
			fmt.Println("   货舱内货物可通过 'orbit release' 释放。")
		}
	}
}

// windowCommand 显示从地球到目标天体的发射窗口
func (r *Repl) windowCommand(targetID string) {
	target, ok := r.game.StarSystem.CelestialBodies[targetID]
	if !ok {
		fmt.Printf("错误: 未找到天体 '%s'\n", targetID)
		return
	}
	earth, ok := r.game.StarSystem.CelestialBodies["earth"]
	if !ok {
		fmt.Println("错误: 未找到地球数据")
		return
	}
	if targetID == "earth" {
		fmt.Println("目标为地球，无需发射窗口。可直接发射进入地球轨道。")
		return
	}

	windowBody := target
	if target.ParentID != "" && target.ParentID != "sol" {
		if parent, ok := r.game.StarSystem.CelestialBodies[target.ParentID]; ok {
			windowBody = parent
			fmt.Printf("注意: %s 是 %s 的卫星，窗口与 %s 相同\n", target.Name, parent.Name, parent.Name)
		}
	}

	windowStart, windowEnd, waitDays, dv, err := engine.NextWindow(earth, windowBody, r.game.CurrentTime, r.cfg.Physics.WindowSearchDays)
	if err != nil {
		fmt.Printf("计算窗口失败: %v\n", err)
		return
	}
	fmt.Printf("\n=== 发射窗口: 地球 -> %s ===\n", target.Name)
	fmt.Printf("目标天体: %s\n", target.Name)
	fmt.Printf("当前游戏时间: %s\n", r.game.CurrentTime.Format("2006-01-02"))
	fmt.Printf("下一个窗口开始: %s\n", windowStart.Format("2006-01-02"))
	fmt.Printf("窗口结束: %s\n", windowEnd.Format("2006-01-02"))
	fmt.Printf("需要等待: %d 天\n", waitDays)
	fmt.Printf("霍曼转移所需 Δv: %.0f m/s\n", dv*1000)
	fmt.Println("(窗口宽度约为 ±1 天，实际可允许容差较小)")
}
