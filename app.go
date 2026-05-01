package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	steamAppID  = "881020"
	gameExeName = "granblue_fantasy_relink.exe"
	gameFolder  = "Granblue Fantasy Relink"
)

// ── 补丁定义 ──

// PatchDef 描述一个补丁点
type PatchDef struct {
	ID          string // 唯一标识
	Name        string // 显示名称
	RVA         uint32 // 补丁目标 RVA
	OrigBytes   []byte // 原始字节（用于校验和恢复）
	PatchSize   int    // 补丁覆盖的字节数
	NeedCave    bool   // 是否需要代码跳板
	CallTarget  uint32 // 跳板中 call 的目标 RVA（仅 NeedCave 时使用）
}

var patchDefs = []PatchDef{
	{
		ID:        "mission",
		Name:      "挑战次数",
		RVA:       0x003583FF,
		OrigBytes: []byte{0xB8, 0x3F, 0x42, 0x0F, 0x00, 0x41, 0x0F, 0x42, 0xC0},
		PatchSize: 9,
		NeedCave:  false,
	},
	{
		ID:        "likes",
		Name:      "点赞数值",
		RVA:       0x00A919CF,
		OrigBytes: []byte{0xB8, 0x3F, 0x42, 0x0F, 0x00, 0x0F, 0x42, 0xC6},
		PatchSize: 8,
		NeedCave:  false,
	},
}

// ── 状态结构 ──

type PatchStatus struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	State        string `json:"state"` // "original" | "patched" | "unknown"
	CurrentValue uint32 `json:"currentValue"`
	CurrentBytes string `json:"currentBytes"`
}

type StatusInfo struct {
	ExePath      string        `json:"exePath"`
	FileExists   bool          `json:"fileExists"`
	FileSize     int64         `json:"fileSize"`
	BackupExists bool          `json:"backupExists"`
	BackupSize   int64         `json:"backupSize"`
	Patches      []PatchStatus `json:"patches"`
}

// ── App ──

type App struct {
	ctx     context.Context
	exePath string
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// AutoDetect 自动扫描 Steam 安装路径
func (a *App) AutoDetect() string {
	for _, dir := range findSteamLibraryFolders() {
		candidate := filepath.Join(dir, "steamapps", "common", gameFolder, gameExeName)
		if _, err := os.Stat(candidate); err == nil {
			a.exePath = candidate
			return candidate
		}
	}
	return ""
}

// SetExePath 手动设置 exe 路径
func (a *App) SetExePath(path string) (StatusInfo, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return StatusInfo{}, fmt.Errorf("文件不存在: %s", path)
	}
	a.exePath = path
	return a.GetStatus(""), nil
}

// GetStatus 获取所有补丁点的状态
func (a *App) GetStatus(exePath string) StatusInfo {
	if exePath != "" {
		a.exePath = exePath
	}
	info := StatusInfo{ExePath: a.exePath}
	if a.exePath == "" {
		return info
	}

	bakPath := a.exePath + ".bak"
	if fi, err := os.Stat(a.exePath); err == nil {
		info.FileExists = true
		info.FileSize = fi.Size()
	}
	if fi, err := os.Stat(bakPath); err == nil {
		info.BackupExists = true
		info.BackupSize = fi.Size()
	}
	if !info.FileExists {
		return info
	}

	data, err := os.ReadFile(a.exePath)
	if err != nil {
		return info
	}

	for _, def := range patchDefs {
		ps := PatchStatus{ID: def.ID, Name: def.Name, State: "unknown"}
		offset, ok := rvaToFileOffset(data, def.RVA)
		if !ok || int(offset)+def.PatchSize > len(data) {
			info.Patches = append(info.Patches, ps)
			continue
		}
		target := data[offset : offset+uint32(def.PatchSize)]
		ps.CurrentBytes = bytesToHex(target)

		if bytesEqual(target, def.OrigBytes) {
			ps.State = "original"
		} else if def.NeedCave {
			// 跳板补丁：检查是否为 JMP rel32 + NOPs
			if target[0] == 0xE9 && allNop(target[5:]) {
				ps.State = "patched"
				// 读取跳板中的值
				ps.CurrentValue = readCaveValue(data, offset, def)
			}
		} else {
			// 直接补丁：检查 B8 xx xx xx xx + NOP 填充
			if target[0] == 0xB8 && isNopFill(target[5:]) {
				ps.State = "patched"
				ps.CurrentValue = binary.LittleEndian.Uint32(target[1:5])
			}
		}
		info.Patches = append(info.Patches, ps)
	}
	return info
}

