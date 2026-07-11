package main

import (
	"fmt"
	"unsafe"
)

type combatPatch struct {
	RVA   uintptr
	Orig  []byte
	Patch []byte
}

type combatPatchFeature struct {
	ID      string
	Name    string
	Group   string
	Patches []combatPatch
}

type CombatPatchState struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Group    string `json:"group"`
	Enabled  bool   `json:"enabled"`
	Partial  bool   `json:"partial"`
	Mismatch bool   `json:"mismatch"`
}

func findCombatPatchFeature(id string) *combatPatchFeature {
	for i := range combatPatchFeatures {
		if combatPatchFeatures[i].ID == id {
			return &combatPatchFeatures[i]
		}
	}
	return nil
}

func (a *App) readCombatPatchByte(rva uintptr, size int) ([]byte, error) {
	buf := make([]byte, size)
	addr := a.moduleBase + rva
	if err := readProcessMemory(a.hProcess, addr, unsafe.Pointer(&buf[0]), uintptr(size)); err != nil {
		return nil, err
	}
	return buf, nil
}

func (a *App) combatPatchFeatureState(f *combatPatchFeature) (CombatPatchState, error) {
	state := CombatPatchState{ID: f.ID, Name: f.Name, Group: f.Group}
	patchedCount := 0
	for _, p := range f.Patches {
		cur, err := a.readCombatPatchByte(p.RVA, len(p.Patch))
		if err != nil {
			return state, fmt.Errorf("读取%s指令失败: %w", f.Name, err)
		}
		if bytesEqual(cur, p.Patch) {
			patchedCount++
		} else if !bytesEqual(cur, p.Orig) {
			state.Mismatch = true
		}
	}
	state.Enabled = patchedCount == len(f.Patches)
	state.Partial = patchedCount > 0 && patchedCount < len(f.Patches)
	return state, nil
}

