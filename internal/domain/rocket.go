package domain

// ModuleType 模块类型
type ModuleType string

const (
	ModulePayload   ModuleType = "载荷"
	ModuleAvionics  ModuleType = "航电"
	ModuleEngine    ModuleType = "引擎"
	ModuleFuelTank  ModuleType = "燃料箱"
	ModuleFairing   ModuleType = "整流罩"
	ModuleSatellite ModuleType = "卫星"
	ModuleCargoBay  ModuleType = "货舱"
	// 扩展预留
	ModuleCargo    ModuleType = "货舱"   // 合并到 ModuleCargoBay，保留兼容
	ModuleCrew     ModuleType = "乘员舱"
	ModuleDocking  ModuleType = "对接机构"
)

// Module 表示火箭设计中的一个模块
type Module struct {
	Type     ModuleType `json:"type"`
	PartID   string     `json:"part_id,omitempty"`
	Part     *Part      `json:"-"`
	Position int        `json:"position"`
}

// RocketDesign 火箭设计（含货舱装载信息）
type RocketDesign struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	ContractID  string   `json:"contract_id"`
	Modules     []Module `json:"modules"`
	PayloadMass float64  `json:"payload_mass"`
	TotalMass   float64  `json:"total_mass"`
	DeltaV      float64  `json:"delta_v"`
	MaxAccel    float64  `json:"max_accel"`
	Stages      int      `json:"stages"`

	// 货舱装载信息（按模块顺序，仅包含有装载的货舱）
	CargoBays []CargoBay `json:"cargo_bays"`
}

// Stage 用于物理计算的内部结构（不持久化）
type Stage struct {
	StageNumber     int
	Engine          Part
	FuelTank        Part
	FuelLoad        float64
	Avionics        *Part
	StructuralMass  float64
	Modules         []Module // 该级包含的模块
}
