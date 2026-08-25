package domain

type Part struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`      // Engine, FuelTank, AeroShell, Avionics, Satellite, CargoBay, LifeSupport
	SupplierID   string   `json:"supplier_id"`

	MassDry      float64  `json:"mass_dry"`
	MassFuelMax  float64  `json:"mass_fuel_max"`
	Thrust       float64  `json:"thrust"`
	ISP          float64  `json:"isp"`

	FailureRate  float64  `json:"failure_rate"`
	Diameter     float64  `json:"diameter"`
	DeliveryTime int      `json:"delivery_time"`
	InterfaceTag string   `json:"interface_tag"`

	Cost         int64    `json:"cost"`
	SatelliteID  string   `json:"satellite_id,omitempty"`

	// 货舱容量
	CargoMassCapacity float64 `json:"cargo_mass_capacity"`

	// ---- 新增：控制芯片 ----
	IsControlChip bool `json:"is_control_chip"`

	Capacity     *float64 `json:"capacity,omitempty"`
}

type Supplier struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Owner       string   `json:"owner"`
	PartsList   []string `json:"parts_list"`
	PoliticalStatus string `json:"political_status"`
	DeliveryTimeMod float64 `json:"delivery_time_mod"`
}
