package main

var wrightstoneCN = map[string]string{
	"Dread Wrightstone":         "恐惧祝福",
	"Vitality Wrightstone":      "生命祝福",
	"Fortification Wrightstone": "坚守祝福",
	"Sequestration Wrightstone": "隔绝祝福",
}

var wrightstoneTraitCN = map[string]string{
	"Blight Resistance":             "灾祸抗性",
	"Fast Learner":                  "获得经验值",
	"Rupie Tycoon":                  "获得金币",
	"Path to Mastery":               "获得MSP",
	"Paralysis Resistance":          "麻痹抗性",
	"Skybound Arts Seal Resistance": "奥义封印抗性",
	"Skill Seal Resistance":         "能力封印抗性",
	"Frozen Resistance":             "冰冻抗性",
	"Sandtomb Resistance":           "泥沙抗性",
	"Healing Cap":                   "回复性能",
	"DEF Down Resistance":           "防御DOWN抗性",
	"Stun Resistance":               "昏迷抗性",
	"Poison Resistance":             "中毒抗性",
	"Anomaly Resistance":            "异能耐受",
	"Waterprison Resistance":        "水牢抗性",
	"Burn Resistance":               "灼热抗性",
	"Slow Resistance":               "缓速抗性",
}

func cnWrightstoneTrait(en string) string {
	if !useChinese() {
		return en
	}
	if v, ok := wrightstoneTraitCN[en]; ok {
		return v
	}
	return cnTrait(en)
}

func cnWrightstone(en string) string {
	if !useChinese() {
		return en
	}
	if v, ok := wrightstoneCN[en]; ok {
		return v
	}
	return en
}
