package repl

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
	"astrolex/internal/i18n"
)

// ==================== 卫星设计列表 ====================

// listSatellites 显示所有已设计的卫星（非在轨）
func (r *Repl) listSatellites() {
	if len(r.game.SatelliteDesigns) == 0 {
		fmt.Println(i18n.T("no_satellite_designs"))
		return
	}
	for _, s := range r.game.SatelliteDesigns {
		fmt.Printf(i18n.T("satellite_design_item"),
			s.ID, s.Name, s.TotalMass, s.TotalPower, s.TotalDataRate, s.TotalCost)
		var mods []string
		for _, m := range s.Modules {
			mods = append(mods, m.Name)
		}
		fmt.Printf(i18n.T("satellite_design_modules"), strings.Join(mods, ", "))
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
		fmt.Printf(i18n.T("error_sat_design_not_found"), satID)
		return
	}
	fmt.Printf(i18n.T("edit_satellite_title"), sat.Name, sat.ID)
	r.designSatelliteCommon(sat)
}

func (r *Repl) designSatelliteCommon(preset *domain.SatelliteDesign) {
	isEdit := preset != nil
	var name string

	if isEdit {
		fmt.Printf(i18n.T("edit_satellite_name_prompt"), preset.Name)
		fmt.Print(i18n.T("enter_satellite_name"))
		nameInput, _ := r.reader.ReadString('\n')
		nameInput = strings.TrimSpace(nameInput)
		if nameInput == "" {
			name = preset.Name
		} else {
			name = nameInput
		}
	} else {
		fmt.Print(i18n.T("enter_satellite_name"))
		name, _ = r.reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name == "" {
			name = i18n.T("default_satellite_name")
		}
	}

	moduleCatalog, err := loadSatelliteModules("data/satellite_modules.json")
	if err != nil {
		fmt.Printf(i18n.T("error_load_modules"), err)
		return
	}

	fmt.Println(i18n.T("available_satellite_modules"))
	for _, m := range moduleCatalog {
		fmt.Printf(i18n.T("satellite_module_format"),
			m.ID, m.Name, m.Type, m.Mass, m.PowerConsume, m.DataRate, m.Cost)
	}

	if isEdit {
		var currentMods []string
		for _, m := range preset.Modules {
			currentMods = append(currentMods, m.ID)
		}
		if len(currentMods) > 0 {
			fmt.Printf(i18n.T("edit_satellite_current_modules"), strings.Join(currentMods, " "))
		}
	}

	var selectedModules []domain.SatelliteModule
	fmt.Println(i18n.T("enter_module_ids"))
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
					fmt.Printf(i18n.T("warning_module_not_found"), token)
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
				fmt.Printf(i18n.T("warning_module_not_found"), token)
				continue
			}
			selectedModules = append(selectedModules, *found)
		}
	}

	if len(selectedModules) == 0 {
		fmt.Println(i18n.T("no_modules_selected"))
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
		fmt.Println(i18n.T("satellite_design_updated"))
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
		fmt.Println(i18n.T("satellite_design_complete"))
	}

	fmt.Printf(i18n.T("satellite_design_info"),
		name, satID, totalMass, totalPower, totalDataRate, totalCost, satID)
}

// ==================== 在轨航天器命令 ====================

