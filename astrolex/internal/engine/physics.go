package engine

import (
	"errors"
	"math"
	"math/rand"
	"strconv"

	"astrolex/internal/domain"
)

// PairModules 将模块序列解析为各级（引擎-燃料箱配对）
// 从顶部到底部扫描，引擎向上找最近的未配对燃料箱
func PairModules(modules []domain.Module) ([]domain.Stage, error) {
	var stages []domain.Stage
	var fuelStack []int // 存储燃料箱索引（从顶部到当前位置）

	// 从顶部到底部遍历
	for i := 0; i < len(modules); i++ {
		mod := modules[i]
		switch mod.Type {
		case domain.ModuleFuelTank:
			fuelStack = append(fuelStack, i)
		case domain.ModuleEngine:
			if len(fuelStack) == 0 {
				return nil, errors.New("引擎找不到配对的燃料箱")
			}
			// 弹出栈顶（最近的燃料箱）
			fuelIdx := fuelStack[len(fuelStack)-1]
			fuelStack = fuelStack[:len(fuelStack)-1]
			enginePart := mod.Part
			fuelPart := modules[fuelIdx].Part
			if enginePart == nil || fuelPart == nil {
				return nil, errors.New("零件引用缺失")
			}
			stage := domain.Stage{
				StageNumber:    len(stages) + 1,
				Engine:         *enginePart,
				FuelTank:       *fuelPart,
				FuelLoad:       fuelPart.MassFuelMax,
				Avionics:       nil,
				StructuralMass: 0,
				Modules:        []domain.Module{modules[i], modules[fuelIdx]},
			}
			// 检查该级是否包含航电（在引擎和燃料箱之间）
			start := fuelIdx
			end := i
			if start > end {
				start, end = end, start
			}
			for j := start + 1; j < end; j++ {
				if modules[j].Type == domain.ModuleAvionics {
					if modules[j].Part != nil {
						stage.Avionics = modules[j].Part
					}
					break
				}
			}
			stage.StructuralMass = stage.FuelTank.MassDry * 0.05
			// 添加到stages（顺序是从顶部到底部，但物理上应从底部开始，所以稍后反转）
			stages = append(stages, stage)
		default:
			// 其他模块（航电、整流罩等）暂不参与配对
		}
	}
	if len(fuelStack) > 0 {
		return nil, errors.New("存在未配对的燃料箱")
	}
	if len(stages) == 0 {
		return nil, errors.New("至少需要一个引擎")
	}
	// 反转stages顺序，使第一级在底部
	for i, j := 0, len(stages)-1; i < j; i, j = i+1, j-1 {
		stages[i], stages[j] = stages[j], stages[i]
	}
	// 重新编号级数
	for i := range stages {
		stages[i].StageNumber = i + 1
	}
	return stages, nil
}

// CalcDeltaVFromModules 通过模块序列计算 Δv
func CalcDeltaVFromModules(modules []domain.Module, payloadMass float64) (float64, error) {
	stages, err := PairModules(modules)
	if err != nil {
		return 0, err
	}
	return CalcDeltaV(stages, payloadMass), nil
}

// CalcMaxAccelFromModules 通过模块序列计算最大加速度
func CalcMaxAccelFromModules(modules []domain.Module, payloadMass float64) (float64, error) {
	stages, err := PairModules(modules)
	if err != nil {
		return 0, err
	}
	return CalcMaxAccel(stages, payloadMass), nil
}

