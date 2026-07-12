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
		ID:     "player_pointers",
		Kind:   "capture",
		Desc:   "捕获本地玩家对象指针(HP/奥义槽/坐标基址)",
		PtrSym: "NBGFR001_ptr",
	},
	{
		ID:   "damage_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR003_flt", Extra: 0, Label: "受到伤害倍率", Default: 2.0},
			{Sym: "NBGFR003_flt", Extra: 4, Label: "造成伤害倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR003_flg", Byte: 0, Label: "无敌(免伤)"},
			{Sym: "NBGFR003_flg", Byte: 1, Label: "不死(保留血量阈值)"},
			{Sym: "NBGFR003_flg", Byte: 2, Label: "一击必杀"},
		},
	},
	{
		ID:   "cooldown_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR004A_flt", Extra: 0, Label: "冷却时间倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR004A_flg", Byte: 0, Label: "技能无冷却"},
		},
	},
	{
		ID:   "status_effect_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR005_flt", Extra: 0, Label: "效果时长倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR005_flg", Byte: 0, Label: "增益无限持续"},
			{Sym: "NBGFR005_flg", Byte: 1, Label: "免疫减益"},
		},
	},
	{
		ID:   "ultra_instinct",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR007A_flt", Extra: 0, Label: "触发距离阈值", Default: 0.08},
		},
	},
	{
		ID:   "stun_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR010A_flt", Extra: 0, Label: "眩晕值倍率", Default: 2.0},
			{Sym: "NBGFR010B_flt", Extra: 0, Label: "眩晕冷却倍率", Default: 5.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR010A_flg", Byte: 0, Label: "瞬间眩晕"},
			{Sym: "NBGFR010B_flg", Byte: 0, Label: "眩晕无冷却"},
		},
	},
	{
		ID:   "enemy_mode_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR018_flt", Extra: 0, Label: "破坏值倍率", Default: 2.0},
			{Sym: "NBGFR018_flt", Extra: 4, Label: "Overdrive值倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR018_flg", Byte: 0, Label: "瞬间破坏"},
			{Sym: "NBGFR018_flg", Byte: 1, Label: "禁用红条(Overdrive)"},
		},
	},
	{
		ID:   "arts_level_modifier",
		Kind: "modifier",
		Ints: []caveMetaInt{
			{Sym: "NBGFR022_int", Extra: 0, Label: "奥义等级", Default: 3},
		},
	},
	{
		ID:   "ares_gauge_gain_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR024_flt", Extra: 0, Label: "战神槽获取倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR024_flg", Byte: 0, Label: "瞬间充满槽"},
		},
	},
	{
		ID:   "combo_d_duration",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR025_flt", Extra: 0, Label: "连段D持续(秒)", Default: 5.0},
		},
	},
	{
		ID:   "bull_s_eye_blast_charge_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR026_flt", Extra: 0, Label: "充能倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR026_flg", Byte: 0, Label: "瞬间充能"},
		},
	},
	{
		ID:   "stargaze_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR028_flt", Extra: 0, Label: "观星消耗倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR028_flg", Byte: 0, Label: "观星不消耗"},
		},
	},
	{
		ID:   "armor_piercing_round_modifier",
		Kind: "modifier",
		Ints: []caveMetaInt{
			{Sym: "NBGFR035A_int", Extra: 0, Label: "穿甲弹层数", Default: 20},
		},
	},
	{
		ID:   "rose_garden_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR036A_flt", Extra: 0, Label: "玫瑰持续(秒)", Default: 90.0},
		},
		Ints: []caveMetaInt{
			{Sym: "NBGFR036A_int", Extra: 0, Label: "玫瑰等级", Default: 3},
		},
	},
	{
		ID:   "sword_flurry_charge_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR052_flt", Extra: 0, Label: "充能倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR052_flg", Byte: 0, Label: "瞬间满充能"},
		},
	},
	{
		ID:   "avatar_duration_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR061_flt", Extra: 0, Label: "化身持续倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR061_flg", Byte: 0, Label: "化身无限持续"},
		},
	},
	{
		ID:   "seven_star_s_brilliance_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR063_flt", Extra: 0, Label: "七星光辉倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR063_flg", Byte: 0, Label: "瞬间满七星"},
		},
	},
	{
		ID:   "multilock_hail_cam_speed",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR065_flt", Extra: 0, Label: "运镜速度倍率", Default: 2.0},
		},
	},
	{
		ID:   "embrasque_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR066_flt", Extra: 0, Label: "恩布拉斯克倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR066_flg", Byte: 0, Label: "无限持续"},
		},
	},
	{
		ID:   "loaded_gauge_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR070_flt", Extra: 0, Label: "装填槽倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR070_flg", Byte: 0, Label: "无限装填槽"},
		},
	},
	{
		ID:   "silver_wolf_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR071_flt", Extra: 0, Label: "银狼槽获取倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR071_flg", Byte: 0, Label: "瞬间满银狼槽"},
		},
	},
	{
		ID:   "stance_gauge_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR075A_flt", Extra: 0, Label: "架势槽获取倍率", Default: 2.0},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR075A_flg", Byte: 0, Label: "瞬间满架势槽"},
		},
	},
	{
		ID:     "highlighted_item",
		Kind:   "capture",
		Desc:   "捕获背包中选中物品的对象指针",
		PtrSym: "NBGFR078_ptr",
	},
	{
		ID:     "highlighted_weapon",
		Kind:   "capture",
		Desc:   "捕获选中武器的对象指针",
		PtrSym: "NBGFR079_ptr",
	},
	{
		ID:   "action_speed_modifier",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFR090_flt", Extra: 0, Label: "动作速度倍率", Default: 1.5},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR090_flg", Byte: 0, Label: "应用到队友"},
		},
	},
	{
		ID:   "custom_over_mastery",
		Kind: "modifier",
		Ints: []caveMetaInt{
			{Sym: "NBGFR091_int", Extra: 16, Label: "强制过量精通等级", Default: 20},
		},
		Flags: []caveMetaFlag{
			{Sym: "NBGFR091_flg", Byte: 0, Label: "强制满级精通"},
			{Sym: "NBGFR091_flg", Byte: 1, Label: "强制随机精通"},
		},
	},
	{
		ID:   "instant_fill_blade_gauge",
		Kind: "modifier",
		Floats: []caveMetaFloat{
			{Sym: "NBGFRMAG2_flt", Extra: 0, Label: "刃重槽充满值", Default: 450.0},
		},
	},
}

func (a *App) CaveMeta() []CaveMetaEntry { return caveMetaTable }
