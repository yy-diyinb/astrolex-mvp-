package repl

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
	"astrolex/internal/i18n"
)

// ==================== 辅助函数 ====================

// getOrbitLayerName 根据高度返回轨道层名称
func (r *Repl) getOrbitLayerName(body domain.CelestialBody, altitude float64) string {
	if len(body.OrbitalLayers) == 0 {
		return i18n.T("orbit_layer_unknown")
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
		return i18n.T("orbit_layer_high")
	}
	return i18n.T("orbit_layer_low")
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
	return i18n.T("cargo_unknown")
}

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
			return nil, fmt.Errorf(i18n.T("error_body_not_found"), v.OrbitBodyID)
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}

	if v.OrbitBodyID == "deep_space" {
		fmt.Printf(i18n.T("orbit_info_deep_space_title"), v.Name, v.ID)
		fmt.Printf(i18n.T("orbit_info_type"), v.Type)
		fmt.Printf(i18n.T("orbit_info_departure"), v.DepartureBody)
		fmt.Printf(i18n.T("orbit_info_heliocentric_pos"), v.HeliocentricPos[0], v.HeliocentricPos[1])
		fmt.Printf(i18n.T("orbit_info_heliocentric_vel"), v.HeliocentricVel[0], v.HeliocentricVel[1])
		fmt.Printf(i18n.T("orbit_info_dv_remaining"), v.DeltaVRemaining)
		fmt.Printf(i18n.T("orbit_info_data"), v.DataStored)
		fmt.Printf(i18n.T("orbit_info_power"), v.Power)
		if !v.ArrivalTime.IsZero() {
			fmt.Printf(i18n.T("orbit_info_arrival"), v.ArrivalTime.Format("2006-01-02"))
		}
		if len(v.CargoBays) > 0 {
			fmt.Println(i18n.T("orbit_info_cargo"))
			for _, bay := range v.CargoBays {
				fmt.Printf(i18n.T("orbit_info_cargo_bay"), bay.Index, len(bay.Loaded))
				for i, item := range bay.Loaded {
					fmt.Printf(i18n.T("orbit_info_cargo_item"), i+1, r.getCargoName(item), item.Type)
				}
			}
		}
		fmt.Printf(i18n.T("orbit_info_control"), map[bool]string{true: i18n.T("sat_yes"), false: i18n.T("sat_no")}[v.HasControlCenter])
		if len(v.Firmware) > 0 {
			fmt.Printf(i18n.T("orbit_info_firmware"), len(v.Firmware))
			for _, fw := range v.Firmware {
				for _, vv := range r.game.Vessels {
					if vv.ID == fw {
						fmt.Printf(i18n.T("orbit_info_firmware_item"), vv.Name)
						break
					}
				}
			}
		}
		return
	}

	body, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]
	if !ok {
		fmt.Println(i18n.T("error_body_not_found"))
		return
	}

	layerName := r.getOrbitLayerName(body, v.OrbitAltitude)
	radius := body.Radius + v.OrbitAltitude
	v_c := engine.CircularOrbitVelocity(body.GM, radius)
	v_esc := engine.EscapeVelocity(body.GM, radius)

	fmt.Printf(i18n.T("orbit_info_title"), v.Name)
	fmt.Printf(i18n.T("orbit_info_type"), v.Type)
	fmt.Printf(i18n.T("orbit_info_body"), body.Name)
	fmt.Printf(i18n.T("orbit_info_altitude"), v.OrbitAltitude)
	fmt.Printf(i18n.T("orbit_info_layer"), layerName)
	fmt.Printf(i18n.T("orbit_info_radius"), radius)
	fmt.Printf(i18n.T("orbit_info_circular_velocity"), v_c)
	fmt.Printf(i18n.T("orbit_info_escape_velocity"), v_esc)
	fmt.Printf(i18n.T("orbit_info_dv_remaining"), v.DeltaVRemaining)
	fmt.Printf(i18n.T("orbit_info_heliocentric_pos"), v.HeliocentricPos[0], v.HeliocentricPos[1])
	fmt.Printf(i18n.T("orbit_info_heliocentric_vel"), v.HeliocentricVel[0], v.HeliocentricVel[1])
	if v.OrbitAltitude > 0 {
		fmt.Printf(i18n.T("orbit_info_escape_dv"), engine.EscapeFromOrbitDeltaV(body.GM, radius)*1000)
	}
	if v.IsAssembly {
		fmt.Printf(i18n.T("orbit_info_assembly"), len(v.DockedWith))
		fmt.Printf(i18n.T("orbit_info_docked_with"), strings.Join(v.DockedWith, ", "))
	}
	if len(v.CargoBays) > 0 {
		fmt.Println(i18n.T("orbit_info_cargo"))
		for _, bay := range v.CargoBays {
			fmt.Printf(i18n.T("orbit_info_cargo_bay"), bay.Index, len(bay.Loaded))
			for i, item := range bay.Loaded {
				fmt.Printf(i18n.T("orbit_info_cargo_item"), i+1, r.getCargoName(item), item.Type)
			}
		}
	}
	fmt.Printf(i18n.T("orbit_info_control"), map[bool]string{true: i18n.T("sat_yes"), false: i18n.T("sat_no")}[v.HasControlCenter])
	if len(v.Firmware) > 0 {
		fmt.Printf(i18n.T("orbit_info_firmware"), len(v.Firmware))
		for _, fw := range v.Firmware {
			for _, vv := range r.game.Vessels {
				if vv.ID == fw {
					fmt.Printf(i18n.T("orbit_info_firmware_item"), vv.Name)
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}
	if v.OrbitBodyID == "deep_space" {
		fmt.Println(i18n.T("orbit_transfer_deep_space"))
		return
	}

	body, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]
	if !ok {
		fmt.Println(i18n.T("error_body_not_found"))
		return
	}
	if len(body.OrbitalLayers) == 0 {
		fmt.Println(i18n.T("orbit_transfer_no_layers"))
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
		fmt.Printf(i18n.T("orbit_transfer_invalid_layer"), targetLayer)
		fmt.Print(i18n.T("orbit_transfer_available_layers"))
		for i, layer := range body.OrbitalLayers {
			fmt.Printf("%d:%s ", i, layer.Name)
		}
		fmt.Println()
		return
	}

	currentIndex := r.getOrbitLayerIndex(body, v.OrbitAltitude)
	if currentIndex == targetIndex {
		fmt.Printf(i18n.T("orbit_transfer_already_at"), body.OrbitalLayers[targetIndex].Name)
		return
	}

	currentRadius := body.Radius + v.OrbitAltitude
	targetRadius := body.Radius + body.OrbitalLayers[targetIndex].TypicalAltitude
	dv_kmps := engine.OrbitalLayerDeltaV(body.GM, currentRadius, targetRadius)
	dv_mps := dv_kmps * 1000

	fmt.Printf(i18n.T("orbit_transfer_info"), v.OrbitAltitude, body.OrbitalLayers[targetIndex].Name, body.OrbitalLayers[targetIndex].TypicalAltitude)
	fmt.Printf(i18n.T("orbit_transfer_dv"), dv_mps)

	if v.DeltaVRemaining < dv_mps {
		fmt.Printf(i18n.T("orbit_transfer_dv_insufficient"), dv_mps, v.DeltaVRemaining)
		return
	}
	if v.Power < 50 {
		fmt.Printf(i18n.T("orbit_transfer_power_low"), 50)
		return
	}

	v.DeltaVRemaining -= dv_mps
	v.Power -= 50
	v.OrbitAltitude = body.OrbitalLayers[targetIndex].TypicalAltitude
	fmt.Printf(i18n.T("orbit_transfer_success"), v.OrbitAltitude, v.DeltaVRemaining)
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}
	if v.OrbitBodyID == "deep_space" {
		fmt.Println(i18n.T("orbit_escape_already"))
		return
	}

	body, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]
	if !ok {
		fmt.Println(i18n.T("error_body_not_found"))
		return
	}

	currentRadius := body.Radius + v.OrbitAltitude
	if len(body.OrbitalLayers) > 0 {
		topLayer := body.OrbitalLayers[len(body.OrbitalLayers)-1]
		topRadius := body.Radius + topLayer.TypicalAltitude
		if currentRadius < topRadius-100 {
			fmt.Printf(i18n.T("orbit_escape_not_highest"), topLayer.Name)
			fmt.Printf(i18n.T("orbit_escape_current_alt"), v.OrbitAltitude, topLayer.TypicalAltitude)
			fmt.Println(i18n.T("orbit_escape_transfer_hint"))
			return
		}
	}

	dv_kmps := engine.EscapeFromOrbitDeltaV(body.GM, currentRadius)
	dv_mps := dv_kmps * 1000

	fmt.Printf(i18n.T("orbit_escape_info"), v.OrbitAltitude, body.Name)
	fmt.Printf(i18n.T("orbit_escape_dv"), dv_mps)

	if v.DeltaVRemaining < dv_mps {
		fmt.Printf(i18n.T("orbit_escape_dv_insufficient"), dv_mps, v.DeltaVRemaining)
		return
	}
	if v.Power < 100 {
		fmt.Printf(i18n.T("orbit_escape_power_low"), 100)
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
	fmt.Printf(i18n.T("orbit_escape_success"), v.Name, body.Name)
	fmt.Printf(i18n.T("orbit_escape_remaining_dv"), v.DeltaVRemaining)
	fmt.Println(i18n.T("orbit_escape_travel_hint"))
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
		fmt.Println(i18n.T("orbit_dock_error_not_found"))
		return
	}
	if !master.IsActive || !slave.IsActive {
		fmt.Println(i18n.T("orbit_dock_error_inactive"))
		return
	}
	if master.ID == slave.ID {
		fmt.Println(i18n.T("orbit_dock_error_self"))
		return
	}
	if master.OrbitBodyID == "deep_space" || slave.OrbitBodyID == "deep_space" {
		fmt.Println(i18n.T("orbit_dock_error_deep_space"))
		return
	}
	if master.OrbitBodyID != slave.OrbitBodyID {
		fmt.Printf(i18n.T("orbit_dock_error_diff_orbit"), master.OrbitBodyID, slave.OrbitBodyID)
		return
	}
	if master.OrbitAltitude < 0 || slave.OrbitAltitude < 0 {
		fmt.Println(i18n.T("orbit_dock_error_altitude_invalid"))
		return
	}
	if master.OrbitAltitude > slave.OrbitAltitude+100 || master.OrbitAltitude < slave.OrbitAltitude-100 {
		fmt.Printf(i18n.T("orbit_dock_error_altitude_diff"), master.OrbitAltitude, slave.OrbitAltitude)
		return
	}
	if master.IsAssembly && contains(master.DockedWith, slave.ID) {
		fmt.Printf(i18n.T("orbit_dock_error_already_docked"), master.Name, slave.Name)
		return
	}
	if slave.IsAssembly && contains(slave.DockedWith, master.ID) {
		fmt.Printf(i18n.T("orbit_dock_error_already_docked"), slave.Name, master.Name)
		return
	}

	masterHasControl := master.HasControlCenter
	slaveHasControl := slave.HasControlCenter
	if slaveHasControl && !masterHasControl {
		master.HasControlCenter = true
	}
	if !slaveHasControl {
		master.Firmware = append(master.Firmware, slave.ID)
		fmt.Printf(i18n.T("orbit_dock_firmware"), slave.Name)
	} else {
		master.Firmware = append(master.Firmware, slave.Firmware...)
		fmt.Printf(i18n.T("orbit_dock_control"), len(master.Firmware))
	}

	master.Modules = append(master.Modules, slave.Modules...)
	master.MaxPower += slave.MaxPower
	master.Power += slave.Power
	master.DataStored += slave.DataStored
	master.DataRate += slave.DataRate
	master.DeltaVRemaining += slave.DeltaVRemaining
	master.IsAssembly = true
	master.DockedWith = append(master.DockedWith, slave.ID)
	master.CargoBays = append(master.CargoBays, slave.CargoBays...)
	master.Firmware = append(master.Firmware, slave.Firmware...)

	slave.IsActive = false
	slave.OrbitBodyID = "docked"

	fmt.Printf(i18n.T("orbit_dock_success"), master.Name, slave.Name, master.Name)
	fmt.Printf(i18n.T("orbit_dock_power"), master.Power, master.MaxPower)
	fmt.Printf(i18n.T("orbit_dock_data_rate"), master.DataRate)
	fmt.Printf(i18n.T("orbit_dock_dv"), master.DeltaVRemaining)
	fmt.Printf(i18n.T("orbit_dock_list"), strings.Join(master.DockedWith, ", "))
	if len(slave.CargoBays) > 0 {
		fmt.Printf(i18n.T("orbit_dock_cargo"), len(slave.CargoBays))
	}
	fmt.Printf(i18n.T("orbit_dock_control_status"), map[bool]string{true: i18n.T("sat_yes"), false: i18n.T("sat_no")}[master.HasControlCenter])
	if len(master.Firmware) > 0 {
		fmt.Printf(i18n.T("orbit_dock_firmware_count"), len(master.Firmware))
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}

	state, err := r.getFlybyState(v)
	if err != nil {
		fmt.Printf(i18n.T("error_flyby_state"), err)
		return
	}
	if v.OrbitBodyID != "deep_space" {
		fmt.Printf(i18n.T("orbit_travel_not_deep_space"), v.OrbitBodyID)
		return
	}
	if v.DepartureBody == "" {
		fmt.Println(i18n.T("orbit_travel_no_departure"))
		return
	}

	target, ok := r.game.StarSystem.CelestialBodies[targetID]
	if !ok {
		fmt.Printf(i18n.T("error_body_not_found"), targetID)
		return
	}
	fromBody, ok := r.game.StarSystem.CelestialBodies[v.DepartureBody]
	if !ok {
		fmt.Println(i18n.T("orbit_travel_no_departure"))
		return
	}

	newState, dv, flightDays, err := engine.ApplyHohmannTransfer(state, target, r.game.CurrentTime)
	if err != nil {
		fmt.Printf(i18n.T("orbit_travel_calc_failed"), err)
		return
	}

	windowStart, windowEnd, waitDays, _, err := engine.NextWindow(fromBody, target, r.game.CurrentTime, r.cfg.Physics.WindowSearchDays)
	if err != nil {
		fmt.Printf(i18n.T("error_window_calc_failed"), err)
		return
	}
	current := r.game.CurrentTime
	if current.Before(windowStart) || current.After(windowEnd) {
		fmt.Printf(i18n.T("orbit_travel_window_not_in"))
		fmt.Printf(i18n.T("orbit_travel_window_next"), windowStart.Format("2006-01-02"))
		fmt.Printf(i18n.T("orbit_travel_window_wait"), waitDays)
		fmt.Printf(i18n.T("orbit_travel_window_tick"), waitDays)
		return
	}

	r.updateVesselState(v, newState)
	v.DepartureBody = targetID
	arrivalTime := r.game.CurrentTime.AddDate(0, 0, int(flightDays)+1)
	v.ArrivalTime = arrivalTime

	fmt.Printf(i18n.T("orbit_travel_success"))
	fmt.Printf(i18n.T("orbit_travel_arrived"), v.Name, target.Name, v.OrbitAltitude)
	fmt.Printf(i18n.T("orbit_travel_dv"), dv, v.DeltaVRemaining)
	fmt.Printf(i18n.T("orbit_travel_arrival_time"), arrivalTime.Format("2006-01-02"), flightDays)
}

