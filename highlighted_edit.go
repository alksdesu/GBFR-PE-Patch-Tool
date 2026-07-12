package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

const (
	highlightedItemCaveID   = "highlighted_item"
	highlightedWeaponCaveID = "highlighted_weapon"
	highlightedItemPtrSym   = "NBGFR078_ptr"
	highlightedWeaponPtrSym = "NBGFR079_ptr"

	hlItemOffID     = uintptr(0x00)
	hlItemOffAmount = uintptr(0x04)
	hlItemOffState  = uintptr(0x08)

	hlWeaponOffSkin      = uintptr(0x08)
	hlWeaponOffID        = uintptr(0x04)
	hlWeaponOffExp       = uintptr(0x10)
	hlWeaponOffUncap     = uintptr(0x14)
	hlWeaponOffMirage    = uintptr(0x18)
	hlWeaponOffAwakened  = uintptr(0x1C)
	hlWeaponOffLevel     = uintptr(0x58)
	hlWeaponOffHP        = uintptr(0x5C)
	hlWeaponOffAttack    = uintptr(0x60)
	hlWeaponOffStun      = uintptr(0x64)
	hlWeaponOffCrit      = uintptr(0x68)
	hlWeaponOffImbuedPtr = uintptr(0x00)
	hlWeaponImbuedStone  = uintptr(0x38)
	hlWeaponTraitLvlOff  = uintptr(0x04)

	hlWeaponSaveLimit = uintptr(204)
	hlWeaponSaveStep  = uintptr(4)
)

var hlWeaponTraitSlots = []uintptr{0xA4, 0xAC, 0xB4, 0xBC, 0xC4}
var hlWeaponImbuedSlots = []uintptr{0x20, 0x28, 0x30}

type HighlightedItem struct {
	Captured bool   `json:"captured"`
	Address  uint64 `json:"address"`
	ID       uint32 `json:"id"`
	Amount   uint32 `json:"amount"`
	State    uint32 `json:"state"`
}

type HighlightedWeaponTrait struct {
	ID    uint32 `json:"id"`
	Level uint32 `json:"level"`
}

type HighlightedWeapon struct {
	Captured      bool                     `json:"captured"`
	Address       uint64                   `json:"address"`
	ID            uint32                   `json:"id"`
	Skin          uint32                   `json:"skin"`
	Level         uint32                   `json:"level"`
	HP            uint32                   `json:"hp"`
	Attack        uint32                   `json:"attack"`
	StunPower     uint32                   `json:"stunPower"`
	CritChance    uint32                   `json:"critChance"`
	Exp           uint32                   `json:"exp"`
	UncapLevel    uint32                   `json:"uncapLevel"`
	Mirage        uint32                   `json:"mirage"`
	AwakenedLevel uint32                   `json:"awakenedLevel"`
	Traits        []HighlightedWeaponTrait `json:"traits"`
	ImbuedStone   uint32                   `json:"imbuedStone"`
	HasImbued     bool                     `json:"hasImbued"`
	ImbuedTraits  []HighlightedWeaponTrait `json:"imbuedTraits"`
}

func (a *App) capturedPointer(caveID, ptrSym string) (uintptr, error) {
	ptr, err := a.CaveReadPointer(caveID, ptrSym)
	if err != nil {
		return 0, err
	}
	if ptr == 0 {
		return 0, fmt.Errorf("尚未捕获对象，请先在游戏内选中目标")
	}
	return uintptr(ptr), nil
}

func (a *App) readRemoteUint32(addr uintptr) (uint32, error) {
	buf := make([]byte, 4)
	if err := readProcessMemory(a.hProcess, addr, unsafe.Pointer(&buf[0]), 4); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}

