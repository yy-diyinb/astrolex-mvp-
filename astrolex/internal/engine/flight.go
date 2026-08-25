package engine

import (
	"fmt"
	"time"

	"astrolex/internal/domain"
)

// FlightEvent 表示发射过程中的一个事件
type FlightEvent struct {
	TimeOffset float64 // 从点火起的秒数
	Message    string
	IsFailure  bool // 是否为故障事件（用于中断后续输出）
}

// GenerateFlightEvents 生成发射事件列表
// 参数:
//   - stages: 已配对的级列表（由 PairModules 返回）
//   - success: 发射是否成功（由故障模拟决定）
// 返回: 事件列表，按时间升序
func GenerateFlightEvents(stages []domain.Stage, success bool) []FlightEvent {
	var events []FlightEvent
	currentTime := 0.0

	// 固定事件：点火
	events = append(events, FlightEvent{TimeOffset: 0, Message: "🚀 引擎点火，火箭升空！"})

	// 最大动压（固定 5s）
	events = append(events, FlightEvent{TimeOffset: 5, Message: "通过最大动压区。"})

	// 逐级处理
	for i, stage := range stages {
		burnTime := CalcStageBurnTime(stage)
		if burnTime <= 0 {
			continue // 如果燃烧时间为0，跳过（但通常不会）
		}
		// 该级结束时间
		sepTime := currentTime + burnTime

		if success {
			// 成功：级分离
			events = append(events, FlightEvent{
				TimeOffset: sepTime,
				Message:    fmt.Sprintf("第 %d 级燃料耗尽，关机分离。", i+1),
			})

			if i == len(stages)-1 {
				// 最后一级，入轨成功
				events = append(events, FlightEvent{
					TimeOffset: sepTime + 1,
					Message:    "✅ 载荷成功入轨！发射任务完成。",
				})
			} else {
				// 下一级点火
				nextIgnite := sepTime + 1
				events = append(events, FlightEvent{
					TimeOffset: nextIgnite,
					Message:    fmt.Sprintf("第 %d 级发动机点火。", i+2),
				})
				// 整流罩抛离（在第二级点火后 10s，仅当这是第一级分离时）
				if i == 0 {
					fairingTime := nextIgnite + 10
					events = append(events, FlightEvent{
						TimeOffset: fairingTime,
						Message:    "🧨 整流罩抛离。",
					})
				}
			}
		} else {
			// 失败：在燃烧过程中插入故障事件（约 30% 燃烧时间时）
			failTime := currentTime + burnTime*0.3
			if failTime < 10 {
				failTime = 10 // 至少 10s 后故障，以确保有前序事件
			}
			events = append(events, FlightEvent{
				TimeOffset: failTime,
				Message:    fmt.Sprintf("💥 第 %d 级引擎突发故障，推力丧失！", i+1),
				IsFailure:  true,
			})
			// 故障后不再产生后续事件，直接跳出
			break
		}

		// 更新当前时间
		currentTime = sepTime
	}

	// 如果成功但未生成最后的入轨事件（预防逻辑漏洞）
	if success && len(events) > 0 {
		last := events[len(events)-1]
		if last.Message != "✅ 载荷成功入轨！发射任务完成。" {
			events = append(events, FlightEvent{
				TimeOffset: currentTime + 1,
				Message:    "✅ 载荷成功入轨！发射任务完成。",
			})
		}
	}
	return events
}

// PrintFlightEvents 按顺序打印事件，并在每个事件之间插入延迟
// 参数:
//   - events: 事件列表
//   - delay: 每个事件之间的延迟时间（如 200ms）
func PrintFlightEvents(events []FlightEvent, delay time.Duration) {
	for _, e := range events {
		// 如果事件是故障，打印后立即跳出
		if e.IsFailure {
			fmt.Printf("[T+%.0fs] %s\n", e.TimeOffset, e.Message)
			break
		}
		fmt.Printf("[T+%.0fs] %s\n", e.TimeOffset, e.Message)
		if delay > 0 {
			time.Sleep(delay)
		}
	}
}
