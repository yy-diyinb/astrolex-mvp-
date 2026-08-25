package repl

import (
	"fmt"
)

// ==================== 显示日期 ====================
func (r *Repl) showDate() {
	fmt.Printf("当前游戏时间: %s\n", r.game.CurrentTime.Format("2006-01-02 15:04:05"))
}

// ==================== 时间推进 ====================
func (r *Repl) advanceTime(days int) {
	r.game.CurrentTime = r.game.CurrentTime.AddDate(0, 0, days)
	fmt.Printf("时间前进 %d 天，当前日期: %s\n", days, r.game.CurrentTime.Format("2006-01-02"))
	expiredCount := 0
	for i := range r.game.Contracts {
		contract := &r.game.Contracts[i]
		if contract.Status == "Accepted" && r.game.CurrentTime.After(contract.DeliveryDeadline) {
			contract.Status = "Failed"
			contract.Completed = true
			if r.activeContractID == contract.ID {
				r.activeContractID = ""
			}
			fmt.Printf("⚠️ 合约 %s 已过期，任务失败\n", contract.ID)
			expiredCount++
		}
	}
	if expiredCount == 0 {
		fmt.Println("没有过期的合约。")
	}
}
