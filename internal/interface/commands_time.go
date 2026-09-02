package repl

import (
	"fmt"

	"astrolex/internal/i18n"
)

// ==================== 时间管理命令 ====================

// showDate 显示当前游戏时间
func (r *Repl) showDate() {
	fmt.Printf(i18n.T("date_current")+"\n", r.game.CurrentTime.Format("2006-01-02 15:04:05"))
}

// advanceTime 推进游戏时间并检查过期合约
func (r *Repl) advanceTime(days int) {
	r.game.CurrentTime = r.game.CurrentTime.AddDate(0, 0, days)
	fmt.Printf(i18n.T("tick_advanced")+"\n", days, r.game.CurrentTime.Format("2006-01-02"))
	expiredCount := 0
	for i := range r.game.Contracts {
		contract := &r.game.Contracts[i]
		if contract.Status == "Accepted" && r.game.CurrentTime.After(contract.DeliveryDeadline) {
			contract.Status = "Failed"
			contract.Completed = true
			if r.activeContractID == contract.ID {
				r.activeContractID = ""
			}
			fmt.Printf(i18n.T("tick_expired")+"\n", contract.ID)
			expiredCount++
		}
	}
	if expiredCount == 0 {
		fmt.Println(i18n.T("tick_no_expired"))
	}
}
