package repl

import (
	"fmt"
	"strings"
	"time"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
	"astrolex/internal/i18n"
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
	renderer := NewASCIIRenderer(design, len(stages))
	renderer.Update(0, 0, 0, 0, 0)
	renderer.SetStage(1)
	renderer.SetTailFlame(0)
	rocketFrame := renderer.Render()

	fmt.Println(rocketFrame)
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println(i18n.T("flight_log_title"))

	burnTimes := make([]float64, len(stages))
	for i, s := range stages {
		burnTimes[i] = engine.CalcStageBurnTime(s)
		if burnTimes[i] <= 0 {
			burnTimes[i] = 5
		}
	}

	type Event struct {
		time float64
		msg  string
	}
	var events []Event
	currentTime := 0.0

	events = append(events, Event{time: currentTime, msg: i18n.T("launch_engine_ignite")})

	for i := 0; i < len(stages); i++ {
		currentTime += burnTimes[i]
		if i < len(stages)-1 {
			events = append(events, Event{time: currentTime, msg: fmt.Sprintf(i18n.T("launch_stage_separate"), i+1)})
			currentTime += 1
			events = append(events, Event{time: currentTime, msg: fmt.Sprintf(i18n.T("launch_stage_ignite"), i+2)})
		} else {
			currentTime += 1
			events = append(events, Event{time: currentTime, msg: i18n.T("launch_orbital_insertion")})
		}
	}

	lastTime := 0.0
	for _, evt := range events {
		wait := evt.time - lastTime
		if wait > 0 {
			time.Sleep(time.Duration(wait * float64(time.Second)))
		}
		fmt.Printf("[T+%.0fs] %s\n", evt.time, evt.msg)
		lastTime = evt.time
	}

	fmt.Println("\n" + i18n.T("launch_mission_complete"))
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
		fmt.Printf(i18n.T("error_design_not_found"), designID)
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
			fmt.Println(i18n.T("error_contract_not_found"))
			return
		}
		if contract.Status != "Accepted" {
			fmt.Printf(i18n.T("error_contract_not_accepted"), contract.ID, contract.Status)
			return
		}
		targetID = contract.TargetBodyID
		if len(contract.ForbiddenSuppliers) > 0 {
			for _, mod := range design.Modules {
				if mod.Part != nil && contains(contract.ForbiddenSuppliers, mod.Part.SupplierID) {
					fmt.Printf(i18n.T("error_forbidden_supplier"), mod.Part.Name, mod.Part.SupplierID)
					return
				}
			}
		}
		if contract.MaxAccelLimit > 0 && design.MaxAccel > contract.MaxAccelLimit {
			fmt.Printf(i18n.T("error_accel_exceed"), design.MaxAccel, contract.MaxAccelLimit)
			return
		}
		fmt.Printf(i18n.T("launch_contract"), contract.ID, r.game.StarSystem.CelestialBodies[targetID].Name, contract.TargetPayloadDelivered, contract.PayloadMass)
	} else {
		if targetParam != "" {
			targetID = targetParam
		} else {
			targetID = "mars"
		}
		fmt.Printf(i18n.T("launch_no_contract"), r.game.StarSystem.CelestialBodies[targetID].Name)
	}

	targetBody, ok := r.game.StarSystem.CelestialBodies[targetID]
	if !ok {
		fmt.Printf(i18n.T("error_body_not_found"), targetID)
		return
	}

	targetLayerIndex := -1
	if orbitLayerParam != "" {
		targetLayerIndex = r.getOrbitLayerIndexByName(targetBody, orbitLayerParam)
		if targetLayerIndex == -1 {
			fmt.Printf(i18n.T("error_invalid_orbit_layer"), orbitLayerParam)
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
		fmt.Println(i18n.T("error_earth_not_found"))
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
			fmt.Printf(i18n.T("error_window_calc_failed"), err)
			return
		}
		current := r.game.CurrentTime
		if current.Before(windowStart) || current.After(windowEnd) {
			fmt.Printf(i18n.T("launch_window_not_in"))
			fmt.Printf(i18n.T("launch_window_next"), windowStart.Format("2006-01-02"))
			fmt.Printf(i18n.T("launch_window_wait"), waitDays)
			fmt.Printf(i18n.T("launch_window_tick"), waitDays)
			return
		}
		fmt.Println(i18n.T("launch_window_ok"))
	}

	fmt.Printf(i18n.T("launch_target"), targetBody.Name, targetLayer.Name, targetAltitude)
	fmt.Printf(i18n.T("launch_dv_breakdown"), earthLaunchDV, transferDV, captureDV)
	if isSatellite {
		fmt.Printf(i18n.T("launch_dv_breakdown_satellite"), localTransferDV)
	}
	fmt.Println()

	if design.DeltaV < requiredDV {
		fmt.Printf(i18n.T("launch_dv_insufficient"), design.DeltaV, requiredDV)
		return
	}
	fmt.Printf(i18n.T("launch_dv_sufficient"), design.DeltaV, requiredDV)

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
				fmt.Printf(i18n.T("launch_budget_short"))
				fmt.Printf(i18n.T("launch_cost"), totalCost)
				fmt.Printf(i18n.T("launch_budget_remaining"), remainingBudget)
				fmt.Printf(i18n.T("launch_budget_short_detail"), playerPortion, r.game.Player.Credits)
				return
			}
			if totalCost > 0 {
				ratio := float64(budgetPortion) / float64(totalCost)
				budgetHardwarePortion = int64(float64(hardwareCost) * ratio)
			} else {
				budgetHardwarePortion = 0
			}
			fmt.Printf(i18n.T("launch_budget_short_confirm"), playerPortion)
			fmt.Printf(i18n.T("launch_cost"), totalCost, hardwareCost, fuelCost, padCost)
			fmt.Printf(i18n.T("launch_budget_remaining"), remainingBudget)
			fmt.Print(i18n.T("launch_budget_short_confirm_prompt"))
			confirm, _ := r.reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			if confirm != "y" && confirm != "yes" {
				fmt.Println(i18n.T("launch_cancelled"))
				return
			}
		}

		contract.BudgetUsed += budgetPortion
		contract.BudgetHardwareCost += budgetHardwarePortion
		contract.TotalCost += totalCost
		if playerPortion > 0 {
			contract.PlayerPaid += playerPortion
			r.game.Player.Credits -= playerPortion
			fmt.Printf(i18n.T("launch_player_paid"), playerPortion)
		}
		fmt.Printf(i18n.T("launch_cost"), totalCost, hardwareCost, fuelCost, padCost)
		fmt.Printf(i18n.T("launch_budget_paid"), budgetPortion, playerPortion)
		fmt.Printf(i18n.T("launch_budget_remaining"), contract.Budget-contract.BudgetUsed)
		fmt.Printf(i18n.T("launch_credits"), r.game.Player.Credits)
	} else {
		fmt.Printf(i18n.T("launch_cost"), totalCost, hardwareCost, fuelCost, padCost)
		if r.game.Player.Credits < totalCost {
			fmt.Printf(i18n.T("launch_cost_insufficient"), totalCost, r.game.Player.Credits)
			return
		}
		r.game.Player.Credits -= totalCost
		fmt.Printf(i18n.T("launch_cost_deducted"), totalCost, r.game.Player.Credits)
	}

	stages, err := engine.PairModules(design.Modules)
	if err != nil {
		fmt.Printf(i18n.T("error_parse_design"), err)
		return
	}
	failed := engine.SimulateLaunchFailure(stages)

	if failed {
		fmt.Println(i18n.T("launch_failure"))
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
		mission.FailureReason = i18n.T("launch_failure_reason")
		r.game.Launches = append(r.game.Launches, mission)
		if contract != nil {
			contract.Launches = append(contract.Launches, mission)
			r.game.Player.Credits -= contract.PenaltyCredits
			if r.game.Player.Credits < 0 {
				r.game.Player.Credits = 0
			}
			contract.Status = "Failed"
			fmt.Printf(i18n.T("contract_penalty"), contract.PenaltyCredits)
		}
		fmt.Printf(i18n.T("launch_credits"), r.game.Player.Credits)
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
		mission.FailureReason = i18n.T("launch_failure_reason")
	}
	r.game.Launches = append(r.game.Launches, mission)

	if contract != nil {
		contract.Launches = append(contract.Launches, mission)
		if success {
			contract.TargetPayloadDelivered += design.PayloadMass
			fmt.Printf(i18n.T("launch_success_contract"), design.PayloadMass)
		} else {
			fmt.Println(i18n.T("launch_failure"))
		}
	} else {
		if success {
			fmt.Println(i18n.T("launch_success"))
			r.game.Player.Credits += 10000
			fmt.Printf(i18n.T("launch_reward"), r.game.Player.Credits)
		} else {
			fmt.Println(i18n.T("launch_failure"))
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
					fmt.Printf(i18n.T("deploy_satellite"), satDesign.Name, targetBody.Name, targetLayer.Name, newVessel.OrbitAltitude)
					fmt.Printf(i18n.T("deploy_remaining_dv"), leftoverDV)
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
			fmt.Printf(i18n.T("deploy_assembly"), design.Name, targetBody.Name, targetLayer.Name, targetAltitude, len(design.CargoBays))
			fmt.Printf(i18n.T("deploy_remaining_dv"), leftoverDV)
			if hasControl {
				fmt.Println(i18n.T("deploy_control_active"))
			} else {
				fmt.Println(i18n.T("deploy_control_inactive"))
			}
			fmt.Println(i18n.T("deploy_release_hint"))
		}
	}
}

