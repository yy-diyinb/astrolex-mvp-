package repl

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"astrolex/internal/domain"
)

// ==================== 合约生成 ====================
func (r *Repl) generateContract() {
	_ = time.Now
	var targets []domain.CelestialBody
	for _, body := range r.game.StarSystem.CelestialBodies {
		if body.ParentID != "" && body.ID != "earth" {
			targets = append(targets, body)
		}
	}
	if len(targets) == 0 {
		fmt.Println("错误: 没有可用的目标天体")
		return
	}
	target := targets[rand.Intn(len(targets))]
	payloadMass := 1000 + rand.Float64()*49000
	var forbidden []string
	if len(r.game.SuppliersDB) > 0 && rand.Float64() < 0.5 {
		for id := range r.game.SuppliersDB {
			forbidden = append(forbidden, id)
			break
		}
	}
	// 使用配置计算奖励
	cfg := r.cfg.Contract
	reward := int64(cfg.RewardBase + payloadMass*cfg.RewardMassFactor + (target.SemiMajorAxis/149600000.0)*cfg.RewardDistanceFactor)
	if reward < 1000 {
		reward = 1000
	}
	deadline := r.game.CurrentTime.AddDate(0, 0, 30+rand.Intn(60))
	contractID := fmt.Sprintf("contract_%d", len(r.game.Contracts)+1)
	contract := domain.Contract{
		ID:                 contractID,
		Issuer:             "地球联合政府",
		TargetBodyID:       target.ID,
		PayloadMass:        payloadMass,
		PayloadVolume:      0,
		MaxAccelLimit:      0,
		ForbiddenSuppliers: forbidden,
		RequiredInsurance:  "Basic",
		DeliveryDeadline:   deadline,
		RewardCredits:      reward,
		PenaltyCredits:     reward / 2,
		Status:             "Open",
		Budget:             0,
		HardwareReturn:     0,
		Launches:           []domain.LaunchMission{},
		TotalCost:          0,
		Completed:          false,
		TargetPayloadDelivered: 0,
		// 新字段初始化
		BudgetUsed:         0,
		BudgetHardwareCost: 0,
		PlayerPaid:         0,
	}
	r.game.Contracts = append(r.game.Contracts, contract)
	fmt.Printf("\n=== 新合约已生成 ===\n")
	fmt.Printf("ID: %s\n", contract.ID)
	fmt.Printf("目标: %s\n", target.Name)
	fmt.Printf("载荷质量: %.0f kg\n", contract.PayloadMass)
	if len(contract.ForbiddenSuppliers) > 0 {
		fmt.Printf("禁用供应商: %s\n", strings.Join(contract.ForbiddenSuppliers, ", "))
	}
	fmt.Printf("奖励: %d 信用点\n", contract.RewardCredits)
	fmt.Printf("罚金: %d 信用点\n", contract.PenaltyCredits)
	fmt.Printf("截止日期: %s\n", contract.DeliveryDeadline.Format("2006-01-02"))
	fmt.Printf("状态: %s\n", contract.Status)
	fmt.Printf("\n使用 accept %s 接受此合约\n", contract.ID)
}

// ==================== 接受合约（生成预算） ====================
func (r *Repl) acceptContract(contractID string) {
	var found *domain.Contract
	for i := range r.game.Contracts {
		if r.game.Contracts[i].ID == contractID {
			found = &r.game.Contracts[i]
			break
		}
	}
	if found == nil {
		fmt.Printf("错误: 未找到合约 ID '%s'\n", contractID)
		return
	}
	if found.Status != "Open" {
		fmt.Printf("合约 %s 状态为 '%s'，无法接受\n", contractID, found.Status)
		return
	}

	target, ok := r.game.StarSystem.CelestialBodies[found.TargetBodyID]
	if !ok {
		fmt.Println("错误: 目标天体不存在")
		return
	}
	distanceAU := target.SemiMajorAxis / 149600000.0
	cfg := r.cfg.Contract
	budget := int64(cfg.BudgetBase + found.PayloadMass*cfg.BudgetMassFactor + distanceAU*cfg.BudgetDistanceFactor)
	returnRate := cfg.ReturnMin + rand.Float64()*(cfg.ReturnMax-cfg.ReturnMin)

	found.Status = "Accepted"
	found.Budget = budget
	found.HardwareReturn = returnRate
	found.TotalCost = 0
	found.TargetPayloadDelivered = 0
	found.Launches = []domain.LaunchMission{}
	found.Completed = false

	// 初始化新字段
	found.BudgetUsed = 0
	found.BudgetHardwareCost = 0
	found.PlayerPaid = 0

	r.activeContractID = contractID

	fmt.Printf("✅ 已接受合约 %s\n", contractID)
	fmt.Printf("目标: %s\n", target.Name)
	fmt.Printf("载荷质量: %.0f kg\n", found.PayloadMass)
	fmt.Printf("预算: %d 信用点\n", budget)
	fmt.Printf("硬件返还比例: %.0f%%\n", returnRate*100)
	fmt.Printf("截止日期: %s\n", found.DeliveryDeadline.Format("2006-01-02"))
	if len(found.ForbiddenSuppliers) > 0 {
		fmt.Printf("⚠️ 注意: 禁用供应商: %s\n", strings.Join(found.ForbiddenSuppliers, ", "))
	}
	fmt.Println("\n现在可以使用 launch <设计ID> 执行发射，使用 done 结束任务，使用 budget 查看预算。")
}