// PatchFile 对指定补丁点应用补丁
func (a *App) PatchFile(patchID string, value uint32) error {
	if a.exePath == "" {
		return fmt.Errorf("未选择文件")
	}

	def := findPatchDef(patchID)
	if def == nil {
		return fmt.Errorf("未知补丁: %s", patchID)
	}

	data, err := os.ReadFile(a.exePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	offset, ok := rvaToFileOffset(data, def.RVA)
	if !ok {
		return fmt.Errorf("无法定位 RVA 0x%X", def.RVA)
	}
	if int(offset)+def.PatchSize > len(data) {
		return fmt.Errorf("补丁超出文件范围")
	}

	target := data[offset : offset+uint32(def.PatchSize)]

	// 校验：必须是原始字节或已补丁状态
	isOrig := bytesEqual(target, def.OrigBytes)
	isPatched := false
	if def.NeedCave {
		isPatched = target[0] == 0xE9 && allNop(target[5:])
	} else {
		isPatched = target[0] == 0xB8 && isNopFill(target[5:])
	}
	if !isOrig && !isPatched {
		return fmt.Errorf("目标字节异常，拒绝补丁\n当前: %s", bytesToHex(target))
	}

	if def.NeedCave {
		err = applyCavePatch(data, offset, *def, value, isPatched)
	} else {
		err = applyDirectPatch(data, offset, *def, value)
	}
	if err != nil {
		return err
	}

	return os.WriteFile(a.exePath, data, 0644)
}

// BackupFile 创建备份
func (a *App) BackupFile(force bool) error {
	if a.exePath == "" {
		return fmt.Errorf("未选择文件")
	}
	bakPath := a.exePath + ".bak"
	if _, err := os.Stat(a.exePath); os.IsNotExist(err) {
		return fmt.Errorf("目标文件不存在")
	}
	if !force {
		if _, err := os.Stat(bakPath); err == nil {
			return fmt.Errorf("备份已存在，使用强制覆盖选项")
		}
	}
	data, err := os.ReadFile(a.exePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}
	return os.WriteFile(bakPath, data, 0644)
}

// RestoreFile 从备份恢复
func (a *App) RestoreFile() error {
	if a.exePath == "" {
		return fmt.Errorf("未选择文件")
	}
	bakPath := a.exePath + ".bak"
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在")
	}
	data, err := os.ReadFile(bakPath)
	if err != nil {
		return fmt.Errorf("读取备份失败: %w", err)
	}
	return os.WriteFile(a.exePath, data, 0644)
}

// ── 补丁实现 ──

// applyDirectPatch 直接替换字节（mov eax,imm32 + NOP 填充）
func applyDirectPatch(data []byte, offset uint32, def PatchDef, value uint32) error {
	patch := make([]byte, def.PatchSize)
	patch[0] = 0xB8
	binary.LittleEndian.PutUint32(patch[1:5], value)
	// 剩余字节填 NOP
	switch def.PatchSize - 5 {
	case 4: // 9 字节: mov eax,imm32 + 4-byte NOP (0F 1F 40 00)
		patch[5] = 0x0F; patch[6] = 0x1F; patch[7] = 0x40; patch[8] = 0x00
	case 3: // 8 字节: mov eax,imm32 + 3-byte NOP (0F 1F 00)
		patch[5] = 0x0F; patch[6] = 0x1F; patch[7] = 0x00
	default: // 其他情况用单字节 NOP 填充
		for i := 5; i < def.PatchSize; i++ {
			patch[i] = 0x90
		}
	}
	copy(data[offset:], patch)
	return nil
}

