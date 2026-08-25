package engine

import (
	"errors"
	"math"
	"time"

	"astrolex/internal/domain"
)

// Constants (km, s, rad)
const (
	AU   = 149597870.7 // km
	Day  = 86400.0     // seconds
	SunGM = 1.32712440018e11 // km^3/s^2
)

// ==================== Kepler Equation ====================

// KeplerSolve solves Kepler's equation E - e*sin(E) = M using Newton-Raphson
func KeplerSolve(M, e float64) float64 {
	if e == 0 {
		return M
	}
	E := M
	for i := 0; i < 20; i++ {
		dE := (E - e*math.Sin(E) - M) / (1 - e*math.Cos(E))
		E -= dE
		if math.Abs(dE) < 1e-8 {
			break
		}
	}
	return E
}

// MeanMotion returns mean motion n = 2*pi / Period (rad/day)
func MeanMotion(body domain.CelestialBody) float64 {
	if body.Period <= 0 {
		return 0
	}
	return 2 * math.Pi / body.Period
}

// PositionAtTime computes 2D position (x, y) of a body at time t.
func PositionAtTime(body domain.CelestialBody, t time.Time) (x, y float64) {
	dt := t.Sub(body.Epoch).Hours() / 24.0
	if dt < 0 {
		dt = 0
	}
	M0 := body.MeanAnomalyAtEpoch
	n := MeanMotion(body)
	M := M0 + n*dt
	M = math.Mod(M, 2*math.Pi)
	if M < 0 {
		M += 2 * math.Pi
	}
	E := KeplerSolve(M, body.Eccentricity)
	nu := 2 * math.Atan2(math.Sqrt(1+body.Eccentricity)*math.Sin(E/2), math.Sqrt(1-body.Eccentricity)*math.Cos(E/2))
	r := body.SemiMajorAxis * (1 - body.Eccentricity*math.Cos(E))
	x = r * math.Cos(nu)
	y = r * math.Sin(nu)
	return
}

// OrbitalVelocity returns velocity vector (vx, vy) of a body at time t.
func OrbitalVelocity(body domain.CelestialBody, t time.Time) (vx, vy float64) {
	x, y := PositionAtTime(body, t)
	r := math.Sqrt(x*x + y*y)
	if r <= 0 {
		return 0, 0
	}
	GM := SunGM
	v := math.Sqrt(GM * (2/r - 1/body.SemiMajorAxis))
	if v <= 0 {
		return 0, 0
	}
	vx = -v * y / r
	vy = v * x / r
	return
}

// SurfaceVelocity ...
func SurfaceVelocity(base domain.Base, body domain.CelestialBody, t time.Time) (vx, vy float64) {
	ovx, ovy := OrbitalVelocity(body, t)
	latRad := base.Latitude * math.Pi / 180.0
	rotSpeed := 2 * math.Pi * body.Radius * math.Cos(latRad) / (body.RotationPeriod * 3600.0)
	orbV := math.Sqrt(ovx*ovx + ovy*ovy)
	if orbV < 1e-9 {
		return ovx + rotSpeed, ovy
	}
	ux := ovx / orbV
	uy := ovy / orbV
	return ovx + rotSpeed*ux, ovy + rotSpeed*uy
}

// HohmannTransfer ...
func HohmannTransfer(body1, body2 domain.CelestialBody, t time.Time) (dv float64, flightTimeDays float64, phaseAngleRad float64, err error) {
	r1 := body1.SemiMajorAxis
	r2 := body2.SemiMajorAxis
	if r1 <= 0 || r2 <= 0 {
		return 0, 0, 0, errors.New("invalid semi-major axis")
	}
	GM := SunGM
	at := (r1 + r2) / 2
	if at <= 0 {
		return 0, 0, 0, errors.New("invalid transfer orbit")
	}
	v1_circ := math.Sqrt(GM / r1)
	v2_circ := math.Sqrt(GM / r2)
	v_trans1 := math.Sqrt(GM * (2/r1 - 1/at))
	v_trans2 := math.Sqrt(GM * (2/r2 - 1/at))
	dv1 := v_trans1 - v1_circ
	dv2 := v2_circ - v_trans2
	dv = math.Abs(dv1) + math.Abs(dv2)
	periodTrans := 2 * math.Pi * math.Sqrt(at*at*at/GM)
	flightTimeDays = periodTrans / 2 / Day
	phaseAngleRad = math.Pi * (1 - math.Pow((r1+r2)/(2*r2), 1.5))
	if phaseAngleRad < 0 {
		phaseAngleRad += 2 * math.Pi
	}
	return
}

// NextWindow ...
func NextWindow(body1, body2 domain.CelestialBody, currentTime time.Time, searchDays int) (windowStart, windowEnd time.Time, waitDays int, dv float64, err error) {
	_, _, phaseReq, err := HohmannTransfer(body1, body2, currentTime)
	if err != nil {
		return currentTime, currentTime, 0, 0, err
	}
	tolerance := 0.05
	found := false
	var bestDay int
	for i := 0; i < searchDays; i++ {
		t := currentTime.AddDate(0, 0, i)
		x1, y1 := PositionAtTime(body1, t)
		x2, y2 := PositionAtTime(body2, t)
		theta1 := math.Atan2(y1, x1)
		theta2 := math.Atan2(y2, x2)
		phase := theta2 - theta1
		phase = math.Mod(phase, 2*math.Pi)
		if phase < 0 {
			phase += 2 * math.Pi
		}
		diff := phase - phaseReq
		diff = math.Mod(diff, 2*math.Pi)
		if diff > math.Pi {
			diff -= 2 * math.Pi
		}
		if math.Abs(diff) < tolerance {
			found = true
			bestDay = i
			break
		}
	}
	if !found {
		return currentTime, currentTime, 0, 0, errors.New("no window found in search range")
	}
	windowStart = currentTime.AddDate(0, 0, bestDay-1)
	windowEnd = currentTime.AddDate(0, 0, bestDay+1)
	waitDays = bestDay
	dv, _, _, _ = HohmannTransfer(body1, body2, currentTime)
	return
}

