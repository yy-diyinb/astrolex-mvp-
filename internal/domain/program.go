package domain

import "time"

// ECCLProgram 表示一个完整的 ECCL 程序
type ECCLProgram struct {
	ID          string              `json:"id"`           // 程序唯一标识
	Name        string              `json:"name"`         // 程序名称
	Code        string              `json:"code"`         // 源代码（纯文本，包含注释和标签）
	CodeLines   []string            `json:"-"`            // 解析后的代码行（内存缓存，不序列化）
	Labels      map[string]int      `json:"labels"`       // 标签名 -> 指令索引（解析后填充）
	Registers   map[string]float64  `json:"registers"`    // 运行时寄存器（初始值）
	EventHandlers map[string]string `json:"event_handlers"` // 事件名 -> 对应的标签（如 "RAD_ALARM" -> ":handle_rad"）
	UploadedAt  time.Time           `json:"uploaded_at"`  // 上传时间
	IsRunning   bool                `json:"is_running"`   // 是否正在运行
	LastExecuted time.Time          `json:"last_executed"` // 上次执行时间
	Logs        []string            `json:"logs"`         // 执行日志（最多保留 100 条）
}

// ECCLEventType 事件类型
type ECCLEventType string

const (
	EventModuleArrived ECCLEventType = "MODULE_ARRIVED" // 模块到达指定轨道
	EventDockComplete  ECCLEventType = "DOCK_COMPLETE"  // 对接完成
	EventLowPower      ECCLEventType = "LOW_POWER"      // 电力低
	EventWindowOpen    ECCLEventType = "WINDOW_OPEN"    // 发射窗口打开
	EventRadAlarm      ECCLEventType = "RAD_ALARM"      // 辐射警报
)

// ECCLEvent 事件记录
type ECCLEvent struct {
	Type      ECCLEventType `json:"type"`
	Timestamp time.Time     `json:"timestamp"`
	SourceID  string        `json:"source_id"`  // 触发的航天器或项目ID
	Value     float64       `json:"value"`      // 相关数值
	Message   string        `json:"message"`
}

// ECCLProgramTemplate 预定义的程序模板（用于快速创建）
type ECCLProgramTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
}

// ECCLExecutionResult 表示一次程序执行的结果
type ECCLExecutionResult struct {
	ProgramID   string    `json:"program_id"`
	SatelliteID string    `json:"satellite_id"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	ExecutedAt  time.Time `json:"executed_at"`
	Duration    float64   `json:"duration"`    // 执行耗时（秒）
	Steps       int       `json:"steps"`       // 执行的指令步数
	Logs        []string  `json:"logs"`        // 执行过程中的日志
}
