package main

import "fmt"

const summonTraitMaxLevel = 15

type SummonDef struct {
	Hash        string `json:"hash"`
	DisplayName string `json:"displayName"`
	BaseName    string `json:"baseName"`
	TypeName    string `json:"typeName"`
	Cost        int    `json:"cost"`
}

type SummonTraitDef struct {
	Hash         string `json:"hash"`
	DisplayName  string `json:"displayName"`
	GameMaxLevel *int   `json:"gameMaxLevel"`
}

type SummonSubTraitDef struct {
	Hash        string    `json:"hash"`
	DisplayName string    `json:"displayName"`
	Values      []float64 `json:"values"`
	MaxLevel    int       `json:"maxLevel"`
	IsPercent   bool      `json:"isPercent"`
	Tier        string    `json:"tier"`
}

type SummonCatalog struct {
	Summons      []SummonDef
	Traits       []SummonTraitDef
	SubTraits    []SummonSubTraitDef
	summonByHash map[uint32]*SummonDef
	traitByHash  map[uint32]*SummonTraitDef
	subByHash    map[uint32]*SummonSubTraitDef
}

func LoadSummonCatalog() (*SummonCatalog, error) {
	file, err := loadJSON[struct {
		Summons   []SummonDef         `json:"summons"`
		Traits    []SummonTraitDef    `json:"traits"`
		SubTraits []SummonSubTraitDef `json:"subTraits"`
	}]("data/summons.json")
	if err != nil {
		return nil, fmt.Errorf("加载召唤石数据失败: %w", err)
	}

	c := &SummonCatalog{
		Summons:   file.Summons,
		Traits:    file.Traits,
		SubTraits: file.SubTraits,
	}
	c.summonByHash = make(map[uint32]*SummonDef, len(c.Summons))
	for i := range c.Summons {
		if h, err := ParseHashHex(c.Summons[i].Hash); err == nil {
			c.summonByHash[h] = &c.Summons[i]
		}
	}
	c.traitByHash = make(map[uint32]*SummonTraitDef, len(c.Traits))
	for i := range c.Traits {
		if h, err := ParseHashHex(c.Traits[i].Hash); err == nil {
			c.traitByHash[h] = &c.Traits[i]
		}
	}
	c.subByHash = make(map[uint32]*SummonSubTraitDef, len(c.SubTraits))
	for i := range c.SubTraits {
		if h, err := ParseHashHex(c.SubTraits[i].Hash); err == nil {
			c.subByHash[h] = &c.SubTraits[i]
		}
	}
	return c, nil
}

func (c *SummonCatalog) LookupSummonByHash(hash uint32) *SummonDef {
	return c.summonByHash[hash]
}

func (c *SummonCatalog) LookupTraitByHash(hash uint32) *SummonTraitDef {
	return c.traitByHash[hash]
}

func (c *SummonCatalog) LookupSubTraitByHash(hash uint32) *SummonSubTraitDef {
	return c.subByHash[hash]
}
