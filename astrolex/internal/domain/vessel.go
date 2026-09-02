package domain

import "time"

// VesselType 表示航天器类型
type VesselType string

const (
	VesselSingle    VesselType = "SINGLE"    // 单体卫星
	VesselComposite VesselType = "COMPOSITE" // 组合体
	VesselCargo     VesselType = "CARGO"     // 货物（释放后独立）
	VesselAircraft  VesselType = "AIRCRAFT"  // 飞机（未来扩展）
)

// Vessel 统一航天器结构（替代 ActiveSatellite）
type Vessel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        VesselType `json:"type"`
	DesignID    string    `json:"design_id"`    // 关联的设计ID（火箭/卫星/飞机）
	OrbitBodyID string    `json:"orbit_body_id"` // 当前绕转天体，或 "deep_space"
	OrbitAltitude float64 `json:"orbit_altitude"` // km
	LaunchTime  time.Time `json:"launch_time"`

	// 状态
	Power           float64 `json:"power"`
	MaxPower        float64 `json:"max_power"`
	DataStored      float64 `json:"data_stored"`
	DataRate        float64 `json:"data_rate"`
	DeltaVRemaining float64 `json:"delta_v_remaining"`
	IsActive        bool    `json:"is_active"`

	// 组合体相关
	IsAssembly bool     `json:"is_assembly"`
	DockedWith []string `json:"docked_with"`
	Modules    []SatelliteModule `json:"modules"`

	// 深空状态
	DepartureBody string    `json:"departure_body,omitempty"`
	ArrivalTime   time.Time `json:"arrival_time,omitempty"`
	FlybyHistory  []string  `json:"flyby_history"`

	// 日心状态
	HeliocentricPos [2]float64 `json:"heliocentric_pos"`
	HeliocentricVel [2]float64 `json:"heliocentric_vel"`

	// 货舱
	CargoBays []CargoBay `json:"cargo_bays"`

	// 控制中心与固件
	HasControlCenter bool     `json:"has_control_center"`
	Firmware         []string `json:"firmware"`

	// ECCL 程序
	ECCLPrograms    []ECCLProgram `json:"ecc_programs"`
	ActiveProgramID string        `json:"active_program_id"`

	// ---- 飞机专用（未来扩展） ----
	AircraftDesign *AircraftDesign `json:"aircraft_design,omitempty"`
}