// vesselList 列出所有在轨航天器（Vessel）
func (r *Repl) vesselList() {
	if len(r.game.Vessels) == 0 {
		fmt.Println(i18n.T("sat_list_empty"))
		return
	}
	fmt.Println(i18n.T("sat_list_title"))
	for _, v := range r.game.Vessels {
		if !v.IsActive {
			continue
		}
		bodyName := i18n.T("deep_space")
		if v.OrbitBodyID != "deep_space" && v.OrbitBodyID != "docked" {
			if b, ok := r.game.StarSystem.CelestialBodies[v.OrbitBodyID]; ok {
				bodyName = b.Name
			}
		} else if v.OrbitBodyID == "deep_space" {
			bodyName = i18n.T("deep_space")
		} else {
			bodyName = i18n.T("docked")
		}
		ccMark := ""
		if v.HasControlCenter {
			ccMark = " " + i18n.T("control_center_mark")
		}
		typeMark := ""
		switch v.Type {
		case domain.VesselSingle:
			typeMark = " " + i18n.T("vessel_type_single")
		case domain.VesselComposite:
			typeMark = " " + i18n.T("vessel_type_composite")
		case domain.VesselCargo:
			typeMark = " " + i18n.T("vessel_type_cargo")
		case domain.VesselAircraft:
			typeMark = " " + i18n.T("vessel_type_aircraft")
		}
		status := i18n.T("sat_status_normal")
		if !v.IsActive {
			status = i18n.T("sat_status_failed")
		}
		fmt.Printf(i18n.T("sat_list_item"),
			v.ID, v.Name, ccMark, typeMark, bodyName,
			v.Power, v.DataStored, v.DeltaVRemaining, status)
	}
}