// applyCavePatch 使用代码跳板（用于 likes 类型）
func applyCavePatch(data []byte, offset uint32, def PatchDef, value uint32, alreadyPatched bool) error {
	// 跳板代码布局（17 字节）:
	//   B8 xx xx xx xx    ; mov eax, <value>
	//   89 01             ; mov [rcx], eax
	//   E8 yy yy yy yy   ; call <target>
	//   E9 zz zz zz zz   ; jmp back
	const caveSize = 17

	var caveOffset uint32
	var caveRVA uint32

	if alreadyPatched {
		// 已有跳板，读取 JMP 目标找到 cave 位置
		jmpRel := int32(binary.LittleEndian.Uint32(data[offset+1 : offset+5]))
		jmpNextRVA := def.RVA + 5
		caveRVA = uint32(int32(jmpNextRVA) + jmpRel)
		var ok bool
		caveOffset, ok = rvaToFileOffset(data, caveRVA)
		if !ok {
			return fmt.Errorf("无法定位已有跳板")
		}
	} else {
		// 首次补丁：在 .text 段末尾找空间
		var ok bool
		caveRVA, caveOffset, ok = findCaveSpace(data, caveSize)
		if !ok {
			return fmt.Errorf("找不到可用的代码空间")
		}
	}

	// 写跳板代码
	cave := make([]byte, caveSize)
	cave[0] = 0xB8
	binary.LittleEndian.PutUint32(cave[1:5], value)
	cave[5] = 0x89; cave[6] = 0x01 // mov [rcx], eax

	// call <target>: E8 rel32, rel32 = target - (cave_call_rva + 5)
	cave[7] = 0xE8
	callRVA := caveRVA + 7
	callRel := int32(def.CallTarget) - int32(callRVA+5)
	binary.LittleEndian.PutUint32(cave[8:12], uint32(callRel))

	// jmp back: E9 rel32, rel32 = return_rva - (cave_jmp_rva + 5)
	cave[12] = 0xE9
	returnRVA := def.RVA + uint32(def.PatchSize)
	jmpRVA := caveRVA + 12
	jmpRel := int32(returnRVA) - int32(jmpRVA+5)
	binary.LittleEndian.PutUint32(cave[13:17], uint32(jmpRel))

	copy(data[caveOffset:], cave)

	// 写原始位置的 JMP + NOPs
	patch := make([]byte, def.PatchSize)
	patch[0] = 0xE9
	origJmpRel := int32(caveRVA) - int32(def.RVA+5)
	binary.LittleEndian.PutUint32(patch[1:5], uint32(origJmpRel))
	for i := 5; i < def.PatchSize; i++ {
		patch[i] = 0x90 // NOP
	}
	copy(data[offset:], patch)

	return nil
}