// ==================== orbit release 命令 ====================
func (r *Repl) releaseCommand(vesselID string, bayIndexStr string, itemIndexStr string) {
	bayIndex, err := strconv.Atoi(bayIndexStr)
	if err != nil || bayIndex < 1 {
		fmt.Println(i18n.T("orbit_release_invalid_bay"))
		return
	}
	itemIndex, err := strconv.Atoi(itemIndexStr)
	if err != nil || itemIndex < 1 {
		fmt.Println(i18n.T("orbit_release_invalid_item"))
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}

	bay, _ := r.getCargoBayByIndex(v, bayIndex)
	if bay == nil {
		fmt.Printf(i18n.T("orbit_release_bay_not_found"), bayIndex)
		fmt.Println(i18n.T("orbit_release_available_bays"))
		for _, b := range v.CargoBays {
			fmt.Printf(i18n.T("orbit_release_bay_item"), b.Index, len(b.Loaded))
		}
		return
	}
	if len(bay.Loaded) == 0 {
		fmt.Printf(i18n.T("orbit_release_bay_empty"), bayIndex)
		return
	}

	item, itemIdx := r.getCargoItemByIndex(bay, itemIndex)
	if item == nil {
		fmt.Printf(i18n.T("orbit_release_item_not_found"), itemIndex)
		fmt.Printf(i18n.T("orbit_release_bay_contents"), bayIndex)
		for i, cargo := range bay.Loaded {
			fmt.Printf(i18n.T("orbit_release_bay_content_item"), i+1, r.getCargoName(cargo), cargo.Type)
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
		fmt.Printf(i18n.T("orbit_release_unknown_type"), item.Type)
	}

	if r.hasControlChip(&newVessel) {
		newVessel.HasControlCenter = true
		fmt.Println(i18n.T("orbit_release_control_activated"))
	}

	r.game.Vessels = append(r.game.Vessels, newVessel)

	fmt.Printf(i18n.T("orbit_release_success"))
	fmt.Printf(i18n.T("orbit_release_item"), cargoName)
	fmt.Printf(i18n.T("orbit_release_mass"), cargoMass)
	fmt.Printf(i18n.T("orbit_release_new_id"), newVesselID)
	fmt.Printf(i18n.T("orbit_release_position"),
		r.game.StarSystem.CelestialBodies[v.OrbitBodyID].Name, v.OrbitAltitude)
	fmt.Println(i18n.T("orbit_release_hint"))
	fmt.Println(i18n.T("orbit_release_tip"))

	if len(bay.Loaded) == 0 {
		fmt.Printf(i18n.T("orbit_release_bay_empty"), bayIndex)
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}

	state, err := r.getFlybyState(v)
	if err != nil {
		fmt.Printf(i18n.T("error_flyby_state"), err)
		return
	}
	if state.CurrentBodyID == "deep_space" {
		fmt.Println(i18n.T("orbit_flyby_deep_space"))
		return
	}

	bestPlanet, bestDV, bestState, err := engine.PlanNextFlyby(state, r.game.StarSystem, r.game.CurrentTime, 0)
	if err != nil {
		fmt.Printf(i18n.T("orbit_flyby_plan_failed"), err)
		return
	}
	planet := r.game.StarSystem.CelestialBodies[bestPlanet]
	fmt.Printf(i18n.T("orbit_flyby_plan"), planet.Name)
	fmt.Printf(i18n.T("orbit_flyby_plan_dv"), bestDV)
	fmt.Printf(i18n.T("orbit_flyby_plan_remaining_dv"), bestState.DeltaVRemaining)
	fmt.Printf(i18n.T("orbit_flyby_plan_position"), planet.Name)
	fmt.Println(i18n.T("orbit_flyby_plan_hint"))
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}

	planet, ok := r.game.StarSystem.CelestialBodies[planetID]
	if !ok {
		fmt.Printf(i18n.T("error_body_not_found"), planetID)
		return
	}
	if planetID == v.OrbitBodyID {
		fmt.Println(i18n.T("orbit_flyby_same_body"))
		return
	}

	periapsis := planet.Radius + 500.0
	if periapsisStr != "" {
		p, err := strconv.ParseFloat(periapsisStr, 64)
		if err == nil && p > 0 {
			periapsis = p
			if periapsis < planet.Radius {
				fmt.Printf(i18n.T("orbit_flyby_periapsis_warning"), planet.Radius+100)
				periapsis = planet.Radius + 100
			}
		} else {
			fmt.Println(i18n.T("orbit_flyby_periapsis_invalid"))
		}
	}

	state, err := r.getFlybyState(v)
	if err != nil {
		fmt.Printf(i18n.T("error_flyby_state"), err)
		return
	}

	newState, dv, err := engine.ComputeFlybyState(state, planet, periapsis, r.game.CurrentTime)
	if err != nil {
		fmt.Printf(i18n.T("orbit_flyby_execute_failed"), err)
		return
	}

	r.updateVesselState(v, newState)
	v.FlybyHistory = append(v.FlybyHistory, planetID)

	fmt.Printf(i18n.T("orbit_flyby_execute_success"))
	fmt.Printf(i18n.T("orbit_flyby_execute_planet"), planet.Name)
	fmt.Printf(i18n.T("orbit_flyby_execute_periapsis"), periapsis)
	fmt.Printf(i18n.T("orbit_flyby_execute_dv"), dv)
	fmt.Printf(i18n.T("orbit_flyby_execute_remaining_dv"), v.DeltaVRemaining)
	fmt.Printf(i18n.T("orbit_flyby_execute_new_vel"), newState.Vel[0], newState.Vel[1])
	fmt.Printf(i18n.T("orbit_flyby_execute_position"), planet.Name)
	fmt.Println(i18n.T("orbit_flyby_execute_hint"))
}

