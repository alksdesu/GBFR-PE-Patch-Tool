package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

const (
	badgeVectorSize = 1700
	badgeMaxID      = 1627
)

// badge.tbl in game 2.0.2 has no records for these IDs, including the orphaned message ID 1596.
var unavailableBadgeIDs = map[int]struct{}{
	690: {}, 691: {}, 692: {},
	1263: {}, 1264: {}, 1265: {},
	1272: {}, 1273: {}, 1274: {},
	1380: {}, 1381: {}, 1498: {}, 1596: {},
}

var badgeIDs = buildBadgeIDs()

type BadgeUnlockStatus struct {
	Total         int  `json:"total"`
	Unlocked      int  `json:"unlocked"`
	Viewed        int  `json:"viewed"`
	RewardClaimed int  `json:"rewardClaimed"`
	AllUnlocked   bool `json:"allUnlocked"`
	AllViewed     bool `json:"allViewed"`
}

type BadgeUnlockResult struct {
	Status        BadgeUnlockStatus `json:"status"`
	Changed       int               `json:"changed"`
	ViewedChanged int               `json:"viewedChanged"`
	BackupPath    string            `json:"backupPath"`
}

type badgeSaveContext struct {
	save     *SaveData
	unlocked *BoolSaveDataUnit
	viewed   *BoolSaveDataUnit
	reward   *BoolSaveDataUnit
	mode     os.FileMode
}

func buildBadgeIDs() []int {
	ids := make([]int, 0, badgeMaxID+1-len(unavailableBadgeIDs))
	for id := 0; id <= badgeMaxID; id++ {
		if _, unavailable := unavailableBadgeIDs[id]; !unavailable {
			ids = append(ids, id)
		}
	}
	return ids
}

func (a *App) GetBadgeUnlockStatus(path string) (BadgeUnlockStatus, error) {
	ctx, err := loadBadgeSave(path)
	if err != nil {
		return BadgeUnlockStatus{}, err
	}
	return ctx.status(), nil
}

func (a *App) UnlockAllBadges(path string, markViewed bool) (BadgeUnlockResult, error) {
	ctx, err := loadBadgeSave(path)
	if err != nil {
		return BadgeUnlockResult{}, err
	}

	original := append([]byte(nil), ctx.save.data...)
	originalReward := ctx.vectorBytes(ctx.reward)
	originalUnlocked := ctx.vectorBytes(ctx.unlocked)
	originalViewed := ctx.vectorBytes(ctx.viewed)

	result := BadgeUnlockResult{}
	for _, id := range badgeIDs {
		if ctx.setBool(ctx.unlocked, id) {
			result.Changed++
		}
		if markViewed && ctx.setBool(ctx.viewed, id) {
			result.ViewedChanged++
		}
	}

	if result.Changed == 0 && result.ViewedChanged == 0 {
		result.Status = ctx.status()
		return result, nil
	}
	if err := ctx.save.FixChecksums(); err != nil {
		return BadgeUnlockResult{}, fmt.Errorf("更新存档校验失败: %w", err)
	}
	if err := verifySaveChecksum(ctx.save); err != nil {
		return BadgeUnlockResult{}, err
	}
	if err := ensureFileUnchanged(path, original); err != nil {
		return BadgeUnlockResult{}, err
	}

	backupPath, err := createBadgeBackup(path, original, ctx.mode)
	if err != nil {
		return BadgeUnlockResult{}, err
	}
	result.BackupPath = backupPath

	if err := ensureFileUnchanged(path, original); err != nil {
		return BadgeUnlockResult{}, err
	}
	if err := replaceFileAtomically(path, ctx.save.data, ctx.mode); err != nil {
		return BadgeUnlockResult{}, fmt.Errorf("写入存档失败，原存档未改动，备份位于 %s: %w", backupPath, err)
	}

	verified, err := loadBadgeSave(path)
	if err == nil {
		err = verifyBadgeWrite(verified, markViewed, originalUnlocked, originalViewed, originalReward)
	}
	if err == nil {
		err = verifySaveChecksum(verified.save)
	}
	if err != nil {
		if restoreErr := replaceFileAtomically(path, original, ctx.mode); restoreErr != nil {
			return BadgeUnlockResult{}, fmt.Errorf("写后验证失败且自动恢复失败，请使用备份 %s: %v；恢复错误: %w", backupPath, err, restoreErr)
		}
		return BadgeUnlockResult{}, fmt.Errorf("写后验证失败，已自动恢复原存档，备份位于 %s: %w", backupPath, err)
	}

	result.Status = verified.status()
	return result, nil
}

