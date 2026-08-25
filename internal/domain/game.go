package domain

import "time"

// Game 全局游戏状态（聚合根）
type Game struct {
	Version       string              `json:"version"`          // 存档版本号
	CurrentTime   time.Time           `json:"current_time"`     // 当前游戏时间
	Player        Player              `json:"player"`           // 玩家数据
	StarSystem    StarSystem          `json:"star_system"`      // 星表数据
	PartsDB       map[string]Part     `json:"parts_db"`         // 零件库，ID -> Part
	SuppliersDB   map[string]Supplier `json:"suppliers_db"`     // 供应商库
	BasesDB       map[string]Base     `json:"bases_db"`         // 发射基地库
	Contracts     []Contract          `json:"contracts"`        // 所有合约（包括历史）
	Launches      []LaunchMission     `json:"launches"`         // 发射任务记录
	LogBook       []LogEntry          `json:"log_book"`         // 日志（含玩家批注）
	Config        GameConfig          `json:"config"`           // 游戏配置
	RocketDesigns []RocketDesign      `json:"rocket_designs"`   // 玩家设计的火箭列表
	SatelliteDesigns []SatelliteDesign `json:"satellite_designs"` // 玩家设计的卫星列表
	Vessels       []Vessel            `json:"vessels"`          // 所有在轨航天器（统一结构）
	AssemblyProjects []AssemblyProject `json:"assembly_projects"` // 在轨组装项目
	// SandboxData   *SandboxGameData    `json:"sandbox_data,omitempty"` // 沙盒模式数据（暂未启用）
}

// GameConfig 游戏配置参数
type GameConfig struct {
	EnableSandbox    bool    `json:"enable_sandbox"`     // 是否启用沙盒模式
	EnableECCL       bool    `json:"enable_ecc"`         // 是否启用 ECCL 编程
	TimeScale        float64 `json:"time_scale"`         // 现实1秒对应游戏时间（天）
	AutoSaveInterval int     `json:"auto_save_interval"` // 自动保存间隔（分钟）
}