// windowCommand 显示从地球到目标天体的发射窗口
func (r *Repl) windowCommand(targetID string) {
	target, ok := r.game.StarSystem.CelestialBodies[targetID]
	if !ok {
		fmt.Printf(i18n.T("error_body_not_found"), targetID)
		return
	}
	earth, ok := r.game.StarSystem.CelestialBodies["earth"]
	if !ok {
		fmt.Println(i18n.T("error_earth_not_found"))
		return
	}
	if targetID == "earth" {
		fmt.Println(i18n.T("window_earth"))
		return
	}

	windowBody := target
	if target.ParentID != "" && target.ParentID != "sol" {
		if parent, ok := r.game.StarSystem.CelestialBodies[target.ParentID]; ok {
			windowBody = parent
			fmt.Printf(i18n.T("window_satellite_note"), target.Name, parent.Name, parent.Name)
		}
	}

	windowStart, windowEnd, waitDays, dv, err := engine.NextWindow(earth, windowBody, r.game.CurrentTime, r.cfg.Physics.WindowSearchDays)
	if err != nil {
		fmt.Printf(i18n.T("error_window_calc_failed"), err)
		return
	}
	fmt.Printf(i18n.T("window_title"), target.Name)
	fmt.Printf(i18n.T("window_target"), target.Name)
	fmt.Printf(i18n.T("window_current_time"), r.game.CurrentTime.Format("2006-01-02"))
	fmt.Printf(i18n.T("window_start"), windowStart.Format("2006-01-02"))
	fmt.Printf(i18n.T("window_end"), windowEnd.Format("2006-01-02"))
	fmt.Printf(i18n.T("window_wait_days"), waitDays)
	fmt.Printf(i18n.T("window_dv"), dv*1000)
	fmt.Println(i18n.T("window_note"))
}
