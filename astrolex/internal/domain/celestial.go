package domain

import "time"

// StarSystem 包含所有天体
type StarSystem struct {
	CelestialBodies map[string]CelestialBody `json:"celestial_bodies"`
}

// OrbitalLayer 定义天体的一个轨道层
type OrbitalLayer struct {
	Name            string  `json:"name"`              // 层名称，如 "低轨道"
	AltitudeMin     float64 `json:"altitude_min"`      // 最小高度 (km)
	AltitudeMax     float64 `json:"altitude_max"`      // 最大高度 (km)
	TypicalAltitude float64 `json:"typical_altitude"`  // 典型高度 (km)
}

// CelestialBody 表示一个恒星、行星或卫星
type CelestialBody struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ParentID    string  `json:"parent_id"` // 绕转中心天体ID，空表示恒星
	Mass        float64 `json:"mass"`      // kg
	Radius      float64 `json:"radius"`    // km
	RotationPeriod float64 `json:"rotation_period"` // 小时

	// 开普勒轨道根数（J2000历元）
	SemiMajorAxis     float64 `json:"semi_major_axis"`
	Eccentricity      float64 `json:"eccentricity"`
	Inclination       float64 `json:"inclination"`
	LongitudeAscNode  float64 `json:"longitude_asc_node"`
	ArgumentPeriapsis float64 `json:"argument_periapsis"`
	MeanAnomaly       float64 `json:"mean_anomaly"`

	// 工程参数
	SurfaceGravity   float64 `json:"surface_gravity"`
	DeltaVToLowOrbit float64 `json:"delta_v_to_low_orbit"`

	// 轨道力学字段
	GM                  float64   `json:"gm"`
	Period              float64   `json:"period"`
	Epoch               time.Time `json:"epoch"`
	MeanAnomalyAtEpoch  float64   `json:"mean_anomaly_at_epoch"`

	// ---- 新增：轨道层与希尔球 ----
	OrbitalLayers    []OrbitalLayer `json:"orbital_layers"`      // 该天体的轨道层列表（从低到高）
	HillSphereRadius float64        `json:"hill_sphere_radius"`  // 希尔球半径 (km)
}