func loadBadgeSave(path string) (*badgeSaveContext, error) {
	if !strings.EqualFold(filepath.Ext(path), ".dat") {
		return nil, fmt.Errorf("请选择 .dat 存档文件")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("读取存档信息失败: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("存档路径必须是普通文件")
	}

	save, err := LoadSave(path)
	if err != nil {
		return nil, err
	}
	parsed, err := ParseSaveData(save.data)
	if err != nil {
		return nil, fmt.Errorf("解析存档失败: %w", err)
	}
	if parsed.SlotData == nil {
		return nil, fmt.Errorf("存档缺少 SlotData")
	}

	ctx := &badgeSaveContext{save: save, mode: info.Mode().Perm()}
	if ctx.unlocked, err = uniqueBadgeUnit(parsed.SlotData.BoolTable, SaveID_BadgeUnlocked); err != nil {
		return nil, err
	}
	if ctx.viewed, err = uniqueBadgeUnit(parsed.SlotData.BoolTable, SaveID_BadgeViewed); err != nil {
		return nil, err
	}
	if ctx.reward, err = uniqueBadgeUnit(parsed.SlotData.BoolTable, SaveID_BadgeRewardClaimed); err != nil {
		return nil, err
	}
	for _, unit := range []*BoolSaveDataUnit{ctx.unlocked, ctx.viewed, ctx.reward} {
		if err := ctx.validateUnit(unit); err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

func uniqueBadgeUnit(table []BoolSaveDataUnit, idType uint32) (*BoolSaveDataUnit, error) {
	var found *BoolSaveDataUnit
	for i := range table {
		if table[i].IDType != idType {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("存档中存在重复的称号状态 %d，已拒绝写入", idType)
		}
		found = &table[i]
	}
	if found == nil {
		return nil, fmt.Errorf("存档缺少称号状态 %d，可能不是受支持的 2.0.2 存档", idType)
	}
	return found, nil
}

func (ctx *badgeSaveContext) validateUnit(unit *BoolSaveDataUnit) error {
	if unit.UnitID != 0 {
		return fmt.Errorf("称号状态 %d 的 UnitID 异常: %d", unit.IDType, unit.UnitID)
	}
	if len(unit.ValueData) != badgeVectorSize {
		return fmt.Errorf("称号状态 %d 长度异常: 期望 %d，实际 %d", unit.IDType, badgeVectorSize, len(unit.ValueData))
	}
	start, end, err := ctx.vectorRange(unit)
	if err != nil {
		return err
	}
	for i, value := range ctx.save.data[start:end] {
		if value > 1 {
			return fmt.Errorf("称号状态 %d 在索引 %d 存在非法布尔值 %d", unit.IDType, i, value)
		}
		if unit.ValueData[i] != (value == 1) {
			return fmt.Errorf("称号状态 %d 在索引 %d 解析结果不一致", unit.IDType, i)
		}
	}
	return nil
}

func (ctx *badgeSaveContext) vectorRange(unit *BoolSaveDataUnit) (int, int, error) {
	start := int(ctx.save.slotOff) + unit.valueOff
	end := start + len(unit.ValueData)
	slotEnd := int(ctx.save.slotOff + ctx.save.slotLen)
	if unit.valueOff <= 0 || start < int(ctx.save.slotOff) || end > slotEnd || end > len(ctx.save.data) {
		return 0, 0, fmt.Errorf("称号状态 %d 的数据偏移越界", unit.IDType)
	}
	return start, end, nil
}

func (ctx *badgeSaveContext) vectorBytes(unit *BoolSaveDataUnit) []byte {
	start, end, _ := ctx.vectorRange(unit)
	return append([]byte(nil), ctx.save.data[start:end]...)
}

func (ctx *badgeSaveContext) setBool(unit *BoolSaveDataUnit, index int) bool {
	start, _, _ := ctx.vectorRange(unit)
	if ctx.save.data[start+index] == 1 {
		return false
	}
	ctx.save.data[start+index] = 1
	unit.ValueData[index] = true
	return true
}

func (ctx *badgeSaveContext) status() BadgeUnlockStatus {
	status := BadgeUnlockStatus{Total: len(badgeIDs)}
	for _, id := range badgeIDs {
		if ctx.unlocked.ValueData[id] {
			status.Unlocked++
		}
		if ctx.viewed.ValueData[id] {
			status.Viewed++
		}
		if ctx.reward.ValueData[id] {
			status.RewardClaimed++
		}
	}
	status.AllUnlocked = status.Unlocked == status.Total
	status.AllViewed = status.Viewed == status.Total
	return status
}

func verifyBadgeWrite(ctx *badgeSaveContext, markViewed bool, originalUnlocked, originalViewed, originalReward []byte) error {
	unlocked := ctx.vectorBytes(ctx.unlocked)
	viewed := ctx.vectorBytes(ctx.viewed)
	reward := ctx.vectorBytes(ctx.reward)
	for _, id := range badgeIDs {
		if unlocked[id] != 1 {
			return fmt.Errorf("称号 %d 未成功解锁", id)
		}
		if markViewed && viewed[id] != 1 {
			return fmt.Errorf("称号 %d 未成功标记为已查看", id)
		}
	}
	if !bytes.Equal(reward, originalReward) {
		return fmt.Errorf("称号奖励领取状态发生了意外变化")
	}
	for id := 0; id < badgeVectorSize; id++ {
		_, unavailable := unavailableBadgeIDs[id]
		if unavailable || id > badgeMaxID {
			if unlocked[id] != originalUnlocked[id] {
				return fmt.Errorf("无效称号索引 %d 被意外修改", id)
			}
			if viewed[id] != originalViewed[id] {
				return fmt.Errorf("无效称号已查看索引 %d 被意外修改", id)
			}
		}
		if !markViewed && viewed[id] != originalViewed[id] {
			return fmt.Errorf("称号已查看状态被意外修改")
		}
	}
	return nil
}

func verifySaveChecksum(save *SaveData) error {
	slot := save.slotSpan()
	seed, ok := save.findUnit(SaveID_HashSeed, 0)
	if !ok {
		return fmt.Errorf("存档缺少哈希种子")
	}
	index := int(seed.Uint32() % uint32(len(hashSectionInfos)))
	if len(slot) < 0x14 {
		return fmt.Errorf("SlotData 长度不足")
	}
	hashesOff := int(binary.LittleEndian.Uint32(slot[len(slot)-0x14:]))
	if hashesOff+len(hashSectionInfos)*8 > len(slot) {
		return fmt.Errorf("存档哈希表偏移越界")
	}
	section := hashSectionInfos[index]
	hashLen := hashesOff - section.StartOffset - section.SubSize
	if hashLen <= 0 || section.StartOffset+hashLen > len(slot) {
		return fmt.Errorf("存档哈希区间无效")
	}
	expected := xxHash64(slot[section.StartOffset:section.StartOffset+hashLen], XXHash64SaveSeed)
	actual := binary.LittleEndian.Uint64(slot[hashesOff+index*8:])
	if actual != expected {
		return fmt.Errorf("存档校验不匹配")
	}
	return nil
}

func ensureFileUnchanged(path string, expected []byte) error {
	current, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("写入前重新读取存档失败: %w", err)
	}
	if !bytes.Equal(current, expected) {
		return fmt.Errorf("存档在操作期间被游戏或其他程序修改，已取消写入")
	}
	return nil
}

func createBadgeBackup(path string, data []byte, mode os.FileMode) (string, error) {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(filepath.Base(path), ext)
	timestamp := time.Now().Format("20060102_150405")
	for sequence := 0; sequence < 1000; sequence++ {
		suffix := ""
		if sequence > 0 {
			suffix = fmt.Sprintf("_%03d", sequence)
		}
		// _BackUp keeps generated backup files out of save-slot discovery.
		backupPath := filepath.Join(dir, fmt.Sprintf("%s_BackUp_AllTitles_%s%s%s", stem, timestamp, suffix, ext))
		file, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("创建存档备份失败: %w", err)
		}
		if _, err = file.Write(data); err == nil {
			err = file.Sync()
		}
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(backupPath)
			return "", fmt.Errorf("写入存档备份失败: %w", err)
		}
		written, err := os.ReadFile(backupPath)
		if err != nil || !bytes.Equal(written, data) {
			return "", fmt.Errorf("存档备份验证失败: %s", backupPath)
		}
		return backupPath, nil
	}
	return "", fmt.Errorf("无法生成不重复的存档备份文件名")
}

func replaceFileAtomically(path string, data []byte, mode os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".gbfr-save-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return windows.Rename(tempPath, path)
}