// ==================== orbit 命令分发 ====================

func (r *Repl) handleOrbitCommand(subCmd string, args []string) {
	switch subCmd {
	case "info":
		if len(args) < 1 {
			fmt.Println(i18n.T("orbit_info_usage"))
			return
		}
		r.orbitCommand(args[0])
	case "transfer":
		if len(args) < 2 {
			fmt.Println(i18n.T("orbit_transfer_usage"))
			return
		}
		r.transferCommand(args[0], args[1])
	case "escape":
		if len(args) < 1 {
			fmt.Println(i18n.T("orbit_escape_usage"))
			return
		}
		r.escapeCommand(args[0])
	case "dock":
		if len(args) < 2 {
			fmt.Println(i18n.T("orbit_dock_usage"))
			return
		}
		r.dockCommand(args[0], args[1])
	case "travel":
		if len(args) < 2 {
			fmt.Println(i18n.T("orbit_travel_usage"))
			return
		}
		r.travelCommand(args[0], args[1])
	case "release":
		if len(args) < 3 {
			fmt.Println(i18n.T("orbit_release_usage"))
			return
		}
		r.releaseCommand(args[0], args[1], args[2])
	case "flyby":
		if len(args) < 2 {
			fmt.Println(i18n.T("orbit_flyby_usage"))
			return
		}
		switch args[0] {
		case "plan":
			if len(args) < 2 {
				fmt.Println(i18n.T("orbit_flyby_plan_usage"))
				return
			}
			r.flybyPlan(args[1])
		case "execute":
			if len(args) < 3 {
				fmt.Println(i18n.T("orbit_flyby_execute_usage"))
				return
			}
			alt := ""
			if len(args) >= 4 {
				alt = args[3]
			}
			r.flybyExecute(args[1], args[2], alt)
		default:
			fmt.Printf(i18n.T("orbit_flyby_unknown"), args[0])
		}
	default:
		fmt.Println(i18n.T("orbit_unknown_command"))
		fmt.Println(i18n.T("orbit_help"))
		fmt.Println(i18n.T("orbit_help_info"))
		fmt.Println(i18n.T("orbit_help_transfer"))
		fmt.Println(i18n.T("orbit_help_escape"))
		fmt.Println(i18n.T("orbit_help_dock"))
		fmt.Println(i18n.T("orbit_help_travel"))
		fmt.Println(i18n.T("orbit_help_release"))
		fmt.Println(i18n.T("orbit_help_flyby_plan"))
		fmt.Println(i18n.T("orbit_help_flyby_execute"))
	}
}
