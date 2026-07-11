package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

const (
	summonInventoryPtrRVA = uintptr(0x7C23F48)
	summonRecordsOffset   = uintptr(0xC40)
	summonEntryStride     = uintptr(0x1C)
	summonEntryCount      = 1000
	summonSaveRVA         = uintptr(0x79D820)

	summonOffType      = uintptr(0x00)
	summonOffPrimary   = uintptr(0x08)
	summonOffSecondary = uintptr(0x0C)
	summonOffPrimaryLv = uintptr(0x10)
	summonOffSubParam  = uintptr(0x14)
	summonOffRank      = uintptr(0x18)
)

type SummonOption struct {
	Hash        uint32 `json:"hash"`
	DisplayName string `json:"displayName"`
}

type SummonSubOption struct {
	Hash        uint32    `json:"hash"`
	DisplayName string    `json:"displayName"`
	Values      []float64 `json:"values"`
	MaxLevel    int       `json:"maxLevel"`
	IsPercent   bool      `json:"isPercent"`
}

type SummonMemoryOptions struct {
	Summons   []SummonOption    `json:"summons"`
	Traits    []SummonOption    `json:"traits"`
	SubTraits []SummonSubOption `json:"subTraits"`
	MaxTrait  int               `json:"maxTraitLevel"`
}

type SummonEntry struct {
	Index            int    `json:"index"`
	Address          uint64 `json:"address"`
	TypeHash         uint32 `json:"typeHash"`
	TypeName         string `json:"typeName"`
	PrimaryHash      uint32 `json:"primaryHash"`
	PrimaryName      string `json:"primaryName"`
	PrimaryLevel     uint32 `json:"primaryLevel"`
	SecondaryHash    uint32 `json:"secondaryHash"`
	SecondaryName    string `json:"secondaryName"`
	SecondaryParam   uint32 `json:"secondaryParam"`
	Rank             uint32 `json:"rank"`
}

type SummonUpdate struct {
	Index          int    `json:"index"`
	PrimaryHash    uint32 `json:"primaryHash"`
	PrimaryLevel   uint32 `json:"primaryLevel"`
	SecondaryHash  uint32 `json:"secondaryHash"`
	SecondaryParam uint32 `json:"secondaryParam"`
	Rank           uint32 `json:"rank"`
}

func (a *App) SummonMemoryGetOptions() (SummonMemoryOptions, error) {
	catalog, err := LoadSummonCatalog()
	if err != nil {
		return SummonMemoryOptions{}, err
	}
	result := SummonMemoryOptions{
		Summons:   make([]SummonOption, 0, len(catalog.Summons)),
		Traits:    make([]SummonOption, 0, len(catalog.Traits)),
		SubTraits: make([]SummonSubOption, 0, len(catalog.SubTraits)),
		MaxTrait:  summonTraitMaxLevel,
	}
	for i := range catalog.Summons {
		hash, err := ParseHashHex(catalog.Summons[i].Hash)
		if err != nil {
			continue
		}
		result.Summons = append(result.Summons, SummonOption{Hash: hash, DisplayName: catalog.Summons[i].DisplayName})
	}
	for i := range catalog.Traits {
		hash, err := ParseHashHex(catalog.Traits[i].Hash)
		if err != nil {
			continue
		}
		result.Traits = append(result.Traits, SummonOption{Hash: hash, DisplayName: catalog.Traits[i].DisplayName})
	}
	for i := range catalog.SubTraits {
		sub := &catalog.SubTraits[i]
		hash, err := ParseHashHex(sub.Hash)
		if err != nil {
			continue
		}
		result.SubTraits = append(result.SubTraits, SummonSubOption{
			Hash:        hash,
			DisplayName: sub.DisplayName,
			Values:      sub.Values,
			MaxLevel:    sub.MaxLevel,
			IsPercent:   sub.IsPercent,
		})
	}
	return result, nil
}

func (a *App) summonInventoryBase() (uintptr, error) {
	var inventory uintptr
	root := a.moduleBase + summonInventoryPtrRVA
	if err := readProcessMemory(a.hProcess, root, unsafe.Pointer(&inventory), unsafe.Sizeof(inventory)); err != nil {
		return 0, fmt.Errorf("读取召唤石背包指针失败: %w", err)
	}
	if inventory == 0 {
		return 0, fmt.Errorf("召唤石背包未加载，请进入游戏存档并打开召唤石界面")
	}
	return inventory + summonRecordsOffset, nil
}

func (a *App) summonEntryAddr(records uintptr, index int) uintptr {
	return records + uintptr(index)*summonEntryStride
}

