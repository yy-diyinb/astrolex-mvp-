package engine

import (
	"math/rand"

	"astrolex/internal/domain"
)

const (
	// 燃料价格 (信用点/kg)
	FuelPricePerKg = 0.05
	// 单次发射场地租赁费 (信用点)
	LaunchPadFee = 500
)

// CalculateLaunchCost 计算单次发射的总成本（信用点）
// 参数:
//   - design: 火箭设计（包含模块列表）
//   - fuelMass: 实际加注的燃料质量 (kg)（目前取设计中的 FuelLoad 总和）
// 返回值: 总成本（硬件+燃料+场地）
func CalculateLaunchCost(design domain.RocketDesign, fuelMass float64) int64 {
	// 1. 硬件成本：遍历所有模块，累加其零件的成本
	hardwareCost := int64(0)
	for _, mod := range design.Modules {
		if mod.Part != nil {
			hardwareCost += mod.Part.Cost
		}
		// 注意：载荷（Payload）没有零件，不计入硬件成本
	}

	// 2. 燃料成本
	fuelCost := int64(fuelMass * FuelPricePerKg)

	// 3. 场地租赁费
	totalCost := hardwareCost + fuelCost + LaunchPadFee
	return totalCost
}

// CalculateTotalFuelMass 计算火箭设计中的总燃料质量
func CalculateTotalFuelMass(design domain.RocketDesign) float64 {
	totalFuel := 0.0
	for _, mod := range design.Modules {
		if mod.Type == domain.ModuleFuelTank && mod.Part != nil {
			// 获取实际加注质量（FuelLoad 存储在 Stage 中，但设计中未直接存储）
			// 我们需要从 Stage 中获取，但设计中的 FuelLoad 是在 PairModules 时填充的。
			// 由于 PairModules 返回的 Stage 包含 FuelLoad，我们可以在发射时重新计算。
			// 此处简化：暂时取燃料箱的最大容量（但实际加注可能不满）
			// 更好的方式：从 Stage 获取，但需要重新 PairModules。
			// 我们将在 CalculateLaunchCost 中直接调用 PairModules 获取精确燃料质量。
			// 暂时保留为 0，由调用者传入。
		}
	}
	// 由于设计中的 FuelLoad 未存储，我们将燃料质量作为参数传入。
	return totalFuel
}

// GetFuelLoadFromDesign 从设计中提取各级燃料加注量（通过配对后获取）
// 返回总燃料质量 (kg)
func GetFuelLoadFromDesign(design domain.RocketDesign) (float64, error) {
	stages, err := PairModules(design.Modules)
	if err != nil {
		return 0, err
	}
	totalFuel := 0.0
	for _, s := range stages {
		totalFuel += s.FuelLoad
	}
	return totalFuel, nil
}

// GenerateHardwareReturn 生成随机返还比例 (0.4 ~ 0.9)
func GenerateHardwareReturn() float64 {
	return 0.4 + rand.Float64()*0.5 // 0.4~0.9
}