// ==================== Orbit Layer Helpers ====================

// CircularOrbitVelocity ...
func CircularOrbitVelocity(gm, radius float64) float64 {
	return math.Sqrt(gm / radius)
}

// EscapeVelocity ...
func EscapeVelocity(gm, radius float64) float64 {
	return math.Sqrt(2 * gm / radius)
}

// OrbitalLayerDeltaV ...
func OrbitalLayerDeltaV(gm, fromRadius, toRadius float64) float64 {
	if fromRadius <= 0 || toRadius <= 0 || fromRadius == toRadius {
		return 0
	}
	at := (fromRadius + toRadius) / 2
	v1 := CircularOrbitVelocity(gm, fromRadius)
	v2 := CircularOrbitVelocity(gm, toRadius)
	vt1 := math.Sqrt(gm * (2/fromRadius - 1/at))
	vt2 := math.Sqrt(gm * (2/toRadius - 1/at))
	if toRadius > fromRadius {
		return (vt1 - v1) + (v2 - vt2)
	} else {
		return (v1 - vt1) + (vt2 - v2)
	}
}

// EscapeFromOrbitDeltaV ...
func EscapeFromOrbitDeltaV(gm, radius float64) float64 {
	v_c := CircularOrbitVelocity(gm, radius)
	v_esc := EscapeVelocity(gm, radius)
	return v_esc - v_c
}

// ==================== 引力辅助与状态更新 ====================

// FlybyState 表示飞船在日心坐标系中的状态
type FlybyState struct {
	Pos             [2]float64 // 日心位置 (km)
	Vel             [2]float64 // 日心速度 (km/s)
	DeltaVRemaining float64    // 剩余 Δv (m/s)
	CurrentBodyID   string     // 当前所在天体ID（或 "deep_space"）
}

// ComputeFlybyState 计算引力辅助后的新状态
func ComputeFlybyState(state *FlybyState, planetBody domain.CelestialBody, periapsis float64, t time.Time) (*FlybyState, float64, error) {
	// 行星状态
	px, py := PositionAtTime(planetBody, t)
	pvx, pvy := OrbitalVelocity(planetBody, t)

	// 相对速度
	relVx := state.Vel[0] - pvx
	relVy := state.Vel[1] - pvy
	vInfMag := math.Sqrt(relVx*relVx + relVy*relVy)
	if vInfMag == 0 {
		return nil, 0, errors.New("相对速度为零，无法飞掠")
	}

	if periapsis < planetBody.Radius {
		periapsis = planetBody.Radius + 100
	}
	rP := planetBody.Radius + periapsis
	GM := planetBody.GM
	e := 1 + (rP*vInfMag*vInfMag)/GM
	if e <= 1 {
		return nil, 0, errors.New("无法形成双曲线轨道，近心点太低")
	}
	delta := 2 * math.Asin(1/e)

	// 飞掠方向
	planetSpeed := math.Sqrt(pvx*pvx + pvy*pvy)
	if planetSpeed == 0 {
		return nil, 0, errors.New("行星速度为0")
	}
	pvUnitX := pvx / planetSpeed
	pvUnitY := pvy / planetSpeed
	proj := relVx*pvUnitX + relVy*pvUnitY
	sign := 1.0
	if proj < 0 {
		sign = -1.0
	}
	rotAngle := -sign * delta
	cosA := math.Cos(rotAngle)
	sinA := math.Sin(rotAngle)
	newRelVx := relVx*cosA - relVy*sinA
	newRelVy := relVx*sinA + relVy*cosA

	newVx := pvx + newRelVx
	newVy := pvy + newRelVy

	dvX := newVx - state.Vel[0]
	dvY := newVy - state.Vel[1]
	dv := math.Sqrt(dvX*dvX + dvY*dvY) // km/s
	dv_mps := dv * 1000

	if state.DeltaVRemaining < dv_mps {
		return nil, 0, errors.New("剩余Δv不足")
	}

	newState := *state
	newState.Vel[0] = newVx
	newState.Vel[1] = newVy
	newState.Pos[0] = px
	newState.Pos[1] = py
	newState.DeltaVRemaining -= dv_mps
	newState.CurrentBodyID = planetBody.ID

	return &newState, dv_mps, nil
}

// PlanNextFlyby 推荐下一个最佳飞掠目标
func PlanNextFlyby(state *FlybyState, starSystem domain.StarSystem, currentTime time.Time, periapsisOverride float64) (string, float64, *FlybyState, error) {
	var bestPlanetID string
	var bestDV float64 = 1e18
	var bestState *FlybyState

	for id, body := range starSystem.CelestialBodies {
		if body.ParentID == "" {
			continue
		}
		if id == state.CurrentBodyID {
			continue
		}
		periapsis := body.Radius + 500.0
		if periapsisOverride > 0 {
			periapsis = periapsisOverride
		}
		newState, dv, err := ComputeFlybyState(state, body, periapsis, currentTime)
		if err != nil {
			continue
		}
		if dv < bestDV {
			bestDV = dv
			bestPlanetID = id
			bestState = newState
		}
	}
	if bestPlanetID == "" {
		return "", 0, nil, errors.New("没有可行的飞掠目标")
	}
	return bestPlanetID, bestDV, bestState, nil
}
