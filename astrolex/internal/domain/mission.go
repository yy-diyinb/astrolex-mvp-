package domain

import "time"

// LaunchMission 记录一次发射任务
type LaunchMission struct {
	ID           string       `json:"id"`
	ContractID   string       `json:"contract_id"`   // 关联合约ID（若有）
	RocketDesign RocketDesign `json:"rocket_design"` // 使用的火箭设计
	BaseID       string       `json:"base_id"`       // 发射基地ID
	LaunchTime   time.Time    `json:"launch_time"`   // 发射时间

	Success      bool     `json:"success"`                // 是否成功
	FailureReason string  `json:"failure_reason,omitempty"` // 失败原因
	FinalDeltaV  float64  `json:"final_delta_v"`          // 实际达到的 Δv (m/s)
	Redundancies []string `json:"redundancies"`           // 采用的冗余策略（记录）
}

// LogEntry 手册日志篇（系统或玩家批注）
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Category  string    `json:"category"`   // "System", "Mission", "PlayerNote"
	Content   string    `json:"content"`
}