func (a *App) readRemoteUint64(addr uintptr) (uint64, error) {
	buf := make([]byte, 8)
	if err := readProcessMemory(a.hProcess, addr, unsafe.Pointer(&buf[0]), 8); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func (a *App) HighlightedItemRead() (HighlightedItem, error) {
	if err := a.ensureGameProcess(); err != nil {
		return HighlightedItem{}, err
	}
	base, err := a.capturedPointer(highlightedItemCaveID, highlightedItemPtrSym)
	if err != nil {
		return HighlightedItem{}, err
	}
	result := HighlightedItem{Captured: true, Address: uint64(base)}
	if result.ID, err = a.readRemoteUint32(base + hlItemOffID); err != nil {
		return HighlightedItem{}, err
	}
	if result.Amount, err = a.readRemoteUint32(base + hlItemOffAmount); err != nil {
		return HighlightedItem{}, err
	}
	if result.State, err = a.readRemoteUint32(base + hlItemOffState); err != nil {
		return HighlightedItem{}, err
	}
	return result, nil
}

func (a *App) HighlightedItemUpdate(amount, state uint32) (HighlightedItem, error) {
	if err := a.ensureGameProcess(); err != nil {
		return HighlightedItem{}, err
	}
	base, err := a.capturedPointer(highlightedItemCaveID, highlightedItemPtrSym)
	if err != nil {
		return HighlightedItem{}, err
	}
	if err := writeUint32Remote(a.hProcess, base+hlItemOffAmount, amount); err != nil {
		return HighlightedItem{}, fmt.Errorf("写入数量失败: %w", err)
	}
	if err := writeUint32Remote(a.hProcess, base+hlItemOffState, state); err != nil {
		return HighlightedItem{}, fmt.Errorf("写入状态失败: %w", err)
	}
	return a.HighlightedItemRead()
}

func (a *App) HighlightedWeaponRead() (HighlightedWeapon, error) {
	if err := a.ensureGameProcess(); err != nil {
		return HighlightedWeapon{}, err
	}
	base, err := a.capturedPointer(highlightedWeaponCaveID, highlightedWeaponPtrSym)
	if err != nil {
		return HighlightedWeapon{}, err
	}
	w := HighlightedWeapon{Captured: true, Address: uint64(base)}
	scalars := []struct {
		off uintptr
		dst *uint32
	}{
		{hlWeaponOffID, &w.ID},
		{hlWeaponOffSkin, &w.Skin},
		{hlWeaponOffLevel, &w.Level},
		{hlWeaponOffHP, &w.HP},
		{hlWeaponOffAttack, &w.Attack},
		{hlWeaponOffStun, &w.StunPower},
		{hlWeaponOffCrit, &w.CritChance},
		{hlWeaponOffExp, &w.Exp},
		{hlWeaponOffUncap, &w.UncapLevel},
		{hlWeaponOffMirage, &w.Mirage},
		{hlWeaponOffAwakened, &w.AwakenedLevel},
	}
	for _, s := range scalars {
		v, err := a.readRemoteUint32(base + s.off)
		if err != nil {
			return HighlightedWeapon{}, err
		}
		*s.dst = v
	}
	for _, slot := range hlWeaponTraitSlots {
		id, err := a.readRemoteUint32(base + slot)
		if err != nil {
			return HighlightedWeapon{}, err
		}
		lvl, err := a.readRemoteUint32(base + slot + hlWeaponTraitLvlOff)
		if err != nil {
			return HighlightedWeapon{}, err
		}
		w.Traits = append(w.Traits, HighlightedWeaponTrait{ID: id, Level: lvl})
	}
	imbued, err := a.readRemoteUint64(base + hlWeaponOffImbuedPtr)
	if err != nil {
		return HighlightedWeapon{}, err
	}
	if imbued != 0 {
		w.HasImbued = true
		ibase := uintptr(imbued)
		if w.ImbuedStone, err = a.readRemoteUint32(ibase + hlWeaponImbuedStone); err != nil {
			return HighlightedWeapon{}, err
		}
		for _, slot := range hlWeaponImbuedSlots {
			id, err := a.readRemoteUint32(ibase + slot)
			if err != nil {
				return HighlightedWeapon{}, err
			}
			lvl, err := a.readRemoteUint32(ibase + slot + hlWeaponTraitLvlOff)
			if err != nil {
				return HighlightedWeapon{}, err
			}
			w.ImbuedTraits = append(w.ImbuedTraits, HighlightedWeaponTrait{ID: id, Level: lvl})
		}
	}
	return w, nil
}

type HighlightedWeaponUpdate struct {
	Skin          uint32                   `json:"skin"`
	Level         uint32                   `json:"level"`
	HP            uint32                   `json:"hp"`
	Attack        uint32                   `json:"attack"`
	StunPower     uint32                   `json:"stunPower"`
	CritChance    uint32                   `json:"critChance"`
	Exp           uint32                   `json:"exp"`
	UncapLevel    uint32                   `json:"uncapLevel"`
	Mirage        uint32                   `json:"mirage"`
	AwakenedLevel uint32                   `json:"awakenedLevel"`
	Traits        []HighlightedWeaponTrait `json:"traits"`
	ImbuedStone   uint32                   `json:"imbuedStone"`
	ImbuedTraits  []HighlightedWeaponTrait `json:"imbuedTraits"`
}

func (a *App) HighlightedWeaponUpdate(update HighlightedWeaponUpdate) (HighlightedWeapon, error) {
	if err := a.ensureGameProcess(); err != nil {
		return HighlightedWeapon{}, err
	}
	base, err := a.capturedPointer(highlightedWeaponCaveID, highlightedWeaponPtrSym)
	if err != nil {
		return HighlightedWeapon{}, err
	}
	writes := []struct {
		off   uintptr
		value uint32
		name  string
	}{
		{hlWeaponOffSkin, update.Skin, "皮肤"},
		{hlWeaponOffLevel, update.Level, "等级"},
		{hlWeaponOffHP, update.HP, "HP"},
		{hlWeaponOffAttack, update.Attack, "攻击"},
		{hlWeaponOffStun, update.StunPower, "眩晕值"},
		{hlWeaponOffCrit, update.CritChance, "暴击率"},
		{hlWeaponOffExp, update.Exp, "经验"},
		{hlWeaponOffUncap, update.UncapLevel, "突破等级"},
		{hlWeaponOffMirage, update.Mirage, "幻影弹"},
		{hlWeaponOffAwakened, update.AwakenedLevel, "觉醒等级"},
	}
	for _, w := range writes {
		if err := writeUint32Remote(a.hProcess, base+w.off, w.value); err != nil {
			return HighlightedWeapon{}, fmt.Errorf("写入%s失败: %w", w.name, err)
		}
	}
	for i, slot := range hlWeaponTraitSlots {
		if i >= len(update.Traits) {
			break
		}
		t := update.Traits[i]
		if err := writeUint32Remote(a.hProcess, base+slot, t.ID); err != nil {
			return HighlightedWeapon{}, fmt.Errorf("写入词条%d ID失败: %w", i+1, err)
		}
		if err := writeUint32Remote(a.hProcess, base+slot+hlWeaponTraitLvlOff, t.Level); err != nil {
			return HighlightedWeapon{}, fmt.Errorf("写入词条%d 等级失败: %w", i+1, err)
		}
	}
	imbued, err := a.readRemoteUint64(base + hlWeaponOffImbuedPtr)
	if err != nil {
		return HighlightedWeapon{}, err
	}
	if imbued != 0 {
		ibase := uintptr(imbued)
		if err := writeUint32Remote(a.hProcess, ibase+hlWeaponImbuedStone, update.ImbuedStone); err != nil {
			return HighlightedWeapon{}, fmt.Errorf("写入附魔祝福石失败: %w", err)
		}
		for i, slot := range hlWeaponImbuedSlots {
			if i >= len(update.ImbuedTraits) {
				break
			}
			t := update.ImbuedTraits[i]
			if err := writeUint32Remote(a.hProcess, ibase+slot, t.ID); err != nil {
				return HighlightedWeapon{}, fmt.Errorf("写入附魔词条%d ID失败: %w", i+1, err)
			}
			if err := writeUint32Remote(a.hProcess, ibase+slot+hlWeaponTraitLvlOff, t.Level); err != nil {
				return HighlightedWeapon{}, fmt.Errorf("写入附魔词条%d 等级失败: %w", i+1, err)
			}
		}
	}
	if err := a.saveWeaponEntry(base); err != nil {
		return HighlightedWeapon{}, err
	}
	return a.HighlightedWeaponRead()
}

func (a *App) saveWeaponEntry(base uintptr) error {
	fn := a.moduleBase + sigilMemorySaveRVA
	for off := uintptr(0); off <= hlWeaponSaveLimit; off += hlWeaponSaveStep {
		if err := a.callRemoteOneArg(fn, base+off); err != nil {
			return fmt.Errorf("保存武器字段 +0x%02X 失败: %w", off, err)
		}
	}
	return nil
}