// findCaveSpace 在 PE 段的 rawData 末尾找零填充区，
// 并扩展 VirtualSize + SizeOfImage 确保运行时该区域被映射到内存。
func findCaveSpace(data []byte, size int) (rva uint32, fileOffset uint32, ok bool) {
	peOffset := binary.LittleEndian.Uint32(data[0x3C:0x40])
	coffHeader := peOffset + 4
	numSections := binary.LittleEndian.Uint16(data[coffHeader+2 : coffHeader+4])
	optHeaderSize := binary.LittleEndian.Uint16(data[coffHeader+16 : coffHeader+18])
	sectionStart := coffHeader + 20 + uint32(optHeaderSize)
	optHeader := coffHeader + 20

	// SizeOfImage 在 optional header offset 56 (PE32+)
	sizeOfImageOff := optHeader + 56
	// SectionAlignment 在 optional header offset 32
	sectionAlignment := binary.LittleEndian.Uint32(data[optHeader+32 : optHeader+36])

	for i := uint16(0); i < numSections; i++ {
		off := sectionStart + uint32(i)*40
		if int(off)+40 > len(data) {
			continue
		}
		virtualSize := binary.LittleEndian.Uint32(data[off+8 : off+12])
		virtualAddr := binary.LittleEndian.Uint32(data[off+12 : off+16])
		rawSize := binary.LittleEndian.Uint32(data[off+16 : off+20])
		rawPtr := binary.LittleEndian.Uint32(data[off+20 : off+24])
		characteristics := binary.LittleEndian.Uint32(data[off+36 : off+40])

		isExecutable := (characteristics & 0x20000020) != 0
		if !isExecutable || rawSize == 0 || rawPtr == 0 {
			continue
		}

		rawEnd := rawPtr + rawSize
		if rawEnd > uint32(len(data)) {
			rawEnd = uint32(len(data))
		}

		// 从段 raw 末尾往前找连续零字节
		zeroCount := 0
		for pos := int(rawEnd) - 1; pos >= int(rawPtr) && pos >= 0; pos-- {
			if data[pos] == 0 {
				zeroCount++
			} else {
				break
			}
		}
		if zeroCount < size+16 {
			continue
		}

		caveFileOff := rawEnd - uint32(size) - 8
		caveRVA := virtualAddr + (caveFileOff - rawPtr)

		// 关键：如果 cave 超出 virtualSize，扩展 VirtualSize 使其被映射到内存
		caveEnd := caveRVA - virtualAddr + uint32(size) + 8
		if caveEnd > virtualSize {
			// 对齐到 SectionAlignment
			newVirtualSize := alignUp(caveEnd, sectionAlignment)
			binary.LittleEndian.PutUint32(data[off+8:off+12], newVirtualSize)

			// 更新 SizeOfImage = 最后一个段的 VirtualAddress + 对齐后的 VirtualSize
			// 找最后一个段来计算
			newSizeOfImage := uint32(0)
			for j := uint16(0); j < numSections; j++ {
				soff := sectionStart + uint32(j)*40
				va := binary.LittleEndian.Uint32(data[soff+12 : soff+16])
				vs := binary.LittleEndian.Uint32(data[soff+8 : soff+12])
				end := va + alignUp(vs, sectionAlignment)
				if end > newSizeOfImage {
					newSizeOfImage = end
				}
			}
			binary.LittleEndian.PutUint32(data[sizeOfImageOff:sizeOfImageOff+4], newSizeOfImage)
		}

		return caveRVA, caveFileOff, true
	}
	return 0, 0, false
}

func alignUp(value, alignment uint32) uint32 {
	if alignment == 0 {
		return value
	}
	return (value + alignment - 1) & ^(alignment - 1)
}

// readCaveValue 从跳板中读取当前值
func readCaveValue(data []byte, offset uint32, def PatchDef) uint32 {
	if data[offset] != 0xE9 {
		return 0
	}
	jmpRel := int32(binary.LittleEndian.Uint32(data[offset+1 : offset+5]))
	caveRVA := uint32(int32(def.RVA+5) + jmpRel)
	caveOffset, ok := rvaToFileOffset(data, caveRVA)
	if !ok || int(caveOffset)+5 > len(data) {
		return 0
	}
	if data[caveOffset] != 0xB8 {
		return 0
	}
	return binary.LittleEndian.Uint32(data[caveOffset+1 : caveOffset+5])
}

func allNop(b []byte) bool {
	for _, v := range b {
		if v != 0x90 {
			return false
		}
	}
	return true
}

// isNopFill 检查字节是否为已知的多字节 NOP 填充
func isNopFill(b []byte) bool {
	switch len(b) {
	case 4: // 0F 1F 40 00
		return b[0] == 0x0F && b[1] == 0x1F && b[2] == 0x40 && b[3] == 0x00
	case 3: // 0F 1F 00
		return b[0] == 0x0F && b[1] == 0x1F && b[2] == 0x00
	default:
		return allNop(b)
	}
}

func findPatchDef(id string) *PatchDef {
	for i := range patchDefs {
		if patchDefs[i].ID == id {
			return &patchDefs[i]
		}
	}
	return nil
}

