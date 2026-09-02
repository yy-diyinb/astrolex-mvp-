package domain

import "time"

// AssemblyStatus 组装项目状态
type AssemblyStatus string

const (
	AssemblyPlanning AssemblyStatus = "planning"
	AssemblyInProgress AssemblyStatus = "in_progress"
	AssemblyPaused    AssemblyStatus = "paused"
	AssemblyCompleted AssemblyStatus = "completed"
	AssemblyFailed    AssemblyStatus = "failed"
)

// AssemblyStep 组装步骤
type AssemblyStep struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	ModuleID    string    `json:"module_id"`     // 模块设计ID（如卫星、货舱等）
	Quantity    int       `json:"quantity"`      // 数量
	Launched    bool      `json:"launched"`      // 是否已发射
	Docked      bool      `json:"docked"`        // 是否已对接
	VesselID    string    `json:"vessel_id"`     // 在轨航天器ID（对接后）
	LaunchTime  time.Time `json:"launch_time"`   // 发射时间
	DockTime    time.Time `json:"dock_time"`     // 对接时间
}

// AssemblyProject 组装项目
type AssemblyProject struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Status      AssemblyStatus  `json:"status"`
	Steps       []AssemblyStep  `json:"steps"`
	CurrentStep int             `json:"current_step"` // 当前执行的步骤索引
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt time.Time       `json:"completed_at"`
	// 关联程序（ECCL程序控制）
	ProgramID   string          `json:"program_id"`   // 可选，用于自动化控制
}
