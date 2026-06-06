package main

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed data/quest_names_i18n.csv
var questCSVData []byte

// ── Exported types for Wails binding ──

type SaveSummary struct {
	FilePath       string  `json:"filePath"`
	FileName       string  `json:"fileName"`
	Rupees         int32   `json:"rupees"`
	MasteryPoints  int32   `json:"masteryPoints"`
	Commendations  int32   `json:"commendations"`
	StageID        uint32  `json:"stageId"`
	PartyHealth    []int32 `json:"partyHealth"`
	FavoriteChara  uint32  `json:"favoriteChara"`
	ItemCount      int     `json:"itemCount"`
	WeaponCount    int     `json:"weaponCount"`
	GemCount       int     `json:"gemCount"`
	QuestClears    int     `json:"questClears"`
	QuestTotalClears uint32 `json:"questTotalClears"`
	Unlocks        int     `json:"unlocks"`
}

type QuestEntry struct {
	QuestID   uint32 `json:"questId"`
	QuestName string `json:"questName"`
	QuestNameCN string `json:"questNameCn"`
	Clears    uint32 `json:"clears"`
}

type SaveSlot struct {
	Index int    `json:"index"`
	Path  string `json:"path"`
	Name  string `json:"name"`
}

// ── Quest name mapping ──

var questNames map[int]string
var questNamesCN map[int]string

func init() {
	questNames = make(map[int]string)
	questNamesCN = make(map[int]string)
	r := csv.NewReader(strings.NewReader(string(questCSVData)))
	records, err := r.ReadAll()
	if err != nil {
		return
	}
	for _, row := range records[1:] { // skip header
		if len(row) >= 2 {
			if id, err := strconv.Atoi(row[0]); err == nil {
				questNames[id] = row[1]
				if len(row) >= 3 && row[2] != "" {
					questNamesCN[id] = row[2]
				}
			}
		}
	}
}

func questIDToName(stored uint32) string {
	hexStr := fmt.Sprintf("%06X", stored)
	qid, _ := strconv.Atoi(hexStr)
	if name, ok := questNames[qid]; ok {
		return name
	}
	return fmt.Sprintf("Unknown_%d", qid)
}

func questIDToNameCN(stored uint32) string {
	hexStr := fmt.Sprintf("%06X", stored)
	qid, _ := strconv.Atoi(hexStr)
	if name, ok := questNamesCN[qid]; ok {
		return name
	}
	return ""
}

func storedToQuestID(stored uint32) uint32 {
	hexStr := fmt.Sprintf("%06X", stored)
	qid, _ := strconv.Atoi(hexStr)
	return uint32(qid)
}

// ── App save methods (bound to Wails) ──

// FindSaveFiles scans the default GBFR save directory
func (a *App) FindSaveFiles() []SaveSlot {
	gbfrFolder := filepath.Join(os.Getenv("LOCALAPPDATA"), "GBFR", "Saved", "SaveGames")
	entries, err := os.ReadDir(gbfrFolder)
	if err != nil {
		return nil
	}

	var slots []SaveSlot
	idx := 1
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "SaveData") && strings.HasSuffix(name, ".dat") && !strings.Contains(name, "_BackUp") {
			slots = append(slots, SaveSlot{
				Index: idx,
				Path:  filepath.Join(gbfrFolder, name),
				Name:  name,
			})
			idx++
		}
	}
	return slots
}

// LoadSave loads and parses a save file, returning a summary
func (a *App) LoadSave(path string) (*SaveSummary, error) {
	save, err := LoadSaveFile(path)
	if err != nil {
		return nil, fmt.Errorf("解析存档失败: %w", err)
	}
	if save.SlotData == nil {
		return nil, fmt.Errorf("存档SlotData为空")
	}

	s := &SaveSummary{
		FilePath: path,
		FileName: filepath.Base(path),
	}

	// Rupees (int)
	if unit := save.SlotData.GetIntUnit(SaveID_Rupees); unit != nil && len(unit.ValueData) > 0 {
		s.Rupees = unit.ValueData[0]
	}
	// Mastery Points (int)
	if unit := save.SlotData.GetIntUnit(SaveID_MasteryPoints); unit != nil && len(unit.ValueData) > 0 {
		s.MasteryPoints = unit.ValueData[0]
	}
	// Commendations (int)
	if unit := save.SlotData.GetIntUnit(SaveID_Commendations); unit != nil && len(unit.ValueData) > 0 {
		s.Commendations = unit.ValueData[0]
	}
	// Stage ID (uint)
	if unit := save.SlotData.GetUIntUnit(SaveID_CurrentStageID); unit != nil && len(unit.ValueData) > 0 {
		s.StageID = unit.ValueData[0]
	}
	// Party Health (int)
	if unit := save.SlotData.GetIntUnit(SaveID_PartyHealth); unit != nil {
		s.PartyHealth = unit.ValueData
	}
	// Favorite Character (int)
	if unit := save.SlotData.GetIntUnit(SaveID_FavoriteChara); unit != nil && len(unit.ValueData) > 0 {
		s.FavoriteChara = uint32(unit.ValueData[0])
	}

	// Count items
	for _, u := range save.SlotData.UIntTable {
		switch u.IDType {
		case SaveID_ItemID:
			s.ItemCount += len(u.ValueData)
		case SaveID_WeaponID:
			s.WeaponCount += len(u.ValueData)
		case SaveID_GemID:
			s.GemCount += len(u.ValueData)
		}
	}

	// Quest stats
	qIDs := save.SlotData.GetUIntUnit(SaveID_QuestIDs)
	qCounts := save.SlotData.GetUIntUnit(SaveID_QuestCompleteCount)
	if qIDs != nil && qCounts != nil {
		for i := 0; i < len(qIDs.ValueData) && i < len(qCounts.ValueData); i++ {
			if qCounts.ValueData[i] > 0 {
				s.QuestClears++
				s.QuestTotalClears += qCounts.ValueData[i]
			}
		}
	}

	// Unlocks
	if unit := save.SlotData.GetBoolUnit(SaveID_IsUnlocked); unit != nil {
		for _, v := range unit.ValueData {
			if v {
				s.Unlocks++
			}
		}
	}

	return s, nil
}

// GetQuests returns the full quest list with names and clear counts
func (a *App) GetQuests(path string) ([]QuestEntry, error) {
	save, err := LoadSaveFile(path)
	if err != nil {
		return nil, err
	}
	if save.SlotData == nil {
		return nil, fmt.Errorf("存档SlotData为空")
	}

	qIDs := save.SlotData.GetUIntUnit(SaveID_QuestIDs)
	qCounts := save.SlotData.GetUIntUnit(SaveID_QuestCompleteCount)
	if qIDs == nil || qCounts == nil {
		return nil, nil
	}

	var quests []QuestEntry
	for i := 0; i < len(qIDs.ValueData); i++ {
		if qIDs.ValueData[i] == 0 {
			continue
		}
		count := uint32(0)
		if i < len(qCounts.ValueData) {
			count = qCounts.ValueData[i]
		}
		qid := storedToQuestID(qIDs.ValueData[i])
		name := questIDToName(qIDs.ValueData[i])
		nameCN := questIDToNameCN(qIDs.ValueData[i])
		quests = append(quests, QuestEntry{
			QuestID:     qid,
			QuestName:   name,
			QuestNameCN: nameCN,
			Clears:      count,
		})
	}
	return quests, nil
}
