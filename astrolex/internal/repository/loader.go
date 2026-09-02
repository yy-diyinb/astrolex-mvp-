package repository

import (
	"encoding/json"
	"os"

	"astrolex/internal/domain"
)

// LoadStarCatalog 加载星表
func LoadStarCatalog(path string) (domain.StarSystem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.StarSystem{}, err
	}
	var sys domain.StarSystem
	err = json.Unmarshal(data, &sys)
	return sys, err
}

// LoadParts 加载零件库，返回 map[string]domain.Part
func LoadParts(path string) (map[string]domain.Part, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var parts map[string]domain.Part
	err = json.Unmarshal(data, &parts)
	return parts, err
}

// LoadSuppliers 加载供应商，返回 map[string]domain.Supplier
func LoadSuppliers(path string) (map[string]domain.Supplier, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var suppliers map[string]domain.Supplier
	err = json.Unmarshal(data, &suppliers)
	return suppliers, err
}

// LoadBases 加载基地，返回 map[string]domain.Base
func LoadBases(path string) (map[string]domain.Base, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bases map[string]domain.Base
	err = json.Unmarshal(data, &bases)
	return bases, err
}
