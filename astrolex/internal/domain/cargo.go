package domain

type CargoItem struct {
	Type string `json:"type"` // "rocket", "satellite", "part"
	ID   string `json:"id"`   // 设计ID或零件ID
}

type CargoBay struct {
	Index   int         `json:"index"`    // 在火箭中的位置（从1开始），用于标识
	Loaded  []CargoItem `json:"loaded"`   // 货物列表
}