// CalcDeltaV 计算多级火箭真空总 Δv (m/s)
func CalcDeltaV(stages []domain.Stage, payloadMass float64) float64 {
	if len(stages) == 0 {
		return 0
	}
	type stageMass struct {
		dry  float64
		fuel float64
		isp  float64
	}
	masses := make([]stageMass, len(stages))
	for i, s := range stages {
		dry := s.Engine.MassDry + s.StructuralMass
		if s.Avionics != nil {
			dry += s.Avionics.MassDry
		}
		fuel := s.FuelLoad
		if fuel > s.FuelTank.MassFuelMax {
			fuel = s.FuelTank.MassFuelMax
		}
		isp := s.Engine.ISP
		if isp < 0 {
			isp = 0
		}
		masses[i] = stageMass{dry: dry, fuel: fuel, isp: isp}
	}
	totalMass := payloadMass
	for _, m := range masses {
		totalMass += m.dry + m.fuel
	}
	if totalMass <= 0 {
		return 0
	}
	totalDV := 0.0
	for i := 0; i < len(stages); i++ {
		m := masses[i]
		if m.isp <= 0 || m.fuel <= 0 {
			totalMass -= (m.dry + m.fuel)
			continue
		}
		initialMass := totalMass
		finalMass := totalMass - m.fuel
		if finalMass <= 0 {
			break
		}
		stageDV := m.isp * 9.80665 * math.Log(initialMass/finalMass)
		totalDV += stageDV
		totalMass -= (m.dry + m.fuel)
		if totalMass <= 0 {
			break
		}
	}
	return totalDV
}

// CalcMaxAccel 计算最大加速度 (G)
func CalcMaxAccel(stages []domain.Stage, payloadMass float64) float64 {
	if len(stages) == 0 {
		return 0
	}
	type stageMass struct {
		dry    float64
		fuel   float64
		thrust float64
	}
	masses := make([]stageMass, len(stages))
	for i, s := range stages {
		dry := s.Engine.MassDry + s.StructuralMass
		if s.Avionics != nil {
			dry += s.Avionics.MassDry
		}
		fuel := s.FuelLoad
		if fuel > s.FuelTank.MassFuelMax {
			fuel = s.FuelTank.MassFuelMax
		}
		masses[i] = stageMass{
			dry:    dry,
			fuel:   fuel,
			thrust: s.Engine.Thrust,
		}
	}
	totalMass := payloadMass
	for _, m := range masses {
		totalMass += m.dry + m.fuel
	}
	if totalMass <= 0 {
		return 0
	}
	maxAccel := 0.0
	for i := 0; i < len(stages); i++ {
		m := masses[i]
		if m.thrust <= 0 || m.fuel <= 0 {
			totalMass -= (m.dry + m.fuel)
			continue
		}
		thrustN := m.thrust * 1000.0
		minMass := totalMass - m.fuel
		if minMass <= 0 {
			break
		}
		accel := thrustN / minMass / 9.80665
		if accel > maxAccel {
			maxAccel = accel
		}
		totalMass -= (m.dry + m.fuel)
		if totalMass <= 0 {
			break
		}
	}
	return maxAccel
}

// CalcStageBurnTime 计算单级燃烧时间 (秒)
func CalcStageBurnTime(stage domain.Stage) float64 {
	if stage.Engine.Thrust <= 0 || stage.FuelLoad <= 0 || stage.Engine.ISP <= 0 {
		return 0
	}
	thrustN := stage.Engine.Thrust * 1000.0
	massFlow := thrustN / (stage.Engine.ISP * 9.80665)
	if massFlow <= 0 {
		return 0
	}
	return stage.FuelLoad / massFlow
}

// ValidateRocketDesign 验证设计是否满足加速度限制
func ValidateRocketDesign(design *domain.RocketDesign, maxAllowedG float64) error {
	if maxAllowedG <= 0 {
		return nil
	}
	accel, err := CalcMaxAccelFromModules(design.Modules, design.PayloadMass)
	if err != nil {
		return err
	}
	if accel > maxAllowedG {
		return &TooHighAccelError{MaxG: maxAllowedG, ActualG: accel}
	}
	return nil
}

type TooHighAccelError struct {
	MaxG    float64
	ActualG float64
}

func (e *TooHighAccelError) Error() string {
	return "最大加速度 " + strconv.FormatFloat(e.ActualG, 'f', 2, 64) + "G 超过限制 " + strconv.FormatFloat(e.MaxG, 'f', 2, 64) + "G"
}

// SimulateLaunchFailure 模拟发射故障
func SimulateLaunchFailure(stages []domain.Stage) bool {
	for _, s := range stages {
		if rand.Float64() < s.Engine.FailureRate {
			return true
		}
		if rand.Float64() < s.FuelTank.FailureRate {
			return true
		}
		if s.Avionics != nil && rand.Float64() < s.Avionics.FailureRate {
			return true
		}
	}
	return false
}
