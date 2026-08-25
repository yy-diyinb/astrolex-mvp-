package engine

import (
	"errors"
	"math"
	"time"

	"astrolex/internal/domain"
)

// ApplyHohmannTransfer 从当前状态执行霍曼转移到目标天体（到达其高轨道）
// 返回: 新状态、消耗Δv(m/s)、飞行时间(天)、错误
func ApplyHohmannTransfer(state *FlybyState, targetBody domain.CelestialBody, currentTime time.Time) (*FlybyState, float64, float64, error) {
	// 获取目标天体在 currentTime 的位置（日心）
	tx, ty := PositionAtTime(targetBody, currentTime)

	// 当前状态的位置（应已处于日心坐标系）
	// 计算当前轨道半径（近似为距太阳距离）
	r1 := math.Sqrt(state.Pos[0]*state.Pos[0] + state.Pos[1]*state.Pos[1])
	if r1 == 0 {
		return nil, 0, 0, errors.New("当前位置异常")
	}
	// 目标轨道半径（目标天体到太阳的距离）
	r2 := math.Sqrt(tx*tx + ty*ty)
	if r2 == 0 {
		return nil, 0, 0, errors.New("目标位置异常")
	}

	// 计算霍曼转移参数
	at := (r1 + r2) / 2.0
	if at <= 0 {
		return nil, 0, 0, errors.New("转移轨道半长轴无效")
	}
	// 当前轨道速度（圆轨道近似）
	v1 := math.Sqrt(SunGM / r1)
	// 转移轨道在起点速度
	vt1 := math.Sqrt(SunGM * (2/r1 - 1/at))
	// 目标轨道速度（圆轨道近似）
	v2 := math.Sqrt(SunGM / r2)
	// 转移轨道在终点速度
	vt2 := math.Sqrt(SunGM * (2/r2 - 1/at))

	// 所需Δv（起点加速+终点减速）
	dv1 := vt1 - v1
	dv2 := v2 - vt2
	if dv1 < 0 {
		dv1 = -dv1
	}
	if dv2 < 0 {
		dv2 = -dv2
	}
	dv_kmps := dv1 + dv2
	dv_mps := dv_kmps * 1000.0

	// 检查剩余Δv是否足够
	if state.DeltaVRemaining < dv_mps {
		return nil, 0, 0, errors.New("剩余Δv不足")
	}

	// 计算飞行时间（转移轨道半周期）
	periodTrans := 2 * math.Pi * math.Sqrt(at*at*at/SunGM) // 秒
	flightTimeDays := periodTrans / 2 / Day

	// 计算到达位置（按比例内插，近似为在目标位置）
	// 简化：到达时飞船位置与目标天体相同
	newPos := [2]float64{tx, ty}
	// 到达后，飞船速度应等于目标轨道速度（圆轨道速度），方向垂直于径向（逆时针）
	// 计算目标轨道速度方向（垂直于径向）
	theta := math.Atan2(ty, tx)
	// 径向单位向量 (cos theta, sin theta)
	// 切向单位向量（逆时针）: (-sin theta, cos theta)
	vx := -v2 * math.Sin(theta)
	vy := v2 * math.Cos(theta)

	newState := *state
	newState.Pos = newPos
	newState.Vel = [2]float64{vx, vy}
	newState.DeltaVRemaining -= dv_mps
	newState.CurrentBodyID = targetBody.ID

	return &newState, dv_mps, flightTimeDays, nil
}

// ApplyDirectArrival 直接以当前速度到达目标（不消耗Δv），仅更新位置为当前位置（通常用于已经是目标天体附近）
func ApplyDirectArrival(state *FlybyState, targetBody domain.CelestialBody, currentTime time.Time) (*FlybyState, error) {
	tx, ty := PositionAtTime(targetBody, currentTime)
	newState := *state
	newState.Pos = [2]float64{tx, ty}
	newState.CurrentBodyID = targetBody.ID
	return &newState, nil
}
