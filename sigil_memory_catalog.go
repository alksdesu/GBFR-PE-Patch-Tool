package main

import "fmt"

const memoryCatalogMaxLevel = 15

var memoryAwakeningPrimaryTraits = map[string]string{
	"转世之觉醒": "转世的恩宠",
	"刃姬之觉醒": "刃姬的小夜曲",
	"狼王之觉醒": "狼王的激昂",
	"黑龙之觉醒": "黑龙的咒印",
	"雷狼之觉醒": "雷狼的弹匣",
	"群青之觉醒": "群青的剑光",
}

// memoryCatalogDefs adapts verified memory-editor hashes for legacy generators.
// IDs are local catalog keys only; save writes use Hash.
func memoryCatalogDefs() ([]SigilDef, []TraitDef) {
	traitsByName := make(map[string]uint32, len(sigilMemoryTraits))
	for _, trait := range sigilMemoryTraits {
		traitsByName[trait.Name] = trait.Hash
	}

	sigils := make([]SigilDef, 0, len(sigilMemorySigils))
	traits := make([]TraitDef, 0, len(sigilMemoryTraits))
	traitSeen := make(map[uint32]bool, len(sigilMemoryTraits))
	canAppear := true
	for _, sigil := range sigilMemorySigils {
		traitHash, ok := traitsByName[sigil.Name]
		traitName := sigil.Name
		if !ok {
			traitName, ok = memoryAwakeningPrimaryTraits[sigil.Name]
			if ok {
				traitHash, ok = traitsByName[traitName]
			}
		}
		if !ok {
			continue
		}
		traitID := fmt.Sprintf("MEMORY_TRAIT_%08X", traitHash)
		traitMaxLevel := memoryCatalogMaxLevel
		if traitName == "相扑斗力" {
			traitMaxLevel = 5
		} else if traitName == "可怕的漆黑钳蟹因子" {
			traitMaxLevel = 20
		}
		if !traitSeen[traitHash] {
			traits = append(traits, TraitDef{
				InternalID:           traitID,
				Hash:                 fmt.Sprintf("0x%08X", traitHash),
				DisplayName:          traitName,
				MaxLevel:             &traitMaxLevel,
				CanAppearAsPrimary:   &canAppear,
				CanAppearAsSecondary: &canAppear,
			})
			traitSeen[traitHash] = true
		}
		sigilMaxLevel := memoryCatalogMaxLevel
		if sigil.Name == "可怕的漆黑钳蟹因子" || sigil.Name == "相扑斗力" {
			sigilMaxLevel = 0
		}
		sigils = append(sigils, SigilDef{
			InternalID:               fmt.Sprintf("MEMORY_SIGIL_%08X", sigil.Hash),
			Hash:                     fmt.Sprintf("0x%08X", sigil.Hash),
			DisplayName:              sigil.Name,
			SupportsSecondaryTrait:   boolPtr(!singleTraitMemorySigil(sigil.Name)),
			OptionalSecondaryTrait:   boolPtr(false),
			AllowedSecondaryTraitIDs: memoryTraitIDs(traitsByName),
			AllowedSigilLevels:       []int{sigilMaxLevel},
			DefaultSigilLevel:        &sigilMaxLevel,
			MaxSigilLevel:            &sigilMaxLevel,
			PrimaryTraitID:           traitID,
			PrimaryTraitName:         stringPtr(traitName),
			FirstTraitMaxLevel:       &traitMaxLevel,
			AllowedFirstTraitLevels:  levelsUpTo(traitMaxLevel),
		})
	}
	return sigils, traits
}

func boolPtr(value bool) *bool { return &value }

func singleTraitMemorySigil(name string) bool {
	return name == "相扑斗力" || name == "漆黑之谊" || name == "可怕的漆黑钳蟹因子"
}

func stringPtr(value string) *string { return &value }

func levelsUpTo(max int) []int {
	levels := make([]int, max)
	for i := range levels {
		levels[i] = i + 1
	}
	return levels
}

func memoryTraitIDs(traitsByName map[string]uint32) []string {
	ids := make([]string, 0, len(traitsByName))
	for _, hash := range traitsByName {
		ids = append(ids, fmt.Sprintf("MEMORY_TRAIT_%08X", hash))
	}
	return ids
}

func memoryWrightstoneTraits() []WrightstoneTraitDef {
	_, traits := memoryCatalogDefs()
	result := make([]WrightstoneTraitDef, 0, len(traits))
	for _, trait := range traits {
		result = append(result, WrightstoneTraitDef{
			InternalID:  trait.InternalID,
			Hash:        trait.Hash,
			DisplayName: trait.DisplayName,
			MaxLevel:    trait.MaxLevel,
		})
	}
	return result
}
