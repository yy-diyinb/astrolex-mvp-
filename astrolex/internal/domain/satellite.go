package domain

import "time"

// SatelliteModule 卫星模块定义
type SatelliteModule struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`          // Power, Comms, Sensor, Propulsion, Structure
	Mass         float64 `json:"mass"`          // kg
	PowerConsume float64 `json:"power_consume"` // W (正值消耗，负值表示发电)
	DataRate     float64 `json:"data_rate"`     // kbps
	Cost         int64   `json:"cost"`          // 信用点
}

// SatelliteDesign 玩家设计的卫星
type SatelliteDesign struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Modules       []SatelliteModule  `json:"modules"`
	TotalMass     float64            `json:"total_mass"`      // kg
	TotalPower    float64            `json:"total_power"`     // W (净功耗，正为消耗)
	TotalDataRate float64            `json:"total_data_rate"` // kbps
	TotalCost     int64              `json:"total_cost"`      // 信用点
	CreatedAt     time.Time          `json:"created_at"`
}

// ActiveSatellite 在轨卫星实例（支持组合体、货舱、控制中心、ECCL 编程、引力辅助）
type ActiveSatellite struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DesignID    string    `json:"design_id"`      // 关联的卫星设计ID（组合体时留空）
	OrbitBodyID string    `json:"orbit_body_id"`  // 当前绕转的天体ID，或 "deep_space"
	OrbitAltitude float64 `json:"orbit_altitude"` // 当前轨道高度 (km)，深空时为0
	LaunchTime  time.Time `json:"launch_time"`

	// 状态
	Power           float64 `json:"power"`            // 当前电力 (Wh)
	MaxPower        float64 `json:"max_power"`        // 最大储能 (Wh)
	DataStored      float64 `json:"data_stored"`      // 已存储数据 (MB)
	DataRate        float64 `json:"data_rate"`        // 数据采集速率 (kbps)
	DeltaVRemaining float64 `json:"delta_v_remaining"` // 剩余 Δv (m/s)，用于深空机动
	IsActive        bool    `json:"is_active"`        // 是否正常工作

	// 组合体相关
	IsAssembly   bool     `json:"is_assembly"`    // 是否为组合体
	DockedWith   []string `json:"docked_with"`    // 对接的子卫星ID列表（仅组合体主星有效）
	Modules      []SatelliteModule `json:"modules"` // 模块列表（组合体为所有子模块合并）

	// 深空飞行历史
	DepartureBody string    `json:"departure_body,omitempty"` // 弹射离开的天体
	ArrivalTime   time.Time `json:"arrival_time,omitempty"`   // 预计到达时间（旅行中）

	// ---- 货舱系统 ----
	CargoBays []CargoBay `json:"cargo_bays"` // 该卫星/组合体携带的货舱及其装载的货物

	// ---- 控制中心与固件 ----
	HasControlCenter bool     `json:"has_control_center"` // 是否拥有控制中心（包含航电）
	Firmware         []string `json:"firmware"`           // 已连接的固件模块ID列表

	// ---- ECCL 编程系统 ----
	ECCLPrograms    []ECCLProgram `json:"ecc_programs"`     // 该卫星上存储的程序列表
	ActiveProgramID string        `json:"active_program_id"` // 当前正在执行的程序ID

	// ---- 引力辅助与状态追踪 ----
	HeliocentricPos [2]float64 `json:"heliocentric_pos"` // 日心位置 (km)
	HeliocentricVel [2]float64 `json:"heliocentric_vel"` // 日心速度 (km/s)
	FlybyHistory   []string   `json:"flyby_history"`     // 已飞掠的天体ID列表（按顺序）
}