// ==================== 结束合约并结算 ====================
func (r *Repl) doneContract() {
	if r.activeContractID == "" {
		fmt.Println("当前没有进行中的合约。")
		return
	}
	var contract *domain.Contract
	for i := range r.game.Contracts {
		if r.game.Contracts[i].ID == r.activeContractID {
			contract = &r.game.Contracts[i]
			break
		}
	}
	if contract == nil {
		fmt.Println("错误: 当前合约不存在")
		r.activeContractID = ""
		return
	}
	if contract.Status != "Accepted" {
		fmt.Printf("合约 %s 状态为 '%s'，无法结束\n", contract.ID, contract.Status)
		r.activeContractID = ""
		return
	}

	success := contract.TargetPayloadDelivered >= contract.PayloadMass && !r.game.CurrentTime.After(contract.DeliveryDeadline)

	if success {
		reward := contract.RewardCredits
		r.game.Player.Credits += reward
		contract.Status = "Completed"
		contract.Completed = true
		fmt.Printf("🎉 合约 %s 完成！获得奖金 %d 信用点\n", contract.ID, reward)
		fmt.Printf("总预算: %d, 预算已用: %d, 玩家超支: %d\n", contract.Budget, contract.BudgetUsed, contract.PlayerPaid)
		fmt.Printf("任务总花费: %d 信用点\n", contract.TotalCost)
	} else {
		// ---- 失败返还：只返还预算内硬件成本 ----
		returnAmount := int64(float64(contract.BudgetHardwareCost) * contract.HardwareReturn)
		r.game.Player.Credits += returnAmount
		contract.Status = "Failed"
		contract.Completed = true

		fmt.Printf("❌ 合约 %s 失败\n", contract.ID)
		fmt.Printf("   预算内硬件成本: %d 信用点\n", contract.BudgetHardwareCost)
		fmt.Printf("   返还比例: %.0f%%\n", contract.HardwareReturn*100)
		fmt.Printf("   返还金额: %d 信用点\n", returnAmount)
		if contract.PlayerPaid > 0 {
			fmt.Printf("   玩家超支部分 (不返还): %d 信用点\n", contract.PlayerPaid)
		}
		fmt.Printf("   总预算: %d, 预算已用: %d, 净损失: %d\n",
			contract.Budget, contract.BudgetUsed, contract.BudgetUsed-returnAmount)
	}
	r.activeContractID = ""
	fmt.Printf("当前信用点: %d\n", r.game.Player.Credits)
}

// ==================== 查看预算 ====================
func (r *Repl) showBudget() {
	if r.activeContractID == "" {
		fmt.Println("当前没有进行中的合约。")
		return
	}
	var contract *domain.Contract
	for i := range r.game.Contracts {
		if r.game.Contracts[i].ID == r.activeContractID {
			contract = &r.game.Contracts[i]
			break
		}
	}
	if contract == nil {
		fmt.Println("错误: 当前合约不存在")
		r.activeContractID = ""
		return
	}
	fmt.Printf("合约 %s\n", contract.ID)
	fmt.Printf("目标: %s\n", r.game.StarSystem.CelestialBodies[contract.TargetBodyID].Name)
	fmt.Printf("载荷要求: %.0f kg, 已送达: %.0f kg\n", contract.PayloadMass, contract.TargetPayloadDelivered)
	fmt.Printf("总预算: %d 信用点\n", contract.Budget)
	fmt.Printf("已使用预算: %d 信用点\n", contract.BudgetUsed)
	fmt.Printf("剩余预算: %d 信用点\n", contract.Budget-contract.BudgetUsed)
	fmt.Printf("玩家超支: %d 信用点\n", contract.PlayerPaid)
	fmt.Printf("预算内硬件成本: %d 信用点\n", contract.BudgetHardwareCost)
	fmt.Printf("硬件返还比例: %.0f%%\n", contract.HardwareReturn*100)
	fmt.Printf("截止日期: %s\n", contract.DeliveryDeadline.Format("2006-01-02"))
	if contract.Completed {
		fmt.Printf("状态: 已结束 (%s)\n", contract.Status)
	} else {
		fmt.Printf("状态: 进行中\n")
	}
}
