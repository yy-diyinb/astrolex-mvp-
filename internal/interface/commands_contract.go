package repl

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"astrolex/internal/domain"
	"astrolex/internal/i18n"
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
		fmt.Println(i18n.T("contract_no_targets"))
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
		BudgetUsed:         0,
		BudgetHardwareCost: 0,
		PlayerPaid:         0,
	}
	r.game.Contracts = append(r.game.Contracts, contract)
	fmt.Println(i18n.T("contract_generate_title"))
	fmt.Printf(i18n.T("contract_id"), contract.ID)
	fmt.Printf(i18n.T("contract_target"), target.Name)
	fmt.Printf(i18n.T("contract_payload"), contract.PayloadMass)
	if len(contract.ForbiddenSuppliers) > 0 {
		fmt.Printf(i18n.T("contract_forbidden"), strings.Join(contract.ForbiddenSuppliers, ", "))
	}
	fmt.Printf(i18n.T("contract_reward"), contract.RewardCredits)
	fmt.Printf(i18n.T("contract_penalty"), contract.PenaltyCredits)
	fmt.Printf(i18n.T("contract_deadline"), contract.DeliveryDeadline.Format("2006-01-02"))
	fmt.Printf(i18n.T("contract_status"), contract.Status)
	fmt.Printf(i18n.T("contract_accept_hint"), contract.ID)
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
		fmt.Printf(i18n.T("error_contract_not_found"), contractID)
		return
	}
	if found.Status != "Open" {
		fmt.Printf(i18n.T("contract_accept_not_open"), contractID, found.Status)
		return
	}

	target, ok := r.game.StarSystem.CelestialBodies[found.TargetBodyID]
	if !ok {
		fmt.Println(i18n.T("error_body_not_found"))
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
	found.BudgetUsed = 0
	found.BudgetHardwareCost = 0
	found.PlayerPaid = 0

	r.activeContractID = contractID

	fmt.Printf(i18n.T("accept_success"), contractID)
	fmt.Printf(i18n.T("accept_target"), target.Name)
	fmt.Printf(i18n.T("accept_payload"), found.PayloadMass)
	fmt.Printf(i18n.T("accept_budget"), budget)
	fmt.Printf(i18n.T("accept_return"), returnRate*100)
	fmt.Printf(i18n.T("accept_deadline"), found.DeliveryDeadline.Format("2006-01-02"))
	if len(found.ForbiddenSuppliers) > 0 {
		fmt.Printf(i18n.T("accept_forbidden"), strings.Join(found.ForbiddenSuppliers, ", "))
	}
	fmt.Println(i18n.T("accept_launch_hint"))
}

// ==================== 结束合约并结算 ====================

func (r *Repl) doneContract() {
	if r.activeContractID == "" {
		fmt.Println(i18n.T("done_no_contract"))
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
		fmt.Println(i18n.T("error_contract_not_found"))
		r.activeContractID = ""
		return
	}
	if contract.Status != "Accepted" {
		fmt.Printf(i18n.T("done_invalid_status"), contract.ID, contract.Status)
		r.activeContractID = ""
		return
	}

	success := contract.TargetPayloadDelivered >= contract.PayloadMass && !r.game.CurrentTime.After(contract.DeliveryDeadline)

	if success {
		reward := contract.RewardCredits
		r.game.Player.Credits += reward
		contract.Status = "Completed"
		contract.Completed = true
		fmt.Printf(i18n.T("done_success"), contract.ID, reward)
		fmt.Printf(i18n.T("done_success_summary"), contract.Budget, contract.BudgetUsed, contract.PlayerPaid)
		fmt.Printf(i18n.T("done_success_total"), contract.TotalCost)
	} else {
		returnAmount := int64(float64(contract.BudgetHardwareCost) * contract.HardwareReturn)
		r.game.Player.Credits += returnAmount
		contract.Status = "Failed"
		contract.Completed = true

		fmt.Printf(i18n.T("done_failure"), contract.ID)
		fmt.Printf(i18n.T("done_failure_hardware"), contract.BudgetHardwareCost)
		fmt.Printf(i18n.T("done_failure_return_rate"), contract.HardwareReturn*100)
		fmt.Printf(i18n.T("done_failure_return"), returnAmount)
		if contract.PlayerPaid > 0 {
			fmt.Printf(i18n.T("done_failure_player_paid"), contract.PlayerPaid)
		}
		fmt.Printf(i18n.T("done_failure_summary"),
			contract.Budget, contract.BudgetUsed, contract.BudgetUsed-returnAmount)
	}
	r.activeContractID = ""
	fmt.Printf(i18n.T("done_credits"), r.game.Player.Credits)
}

// ==================== 查看预算 ====================

func (r *Repl) showBudget() {
	if r.activeContractID == "" {
		fmt.Println(i18n.T("budget_no_contract"))
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
		fmt.Println(i18n.T("error_contract_not_found"))
		r.activeContractID = ""
		return
	}
	fmt.Printf(i18n.T("budget_title"), contract.ID)
	fmt.Printf(i18n.T("budget_target"), r.game.StarSystem.CelestialBodies[contract.TargetBodyID].Name)
	fmt.Printf(i18n.T("budget_payload"), contract.PayloadMass, contract.TargetPayloadDelivered)
	fmt.Printf(i18n.T("budget_total"), contract.Budget)
	fmt.Printf(i18n.T("budget_used"), contract.BudgetUsed)
	fmt.Printf(i18n.T("budget_remaining"), contract.Budget-contract.BudgetUsed)
	fmt.Printf(i18n.T("budget_player_paid"), contract.PlayerPaid)
	fmt.Printf(i18n.T("budget_hardware_cost"), contract.BudgetHardwareCost)
	fmt.Printf(i18n.T("budget_return"), contract.HardwareReturn*100)
	fmt.Printf(i18n.T("budget_deadline"), contract.DeliveryDeadline.Format("2006-01-02"))
	if contract.Completed {
		fmt.Printf(i18n.T("budget_status"), contract.Status)
	} else {
		fmt.Println(i18n.T("budget_status_active"))
	}
}