// vesselStatus 显示航天器详细信息
func (r *Repl) vesselStatus(vesselID string) {
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
		fmt.Printf(i18n.T("sat_status_deep_space_title"), v.Name, v.ID)
		fmt.Printf(i18n.T("sat_status_type"), v.Type)
		fmt.Printf(i18n.T("sat_status_departure"), v.DepartureBody)
		fmt.Printf(i18n.T("sat_status_dv"), v.DeltaVRemaining)
		fmt.Printf(i18n.T("sat_status_power"), v.Power, v.MaxPower)
		fmt.Printf(i18n.T("sat_status_data"), v.DataStored)
		fmt.Printf(i18n.T("sat_status_datarate"), v.DataRate)
		fmt.Printf(i18n.T("sat_status_condition"), map[bool]string{true: i18n.T("sat_status_normal"), false: i18n.T("sat_status_failed")}[v.IsActive])
		if !v.ArrivalTime.IsZero() {
			fmt.Printf(i18n.T("sat_status_arrival"), v.ArrivalTime.Format("2006-01-02"))
		}
		if v.IsAssembly {
			fmt.Printf(i18n.T("sat_status_assembly"), len(v.DockedWith))
			fmt.Printf(i18n.T("sat_status_docked_list"), strings.Join(v.DockedWith, ", "))
		}
		if len(v.CargoBays) > 0 {
			fmt.Println(i18n.T("sat_status_cargo"))
			for _, bay := range v.CargoBays {
				fmt.Printf(i18n.T("sat_status_cargo_bay"), bay.Index, len(bay.Loaded))
				for i, item := range bay.Loaded {
					fmt.Printf(i18n.T("sat_status_cargo_item"), i+1, r.getCargoName(item), item.Type)
				}
			}
		}
		fmt.Printf(i18n.T("sat_status_control"), map[bool]string{true: i18n.T("sat_yes"), false: i18n.T("sat_no")}[v.HasControlCenter])
		if len(v.Firmware) > 0 {
			fmt.Printf(i18n.T("sat_status_firmware"), len(v.Firmware))
			for _, fw := range v.Firmware {
				for _, vv := range r.game.Vessels {
					if vv.ID == fw {
						fmt.Printf(i18n.T("sat_status_firmware_item"), vv.Name)
						break
					}
				}
			}
		}
		return
	}

	if v.OrbitBodyID == "docked" {
		fmt.Printf(i18n.T("sat_status_docked"), v.Name)
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

	fmt.Printf(i18n.T("sat_status_title"), v.Name, v.ID)
	fmt.Printf(i18n.T("sat_status_type"), v.Type)
	fmt.Printf(i18n.T("sat_status_body"), body.Name)
	fmt.Printf(i18n.T("sat_status_layer"), layerName, v.OrbitAltitude)
	fmt.Printf(i18n.T("sat_status_radius"), radius)
	fmt.Printf(i18n.T("sat_status_circular_velocity"), v_c)
	fmt.Printf(i18n.T("sat_status_escape_velocity"), v_esc)
	fmt.Printf(i18n.T("sat_status_dv"), v.DeltaVRemaining)
	fmt.Printf(i18n.T("sat_status_launch_time"), v.LaunchTime.Format("2006-01-02"))
	fmt.Printf(i18n.T("sat_status_power"), v.Power, v.MaxPower)
	fmt.Printf(i18n.T("sat_status_data"), v.DataStored)
	fmt.Printf(i18n.T("sat_status_datarate"), v.DataRate)
	fmt.Printf(i18n.T("sat_status_condition"), map[bool]string{true: i18n.T("sat_status_normal"), false: i18n.T("sat_status_failed")}[v.IsActive])
	if v.IsAssembly {
		fmt.Printf(i18n.T("sat_status_assembly"), len(v.DockedWith))
		fmt.Printf(i18n.T("sat_status_docked_list"), strings.Join(v.DockedWith, ", "))
	}
	if len(v.CargoBays) > 0 {
		fmt.Println(i18n.T("sat_status_cargo"))
		for _, bay := range v.CargoBays {
			fmt.Printf(i18n.T("sat_status_cargo_bay"), bay.Index, len(bay.Loaded))
			for i, item := range bay.Loaded {
				fmt.Printf(i18n.T("sat_status_cargo_item"), i+1, r.getCargoName(item), item.Type)
			}
		}
	}
	fmt.Printf(i18n.T("sat_status_control"), map[bool]string{true: i18n.T("sat_yes"), false: i18n.T("sat_no")}[v.HasControlCenter])
	if len(v.Firmware) > 0 {
		fmt.Printf(i18n.T("sat_status_firmware"), len(v.Firmware))
		for _, fw := range v.Firmware {
			for _, vv := range r.game.Vessels {
				if vv.ID == fw {
					fmt.Printf(i18n.T("sat_status_firmware_item"), vv.Name)
					break
				}
			}
		}
	}
	fmt.Println(i18n.T("sat_status_modules"))
	for _, m := range v.Modules {
		fmt.Printf(i18n.T("sat_status_module"), m.Name, m.Type, m.PowerConsume, m.DataRate)
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}
	if v.OrbitBodyID == "docked" {
		fmt.Println(i18n.T("sat_measure_docked"))
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
		fmt.Println(i18n.T("sat_measure_no_sensor"))
		return
	}
	if v.Power < r.cfg.Satellite.MeasurePowerCost {
		fmt.Printf(i18n.T("sat_measure_power_low"), r.cfg.Satellite.MeasurePowerCost)
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
	fmt.Printf(i18n.T("sat_measure_ok"), dataAmount, v.Power, v.DataStored)
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}
	if v.OrbitBodyID == "docked" {
		fmt.Println(i18n.T("sat_send_docked"))
		return
	}
	if v.DataStored == 0 {
		fmt.Println(i18n.T("sat_send_no_data"))
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
		fmt.Println(i18n.T("sat_send_no_comms"))
		return
	}
	if v.Power < r.cfg.Satellite.SendPowerCost {
		fmt.Printf(i18n.T("sat_send_power_low"), r.cfg.Satellite.SendPowerCost)
		return
	}
	v.Power -= r.cfg.Satellite.SendPowerCost
	reward := int64(v.DataStored * float64(r.cfg.Satellite.DataRewardPerMB))
	r.game.Player.Credits += reward
	fmt.Printf(i18n.T("sat_send_ok"), reward, v.Power)
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
		fmt.Printf(i18n.T("error_vessel_not_found"), vesselID)
		return
	}
	if !v.IsActive {
		fmt.Println(i18n.T("error_vessel_inactive"))
		return
	}
	if v.OrbitBodyID == "docked" {
		fmt.Println(i18n.T("sat_point_docked"))
		return
	}
	fmt.Printf(i18n.T("sat_point_ok"), v.Name, target)
}
