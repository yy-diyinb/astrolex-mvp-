package domain

import "time"

type Contract struct {
    ID          string    `json:"id"`
    Issuer      string    `json:"issuer"`
    TargetBodyID string   `json:"target_body_id"`
    PayloadMass float64   `json:"payload_mass"`
    PayloadVolume float64 `json:"payload_volume"`
    MaxAccelLimit   float64  `json:"max_accel_limit"`
    ForbiddenSuppliers []string `json:"forbidden_suppliers"`
    RequiredInsurance string   `json:"required_insurance"`
    DeliveryDeadline time.Time `json:"delivery_deadline"`
    RewardCredits    int64     `json:"reward_credits"`
    PenaltyCredits   int64     `json:"penalty_credits"`
    Status           string    `json:"status"`

    // 预算与成本
    Budget          int64   `json:"budget"`
    HardwareReturn  float64 `json:"hardware_return"`
    Launches        []LaunchMission `json:"launches"`
    TotalCost       int64   `json:"total_cost"`
    Completed       bool    `json:"completed"`
    TargetPayloadDelivered float64 `json:"target_payload_delivered"`

    // ---- 新增：预算追踪 ----
    BudgetUsed         int64 `json:"budget_used"`          // 已使用的预算金额
    BudgetHardwareCost int64 `json:"budget_hardware_cost"` // 预算内已支付的硬件成本
    PlayerPaid         int64 `json:"player_paid"`          // 玩家超支支付的金额
}
