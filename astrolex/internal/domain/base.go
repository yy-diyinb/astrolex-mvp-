package domain

import "time"

type Base struct {
    ID          string  `json:"id"`
    Name        string  `json:"name"`
    Owner       string  `json:"owner"`         // 势力ID
    Latitude    float64 `json:"latitude"`      // 度
    LaunchSlots int     `json:"launch_slots"`  // 工位数
    Weather     Weather `json:"weather"`
    FuelTypes   []string `json:"fuel_types"`   // 可用燃料列表
    LogisticsModifier float64 `json:"logistics_modifier"` // 交期修正 (1.0标准)
}

type Weather struct {
    WindSpeed     float64   `json:"wind_speed"`      // m/s
    LightningRisk float64   `json:"lightning_risk"`  // 0-1
    UpdatedAt     time.Time `json:"updated_at"`
}