// ── PE / 工具函数 ──

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) { return false }
	for i := range a { if a[i] != b[i] { return false } }
	return true
}

func bytesToHex(b []byte) string {
	parts := make([]string, len(b))
	for i, v := range b { parts[i] = fmt.Sprintf("%02X", v) }
	return strings.Join(parts, " ")
}

func rvaToFileOffset(data []byte, rva uint32) (uint32, bool) {
	if len(data) < 64 { return 0, false }
	if data[0] != 'M' || data[1] != 'Z' { return 0, false }
	peOffset := binary.LittleEndian.Uint32(data[0x3C:0x40])
	if int(peOffset)+24 > len(data) { return 0, false }
	if data[peOffset] != 'P' || data[peOffset+1] != 'E' || data[peOffset+2] != 0 || data[peOffset+3] != 0 {
		return 0, false
	}
	coffHeader := peOffset + 4
	numSections := binary.LittleEndian.Uint16(data[coffHeader+2 : coffHeader+4])
	optHeaderSize := binary.LittleEndian.Uint16(data[coffHeader+16 : coffHeader+18])
	optHeader := coffHeader + 20
	if int(optHeader)+2 > len(data) { return 0, false }
	magic := binary.LittleEndian.Uint16(data[optHeader : optHeader+2])
	if magic != 0x020B { return 0, false }
	sectionStart := optHeader + uint32(optHeaderSize)
	for i := uint16(0); i < numSections; i++ {
		off := sectionStart + uint32(i)*40
		if int(off)+40 > len(data) { return 0, false }
		virtualSize := binary.LittleEndian.Uint32(data[off+8 : off+12])
		virtualAddr := binary.LittleEndian.Uint32(data[off+12 : off+16])
		rawSize := binary.LittleEndian.Uint32(data[off+16 : off+20])
		rawPtr := binary.LittleEndian.Uint32(data[off+20 : off+24])
		span := rawSize
		if virtualSize > span { span = virtualSize }
		if rva >= virtualAddr && rva < virtualAddr+span {
			return rawPtr + (rva - virtualAddr), true
		}
	}
	return 0, false
}

// ── Steam 路径扫描 ──

func findSteamLibraryFolders() []string {
	var dirs []string
	steamPath := ""
	for _, keyPath := range []string{`SOFTWARE\Valve\Steam`, `SOFTWARE\WOW6432Node\Valve\Steam`} {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
		if err != nil { continue }
		val, _, err := k.GetStringValue("InstallPath")
		k.Close()
		if err == nil && val != "" { steamPath = val; dirs = append(dirs, val); break }
	}
	if steamPath == "" {
		k, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Valve\Steam`, registry.QUERY_VALUE)
		if err == nil {
			val, _, err := k.GetStringValue("SteamPath")
			k.Close()
			if err == nil && val != "" { steamPath = filepath.FromSlash(val); dirs = append(dirs, steamPath) }
		}
	}
	if steamPath != "" {
		vdfPath := filepath.Join(steamPath, "steamapps", "libraryfolders.vdf")
		if data, err := os.ReadFile(vdfPath); err == nil {
			dirs = append(dirs, parseLibraryPaths(string(data))...)
		}
	}
	for _, fb := range []string{
		`C:\Program Files (x86)\Steam`, `C:\Program Files\Steam`,
		`D:\Steam`, `D:\SteamLibrary`, `E:\Steam`, `E:\SteamLibrary`,
	} {
		found := false
		for _, d := range dirs { if strings.EqualFold(d, fb) { found = true; break } }
		if !found { dirs = append(dirs, fb) }
	}
	return dirs
}

func parseLibraryPaths(content string) []string {
	var paths []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"path"`) {
			parts := strings.SplitN(line, `"path"`, 2)
			if len(parts) < 2 { continue }
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"`)
			val = strings.ReplaceAll(val, `\\`, `\`)
			if val != "" { paths = append(paths, val) }
		}
	}
	return paths
}