func (a *App) CombatPatchList() ([]CombatPatchState, error) {
	if err := a.ensureGameProcess(); err != nil {
		return nil, err
	}
	states := make([]CombatPatchState, 0, len(combatPatchFeatures))
	for i := range combatPatchFeatures {
		state, err := a.combatPatchFeatureState(&combatPatchFeatures[i])
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func (a *App) CombatPatchSetEnabled(id string, enabled bool) (CombatPatchState, error) {
	if err := a.ensureGameProcess(); err != nil {
		return CombatPatchState{}, err
	}
	f := findCombatPatchFeature(id)
	if f == nil {
		return CombatPatchState{}, fmt.Errorf("未知战斗功能: %s", id)
	}
	for _, p := range f.Patches {
		want := p.Patch
		other := p.Orig
		if !enabled {
			want = p.Orig
			other = p.Patch
		}
		cur, err := a.readCombatPatchByte(p.RVA, len(p.Patch))
		if err != nil {
			return CombatPatchState{}, fmt.Errorf("读取%s指令失败: %w", f.Name, err)
		}
		if bytesEqual(cur, want) {
			continue
		}
		if !bytesEqual(cur, other) {
			return CombatPatchState{}, fmt.Errorf("%s指令字节异常，可能游戏版本不匹配: %s", f.Name, bytesToHex(cur))
		}
		if err := writeCodeMemory(a.hProcess, a.moduleBase+p.RVA, want); err != nil {
			return CombatPatchState{}, fmt.Errorf("写入%s失败: %w", f.Name, err)
		}
	}
	return a.combatPatchFeatureState(f)
}

func (a *App) combatPatchRestoreAll() {
	if a.hProcess == 0 || a.moduleBase == 0 {
		return
	}
	for i := range combatPatchFeatures {
		f := &combatPatchFeatures[i]
		for _, p := range f.Patches {
			cur, err := a.readCombatPatchByte(p.RVA, len(p.Patch))
			if err != nil {
				continue
			}
			if bytesEqual(cur, p.Patch) {
				_ = writeCodeMemory(a.hProcess, a.moduleBase+p.RVA, p.Orig)
			}
		}
	}
}

var combatPatchFeatures = []combatPatchFeature{
	{
		ID:    "infinite_dodges",
		Name:  "无限闪避",
		Group: "战斗",
		Patches: []combatPatch{
			{RVA: 0x26FC4DD, Orig: []byte{0x8D, 0x43, 0x01}, Patch: []byte{0x31, 0xC0, 0x90}},
		},
	},
	{
		ID:    "no_guard_break",
		Name:  "不被破防",
		Group: "战斗",
		Patches: []combatPatch{
			{RVA: 0x1F3C946, Orig: []byte{0xC5, 0xFA, 0x59, 0xC6}, Patch: []byte{0x0F, 0x57, 0xC0, 0x90}},
		},
	},
	{
		ID:    "auto_perfect_block",
		Name:  "自动精准格挡",
		Group: "战斗",
		Patches: []combatPatch{
			{RVA: 0x1F3D050, Orig: []byte{0x77, 0x16}, Patch: []byte{0x90, 0x90}},
			{RVA: 0x1F3D066, Orig: []byte{0x76}, Patch: []byte{0xEB}},
		},
	},
	{
		ID:    "instant_link_time",
		Name:  "瞬发连携时间",
		Group: "战斗",
		Patches: []combatPatch{
			{RVA: 0x4B7889, Orig: []byte{0x0F, 0x87, 0x7C, 0x00, 0x00, 0x00}, Patch: []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "infinite_link_time",
		Name:  "无限连携时间",
		Group: "战斗",
		Patches: []combatPatch{
			{RVA: 0x19ABDC, Orig: []byte{0xC5, 0xFA, 0x59, 0x05, 0xB4, 0x95, 0x30, 0x05}, Patch: []byte{0x0F, 0x57, 0xC0, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "infinite_consumables",
		Name:  "无限消耗品",
		Group: "战斗",
		Patches: []combatPatch{
			{RVA: 0x367188, Orig: []byte{0x41, 0xB8, 0xFF, 0xFF, 0xFF, 0xFF}, Patch: []byte{0x4D, 0x31, 0xC0, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "instant_part_breaks",
		Name:  "瞬间破坏部位",
		Group: "战斗",
		Patches: []combatPatch{
			{RVA: 0x293CF0A, Orig: []byte{0x0F, 0x82, 0x98, 0xFD, 0xFF, 0xFF}, Patch: []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90}},
			{RVA: 0x2C94DCE, Orig: []byte{0x0F, 0x4E, 0xC1}, Patch: []byte{0x31, 0xC0, 0x90}},
			{RVA: 0x2AE990F, Orig: []byte{0x7F, 0x02}, Patch: []byte{0x90, 0x90}},
		},
	},
	{
		ID:    "infinite_boss_break_duration",
		Name:  "无限BOSS破坏时间",
		Group: "战斗",
		Patches: []combatPatch{
			{RVA: 0x1FF4B3C, Orig: []byte{0xC5, 0xF2, 0x58, 0xCA}, Patch: []byte{0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "bypass_summons",
		Name:  "解除召唤石限制",
		Group: "战斗",
		Patches: []combatPatch{
			{RVA: 0x1FA83A3, Orig: []byte{0x8B, 0x84}, Patch: []byte{0x31, 0xC0}},
			{RVA: 0x65E432, Orig: []byte{0x01}, Patch: []byte{0x00}},
			{RVA: 0x1FA84D6, Orig: []byte{0x0F, 0x8C, 0x1D, 0xFF, 0xFF, 0xFF}, Patch: []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "bypass_skybound_art_gauge",
		Name:  "无视奥义槽",
		Group: "通用角色",
		Patches: []combatPatch{
			{RVA: 0x977D5D, Orig: []byte{0x73}, Patch: []byte{0xEB}},
			{RVA: 0x71AC32, Orig: []byte{0x0F, 0x82, 0xE8, 0xFC, 0xFF, 0xFF}, Patch: []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90}},
			{RVA: 0x71BE77, Orig: []byte{0x0F, 0x82, 0x2C, 0xF9, 0xFF, 0xFF}, Patch: []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "endless_hold_duration_skills",
		Name:  "长按技能无限持续",
		Group: "通用角色",
		Patches: []combatPatch{
			{RVA: 0x2803ED2, Orig: []byte{0xC5, 0xFA, 0x10, 0x40, 0x20}, Patch: []byte{0x0F, 0x57, 0xC0, 0x90, 0x90}},
		},
	},
	{
		ID:    "infinite_arts_level_duration",
		Name:  "奥义等级无限持续",
		Group: "古兰",
		Patches: []combatPatch{
			{RVA: 0x334C7C2, Orig: []byte{0xC5, 0xEA, 0x58, 0xC0}, Patch: []byte{0xF3, 0x0F, 0x10, 0xC2}},
		},
	},
	{
		ID:    "infinite_heat_gauge",
		Name:  "无限热量槽",
		Group: "拉卡姆",
		Patches: []combatPatch{
			{RVA: 0x32A5936, Orig: []byte{0xC5, 0xEA, 0x58, 0xC0}, Patch: []byte{0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "instant_max_mystic_vortex",
		Name:  "瞬间满秘术漩涡",
		Group: "伊欧",
		Patches: []combatPatch{
			{RVA: 0x33740BD, Orig: []byte{0x0F, 0x42, 0xC1}, Patch: []byte{0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "bypass_mystic_vortex_orbs",
		Name:  "无视秘术漩涡球",
		Group: "伊欧",
		Patches: []combatPatch{
			{RVA: 0x27A76DC, Orig: []byte{0x0F, 0x84, 0x48, 0x01, 0x00, 0x00}, Patch: []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "always_max_stargaze",
		Name:  "常驻满观星",
		Group: "伊欧",
		Patches: []combatPatch{
			{RVA: 0x27A7706, Orig: []byte{0x44, 0x0F, 0x4F, 0xC1}, Patch: []byte{0x44, 0x8B, 0xC1, 0x90}},
		},
	},
	{
		ID:    "instant_flowery_seven_charge",
		Name:  "瞬间充能花之七",
		Group: "伊欧",
		Patches: []combatPatch{
			{RVA: 0x2A745C6, Orig: []byte{0xC5, 0xFA, 0x10, 0x86, 0x8C, 0x00, 0x00, 0x00}, Patch: []byte{0x0F, 0x57, 0xC0, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "sniper_quick_fire_rate",
		Name:  "狙击快速射击",
		Group: "欧根",
		Patches: []combatPatch{
			{RVA: 0x2A63811, Orig: []byte{0xC5, 0xFA, 0x10, 0x80, 0x88, 0x2A, 0x00, 0x00}, Patch: []byte{0x0F, 0x57, 0xC0, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "instant_detonator",
		Name:  "瞬间引爆",
		Group: "欧根",
		Patches: []combatPatch{
			{RVA: 0x2A69F53, Orig: []byte{0xC5, 0xFA, 0x59, 0x05, 0x91, 0xA0, 0xF4, 0x01}, Patch: []byte{0x0F, 0x57, 0xC0, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "instant_dawnfly_charge_attack",
		Name:  "瞬间晓蝶蓄力攻击",
		Group: "娜露梅",
		Patches: []combatPatch{
			{RVA: 0x33FCC3C, Orig: []byte{0x77}, Patch: []byte{0xEB}},
		},
	},
	{
		ID:    "infinite_ghost_duration",
		Name:  "幽灵无限持续",
		Group: "菲莉",
		Patches: []combatPatch{
			{RVA: 0x339ECAD, Orig: []byte{0xC5, 0xF2, 0x58, 0xC0}, Patch: []byte{0xF3, 0x0F, 0x10, 0xC1}},
		},
	},
	{
		ID:    "infinite_sword_of_lumiel_duration",
		Name:  "光明之剑无限持续",
		Group: "夏洛特",
		Patches: []combatPatch{
			{RVA: 0x33D7164, Orig: []byte{0xC4, 0xC1, 0x72, 0x5C, 0xC8}, Patch: []byte{0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "instant_max_triple_shroud_marks",
		Name:  "瞬间满三重隐匿印",
		Group: "尤达拉哈",
		Patches: []combatPatch{
			{RVA: 0x33E34B1, Orig: []byte{0x0F, 0x42, 0xC1}, Patch: []byte{0x90, 0x90, 0x90}},
			{RVA: 0x33E361A, Orig: []byte{0x0F, 0x42, 0xC1}, Patch: []byte{0x90, 0x90, 0x90}},
			{RVA: 0x33E367C, Orig: []byte{0x0F, 0x42, 0xC1}, Patch: []byte{0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "instant_sharpened_focus",
		Name:  "瞬间凝神",
		Group: "尤达拉哈",
		Patches: []combatPatch{
			{RVA: 0x33E712A, Orig: []byte{0x0F, 0x42, 0xC1}, Patch: []byte{0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "max_empowered_skills",
		Name:  "满强化技能",
		Group: "尤达拉哈",
		Patches: []combatPatch{
			{RVA: 0x33E64A6, Orig: []byte{0x8B, 0x86, 0xBC, 0xCA, 0x01, 0x00}, Patch: []byte{0x31, 0xC0, 0xB0, 0x03, 0x90, 0x90}},
		},
	},
	{
		ID:    "prevent_gyrnoth_gauge_loss",
		Name:  "防止古洛诺斯槽流失",
		Group: "巴萨拉卡",
		Patches: []combatPatch{
			{RVA: 0x2A25B02, Orig: []byte{0xC5, 0xFA, 0x59, 0x0D, 0x8E, 0xE6, 0xA7, 0x02}, Patch: []byte{0x0F, 0x57, 0xC9, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "instant_max_grynoth_gauge_level",
		Name:  "瞬间满古洛诺斯槽等级",
		Group: "巴萨拉卡",
		Patches: []combatPatch{
			{RVA: 0x2036D1C, Orig: []byte{0x77, 0x71}, Patch: []byte{0x90, 0x90}},
			{RVA: 0x2036D47, Orig: []byte{0x0F, 0x4C, 0xD0}, Patch: []byte{0x8B, 0xD0, 0x90}},
		},
	},
	{
		ID:    "instant_arvess_fermare",
		Name:  "瞬间阿尔贝斯·停顿",
		Group: "塞达",
		Patches: []combatPatch{
			{RVA: 0x341E910, Orig: []byte{0xFF, 0xC0}, Patch: []byte{0xB1, 0x04}},
		},
	},
	{
		ID:    "force_arvess_hammer",
		Name:  "强制阿尔贝斯战锤",
		Group: "塞达",
		Patches: []combatPatch{
			{RVA: 0x341F5DC, Orig: []byte{0x7D}, Patch: []byte{0xEB}},
		},
	},
	{
		ID:    "instant_max_eternal_rage_stacks",
		Name:  "瞬间满永恒之怒层数",
		Group: "冈达葛萨",
		Patches: []combatPatch{
			{RVA: 0x340C006, Orig: []byte{0x03, 0x9E, 0x88, 0xCA, 0x01, 0x00}, Patch: []byte{0xB3, 0x0A, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "instant_rage_fist_charge",
		Name:  "瞬间怒拳充能",
		Group: "冈达葛萨",
		Patches: []combatPatch{
			{RVA: 0x3404C03, Orig: []byte{0xC5, 0xF8, 0x28, 0xF1}, Patch: []byte{0x0F, 0x57, 0xF6, 0x90}},
		},
	},
	{
		ID:    "no_eternal_rage_decay",
		Name:  "永恒之怒不衰减",
		Group: "冈达葛萨",
		Patches: []combatPatch{
			{RVA: 0x3406F1F, Orig: []byte{0xC5, 0xEA, 0x5C, 0xC0}, Patch: []byte{0xF3, 0x0F, 0x10, 0xC2}},
		},
	},
	{
		ID:    "instant_max_beatdown",
		Name:  "瞬间满连击",
		Group: "巴恩",
		Patches: []combatPatch{
			{RVA: 0x2036D1C, Orig: []byte{0x77, 0x71}, Patch: []byte{0x90, 0x90}},
		},
	},
	{
		ID:    "infinite_beatdown_stacks",
		Name:  "无限连击层数",
		Group: "巴恩",
		Patches: []combatPatch{
			{RVA: 0x2036BDD, Orig: []byte{0x8D, 0x4F, 0xFF}, Patch: []byte{0x8B, 0xCF, 0x90}},
		},
	},
	{
		ID:    "infinite_dragon_form",
		Name:  "龙化形态无限持续",
		Group: "伊德",
		Patches: []combatPatch{
			{RVA: 0xB0EE3F, Orig: []byte{0xC5, 0xEA, 0x58, 0xC0}, Patch: []byte{0xF3, 0x0F, 0x10, 0xC2}},
		},
	},
	{
		ID:    "instant_versalis_gauge",
		Name:  "瞬间斗气槽",
		Group: "伊德",
		Patches: []combatPatch{
			{RVA: 0xAFDC5B, Orig: []byte{0x5D}, Patch: []byte{0x5F}},
		},
	},
	{
		ID:    "instant_max_fourfold_vengance_charges",
		Name:  "瞬间满四重复仇充能",
		Group: "伊德",
		Patches: []combatPatch{
			{RVA: 0x279B23E, Orig: []byte{0x0F, 0x4C, 0xC1}, Patch: []byte{0x8B, 0xC1, 0x90}},
		},
	},
	{
		ID:    "bypass_foulfold_health_burn",
		Name:  "无视四重复仇扣血",
		Group: "伊德",
		Patches: []combatPatch{
			{RVA: 0xAEE7E4, Orig: []byte{0xC5, 0xF2, 0x58, 0xC0}, Patch: []byte{0xF3, 0x0F, 0x10, 0xC1}},
		},
	},
	{
		ID:    "infinite_godmight_duration",
		Name:  "神威无限持续",
		Group: "伊德",
		Patches: []combatPatch{
			{RVA: 0xAFD32A, Orig: []byte{0xC5, 0xEA, 0x58, 0xC0}, Patch: []byte{0xF3, 0x0F, 0x10, 0xC2}},
		},
	},
	{
		ID:    "instant_godmight",
		Name:  "瞬间神威",
		Group: "伊德",
		Patches: []combatPatch{
			{RVA: 0xB14BCF, Orig: []byte{0xC5, 0xF2, 0x5D, 0xC0}, Patch: []byte{0x0F, 0x57, 0xC0, 0x90}},
		},
	},
	{
		ID:    "infinite_chromatic_wing_form",
		Name:  "彩翼形态无限持续",
		Group: "圣德芬",
		Patches: []combatPatch{
			{RVA: 0x191EF6, Orig: []byte{0xC5, 0xFA, 0x59, 0x0D, 0x9A, 0x22, 0x31, 0x05}, Patch: []byte{0x0F, 0x57, 0xC9, 0x90, 0x90, 0x90, 0x90, 0x90}},
			{RVA: 0x2C2247A, Orig: []byte{0xC5, 0xF2, 0x59, 0xC8}, Patch: []byte{0x0F, 0x57, 0xC9, 0x90}},
		},
	},
	{
		ID:    "instant_max_avatar_swordshine",
		Name:  "瞬间满化身与剑光",
		Group: "希耶提",
		Patches: []combatPatch{
			{RVA: 0x2036D1C, Orig: []byte{0x77, 0x71}, Patch: []byte{0x90, 0x90}},
		},
	},
	{
		ID:    "bypass_ultrasight_arrows",
		Name:  "无视超视箭",
		Group: "索恩",
		Patches: []combatPatch{
			{RVA: 0x343E6D3, Orig: []byte{0xC5, 0xFA, 0x2C, 0xBE, 0x14, 0xCD, 0x01, 0x00}, Patch: []byte{0x31, 0xFF, 0x40, 0xB7, 0x0A, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "infinite_undying_blue",
		Name:  "不灭之蓝无限持续",
		Group: "贝阿朵丽丝",
		Patches: []combatPatch{
			{RVA: 0x3492999, Orig: []byte{0xC5, 0xFA, 0x59, 0x80, 0x9C, 0x01, 0x00, 0x00}, Patch: []byte{0x0F, 0x57, 0xC0, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "no_naed_nulli_damage",
		Name:  "免疫奈德·努利伤害",
		Group: "贝阿朵丽丝",
		Patches: []combatPatch{
			{RVA: 0x349374E, Orig: []byte{0xC5, 0xF2, 0x5C, 0xC0}, Patch: []byte{0xF3, 0x0F, 0x10, 0xC1}},
		},
	},
	{
		ID:    "no_reload",
		Name:  "无需装填",
		Group: "尤斯提斯",
		Patches: []combatPatch{
			{RVA: 0x34B541C, Orig: []byte{0xFF, 0xC9}, Patch: []byte{0x90, 0x90}},
		},
	},
	{
		ID:    "infinite_challenge_time",
		Name:  "无限挑战时间",
		Group: "任务",
		Patches: []combatPatch{
			{RVA: 0x32F05E1, Orig: []byte{0xC5, 0xFA, 0x58, 0xC2}, Patch: []byte{0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "instant_shrouded_treasure",
		Name:  "瞬开隐藏宝箱",
		Group: "任务",
		Patches: []combatPatch{
			{RVA: 0x32F5B8B, Orig: []byte{0x0F, 0x85, 0x1E, 0x01, 0x00, 0x00}, Patch: []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "freeze_quest_timer",
		Name:  "冻结任务计时",
		Group: "任务",
		Patches: []combatPatch{
			{RVA: 0x1FD9BE8, Orig: []byte{0xC5, 0xFA, 0x58, 0xC2}, Patch: []byte{0x90, 0x90, 0x90, 0x90}},
		},
	},
	{
		ID:    "auto_loot_quest_chest",
		Name:  "自动开箱",
		Group: "任务",
		Patches: []combatPatch{
			{RVA: 0x1FD0F3B, Orig: []byte{0x48, 0xB8}, Patch: []byte{0x31, 0xC0}},
		},
	},
	{
		ID:    "skip_result_screen",
		Name:  "跳过结算画面",
		Group: "任务",
		Patches: []combatPatch{
			{RVA: 0x2A023F9, Orig: []byte{0xC5, 0xEA, 0x5C, 0xC0}, Patch: []byte{0x0F, 0x57, 0xC0, 0x90}},
		},
	},
	{
		ID:    "100_terminus_weapon_drop_chance",
		Name:  "终焉武器100%掉率",
		Group: "任务",
		Patches: []combatPatch{
			{RVA: 0x372BC0, Orig: []byte{0x77, 0x25}, Patch: []byte{0x90, 0x90}},
		},
	},
	{
		ID:    "bypass_keys",
		Name:  "无视钥匙",
		Group: "生活品质",
		Patches: []combatPatch{
			{RVA: 0x32248C0, Orig: []byte{0x0F, 0x9F, 0xC0}, Patch: []byte{0xB0, 0x01, 0x90}},
		},
	},
}