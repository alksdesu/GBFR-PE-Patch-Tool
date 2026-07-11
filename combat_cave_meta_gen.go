package main

type caveMetaFloat struct {
	Sym     string  `json:"sym"`
	Extra   int     `json:"extra"`
	Label   string  `json:"label"`
	Default float64 `json:"default"`
}

type caveMetaInt struct {
	Sym     string `json:"sym"`
	Extra   int    `json:"extra"`
	Label   string `json:"label"`
	Default int32  `json:"default"`
}

type caveMetaFlag struct {
	Sym   string `json:"sym"`
	Byte  int    `json:"byte"`
	Label string `json:"label"`
}

type CaveMetaEntry struct {
	ID     string          `json:"id"`
	Kind   string          `json:"kind"`
	Desc   string          `json:"desc"`
	PtrSym string          `json:"ptrSym"`
	Floats []caveMetaFloat `json:"floats"`
	Ints   []caveMetaInt   `json:"ints"`
	Flags  []caveMetaFlag  `json:"flags"`
}

var caveMetaTable = []CaveMetaEntry{
	{
		ID:   "player_pointers",
		Kind: "capture",
		Desc: "捕获本地玩家对象指针(HP/奥义槽/坐标基址)",
		PtrSym: "NBGFR01_ptr",
	},
	{
		ID:   "highlighted_item",
		Kind: "capture",
		Desc: "捕获背包中选中物品的对象指针",
		PtrSym: "NBGFR20_ptr",
	},
	{
		ID:   "highlighted_weapon",
		Kind: "capture",
		Desc: "捕获选中武器的对象指针",
		PtrSym: "NBGFR21_ptr",
	},
	{
		ID:   "damage_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR03_flt", Extra: 0, Label: "受到伤害倍率", Default: 2.0},
			{Sym: "NBGFR03_flt", Extra: 4, Label: "造成伤害倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR03_flg", Byte: 0, Label: "无敌(免伤)"},
			{Sym: "NBGFR03_flg", Byte: 1, Label: "不死(保留血量阈值)"},
			{Sym: "NBGFR03_flg", Byte: 2, Label: "一击必杀"},
		},
	},
	{
		ID:   "cooldown_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR04A_flt", Extra: 0, Label: "冷却时间倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR04A_flg", Byte: 0, Label: "技能无冷却"},
		},
	},
	{
		ID:   "status_effect_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR05_flt", Extra: 0, Label: "效果时长倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR05_flg", Byte: 0, Label: "增益无限持续"},
			{Sym: "NBGFR05_flg", Byte: 1, Label: "免疫减益"},
		},
	},
	{
		ID:   "ultra_instinct",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR07A_flt", Extra: 0, Label: "触发距离阈值", Default: 0.08},
		},
	},
	{
		ID:   "stun_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR10A_flt", Extra: 0, Label: "眩晕值倍率", Default: 2.0},
			{Sym: "NBGFR10B_flt", Extra: 0, Label: "眩晕冷却倍率", Default: 5.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR10A_flg", Byte: 0, Label: "瞬间眩晕"},
			{Sym: "NBGFR10B_flg", Byte: 0, Label: "眩晕无冷却"},
		},
	},
	{
		ID:   "enemy_mode_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR17_flt", Extra: 0, Label: "破坏值倍率", Default: 2.0},
			{Sym: "NBGFR17_flt", Extra: 4, Label: "Overdrive值倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR17_flg", Byte: 0, Label: "瞬间破坏"},
			{Sym: "NBGFR17_flg", Byte: 1, Label: "禁用红条(Overdrive)"},
		},
	},
	{
		ID:   "arts_level_modifier",
		Kind: "modifier",
		Ints: []caveMetaInt{
			{Sym: "NBGFR34_int", Extra: 0, Label: "奥义等级", Default: 3},
		},
	},
	{
		ID:   "ares_gauge_gain_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR36_flt", Extra: 0, Label: "战神槽获取倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR36_flg", Byte: 0, Label: "瞬间充满槽"},
		},
	},
	{
		ID:   "combo_d_duration",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR37_flt", Extra: 0, Label: "连段D持续(秒)", Default: 5.0},
		},
	},
	{
		ID:   "bull_s_eye_blast_charge_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR38_flt", Extra: 0, Label: "充能倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR38_flg", Byte: 0, Label: "瞬间充能"},
		},
	},
	{
		ID:   "stargaze_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR40_flt", Extra: 0, Label: "观星消耗倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR40_flg", Byte: 0, Label: "观星不消耗"},
		},
	},
	{
		ID:   "armor_piercing_round_modifier",
		Kind: "modifier",
		Ints: []caveMetaInt{
			{Sym: "NBGFR47A_int", Extra: 0, Label: "穿甲弹层数", Default: 20},
		},
	},
	{
		ID:   "rose_garden_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR48A_flt", Extra: 0, Label: "玫瑰持续(秒)", Default: 90.0},
		},
		Ints: []caveMetaInt{
			{Sym: "NBGFR48A_int", Extra: 0, Label: "玫瑰等级", Default: 3},
		},
	},
	{
		ID:   "sword_flurry_charge_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR64_flt", Extra: 0, Label: "充能倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR64_flg", Byte: 0, Label: "瞬间满充能"},
		},
	},
	{
		ID:   "avatar_duration_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR73_flt", Extra: 0, Label: "化身持续倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR73_flg", Byte: 0, Label: "化身无限持续"},
		},
	},
	{
		ID:   "seven_star_s_brilliance_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR75_flt", Extra: 0, Label: "七星光辉倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR75_flg", Byte: 0, Label: "瞬间满七星"},
		},
	},
	{
		ID:   "multilock_hail_cam_speed",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR77_flt", Extra: 0, Label: "运镜速度倍率", Default: 2.0},
		},
	},
	{
		ID:   "embrasque_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR78_flt", Extra: 0, Label: "恩布拉斯克倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR78_flg", Byte: 0, Label: "无限持续"},
		},
	},
	{
		ID:   "loaded_gauge_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR82_flt", Extra: 0, Label: "装填槽倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR82_flg", Byte: 0, Label: "无限装填槽"},
		},
	},
	{
		ID:   "action_speed_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR31_flt", Extra: 0, Label: "动作速度倍率", Default: 1.5},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR31_flg", Byte: 0, Label: "应用到队友"},
		},
	},
	{
		ID:   "custom_over_mastery",
		Kind: "modifier",
		Ints: []caveMetaInt{
			{Sym: "NBGFR32_int", Extra: 16, Label: "强制过量精通等级", Default: 20},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR32_flg", Byte: 0, Label: "强制满级精通"},
			{Sym: "NBGFR32_flg", Byte: 1, Label: "强制随机精通"},
		},
	},
	{
		ID:   "lock_blade_gauge",
		Kind: "modifier",
	},
	{
		ID:   "instant_fill_blade_gauge",
		Kind: "modifier",
	},
}

func (a *App) CaveMeta() []CaveMetaEntry { return caveMetaTable }
