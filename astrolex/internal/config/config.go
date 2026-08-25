package config

import (
    "encoding/json"
    "os"
)

type GameConfig struct {
    InitialCredits int64 `json:"initial_credits"`
    InitialTime    string `json:"initial_time"`
    Reputation     ReputationConfig `json:"reputation"`
    Physics        PhysicsConfig `json:"physics"`
    Satellite      SatelliteConfig `json:"satellite"`
    Contract       ContractConfig `json:"contract"`
}

type ReputationConfig struct {
    Safety  int            `json:"safety"`
    Speed   int            `json:"speed"`
    Politic map[string]int `json:"politic"`
}

type PhysicsConfig struct {
    AtmosphereLossMps  float64 `json:"atmosphere_loss_mps"`
    FuelPricePerKg     float64 `json:"fuel_price_credit_per_kg"`
    LaunchPadFee       int64   `json:"launch_pad_fee"`
    WindowSearchDays   int     `json:"window_search_days"`
    LeoDeltaVRequired  float64 `json:"leo_delta_v_required"`
}

type SatelliteConfig struct {
    InitialPowerWh    float64 `json:"initial_power_wh"`
    MaxPowerWh        float64 `json:"max_power_wh"`
    MeasurePowerCost  float64 `json:"measure_power_cost"`
    SendPowerCost     float64 `json:"send_power_cost"`
    DataRewardPerMB   int64   `json:"data_reward_per_mb"`
}

type ContractConfig struct {
    BudgetBase         float64 `json:"budget_base"`
    BudgetMassFactor   float64 `json:"budget_mass_factor"`
    BudgetDistanceFactor float64 `json:"budget_distance_factor"`
    RewardBase         float64 `json:"reward_base"`
    RewardMassFactor   float64 `json:"reward_mass_factor"`
    RewardDistanceFactor float64 `json:"reward_distance_factor"`
    ReturnMin          float64 `json:"return_min"`
    ReturnMax          float64 `json:"return_max"`
}

func LoadConfig(path string) (*GameConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg GameConfig
    err = json.Unmarshal(data, &cfg)
    return &cfg, err
}