func (a *App) decodeSummonEntry(catalog *SummonCatalog, records uintptr, index int, rec []byte) (SummonEntry, bool) {
	typeHash := binary.LittleEndian.Uint32(rec[summonOffType : summonOffType+4])
	if typeHash == EmptyHash || typeHash == 0 {
		return SummonEntry{}, false
	}
	entry := SummonEntry{
		Index:          index,
		Address:        uint64(a.summonEntryAddr(records, index)),
		TypeHash:       typeHash,
		PrimaryHash:    binary.LittleEndian.Uint32(rec[summonOffPrimary : summonOffPrimary+4]),
		SecondaryHash:  binary.LittleEndian.Uint32(rec[summonOffSecondary : summonOffSecondary+4]),
		PrimaryLevel:   binary.LittleEndian.Uint32(rec[summonOffPrimaryLv : summonOffPrimaryLv+4]),
		SecondaryParam: binary.LittleEndian.Uint32(rec[summonOffSubParam : summonOffSubParam+4]),
		Rank:           binary.LittleEndian.Uint32(rec[summonOffRank : summonOffRank+4]),
	}
	if summon := catalog.LookupSummonByHash(entry.TypeHash); summon != nil {
		entry.TypeName = summon.DisplayName
	} else {
		entry.TypeName = fmt.Sprintf("0x%08X", entry.TypeHash)
	}
	if trait := catalog.LookupTraitByHash(entry.PrimaryHash); trait != nil {
		entry.PrimaryName = trait.DisplayName
	}
	if sub := catalog.LookupSubTraitByHash(entry.SecondaryHash); sub != nil {
		entry.SecondaryName = sub.DisplayName
	}
	return entry, true
}

func (a *App) readSummonEntry(catalog *SummonCatalog, records uintptr, index int) (SummonEntry, bool, error) {
	rec := make([]byte, summonEntryStride)
	if err := readProcessMemory(a.hProcess, a.summonEntryAddr(records, index), unsafe.Pointer(&rec[0]), summonEntryStride); err != nil {
		return SummonEntry{}, false, err
	}
	entry, ok := a.decodeSummonEntry(catalog, records, index, rec)
	return entry, ok, nil
}

func (a *App) SummonMemoryList() ([]SummonEntry, error) {
	if err := a.ensureGameProcess(); err != nil {
		return nil, err
	}
	catalog, err := LoadSummonCatalog()
	if err != nil {
		return nil, err
	}
	records, err := a.summonInventoryBase()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, summonEntryCount*int(summonEntryStride))
	if err := readProcessMemory(a.hProcess, records, unsafe.Pointer(&buf[0]), uintptr(len(buf))); err != nil {
		return nil, fmt.Errorf("读取召唤石列表失败: %w", err)
	}
	entries := make([]SummonEntry, 0, 32)
	for i := 0; i < summonEntryCount; i++ {
		off := i * int(summonEntryStride)
		if entry, ok := a.decodeSummonEntry(catalog, records, i, buf[off:off+int(summonEntryStride)]); ok {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("未读取到已解锁的召唤石；请在游戏内进入召唤石界面后重试")
	}
	return entries, nil
}

func (a *App) SummonMemoryUpdate(update SummonUpdate) (SummonEntry, error) {
	if err := a.ensureGameProcess(); err != nil {
		return SummonEntry{}, err
	}
	catalog, err := LoadSummonCatalog()
	if err != nil {
		return SummonEntry{}, err
	}
	if update.Index < 0 || update.Index >= summonEntryCount {
		return SummonEntry{}, fmt.Errorf("召唤石序号超出范围")
	}
	records, err := a.summonInventoryBase()
	if err != nil {
		return SummonEntry{}, err
	}
	_, ok, err := a.readSummonEntry(catalog, records, update.Index)
	if err != nil {
		return SummonEntry{}, err
	}
	if !ok {
		return SummonEntry{}, fmt.Errorf("该槽位召唤石为空")
	}

	if update.PrimaryLevel < 1 {
		update.PrimaryLevel = 1
	}
	if update.PrimaryLevel > summonTraitMaxLevel {
		update.PrimaryLevel = summonTraitMaxLevel
	}
	if update.Rank < 1 {
		update.Rank = 1
	}
	if update.Rank > 3 {
		update.Rank = 3
	}
	if sub := catalog.LookupSubTraitByHash(update.SecondaryHash); sub != nil {
		if update.SecondaryParam > uint32(sub.MaxLevel) {
			update.SecondaryParam = uint32(sub.MaxLevel)
		}
	}

	base := a.summonEntryAddr(records, update.Index)
	writes := []struct {
		offset uintptr
		value  uint32
		name   string
	}{
		{summonOffPrimary, update.PrimaryHash, "主因子"},
		{summonOffSecondary, update.SecondaryHash, "副特性"},
		{summonOffPrimaryLv, update.PrimaryLevel, "主因子等级"},
		{summonOffSubParam, update.SecondaryParam, "副特性参数"},
		{summonOffRank, update.Rank, "阶级"},
	}
	for _, w := range writes {
		if err := writeUint32Remote(a.hProcess, base+w.offset, w.value); err != nil {
			return SummonEntry{}, fmt.Errorf("写入%s失败: %w", w.name, err)
		}
	}
	if err := a.saveSummonEntry(base); err != nil {
		return SummonEntry{}, err
	}
	entry, _, err := a.readSummonEntry(catalog, records, update.Index)
	if err != nil {
		return SummonEntry{}, err
	}
	return entry, nil
}

func (a *App) saveSummonEntry(base uintptr) error {
	fn := a.moduleBase + summonSaveRVA
	for _, offset := range []uintptr{summonOffType, summonOffPrimary, summonOffSecondary, summonOffPrimaryLv, summonOffSubParam, summonOffRank} {
		if err := a.callRemoteOneArg(fn, base+offset); err != nil {
			return fmt.Errorf("保存召唤石字段 +0x%02X 失败: %w", offset, err)
		}
	}
	return nil
}
