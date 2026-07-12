package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"
)

type caveAob struct {
	Sym     string
	Pattern []byte
	Mask    []bool
}

type caveHook struct {
	Aob        string
	TargetOff  int
	PatchLen   int
	NopOnly    bool
	PatchBytes []byte
}

type caveBak struct {
	Off int
	Aob string
	Len int
}

type caveReloc struct {
	Off  int
	Type string
	Aob  string
}

type caveDef struct {
	ID     string
	Name   string
	Group  string
	Code   []byte
	Aobs   []caveAob
	Hooks  []caveHook
	Baks   []caveBak
	Relocs []caveReloc
	Data   map[string]int
}

type caveRuntime struct {
	base     uintptr
	aobAddrs map[string]uintptr
	hooks    []caveHookState
	active   bool
}

// caveHookState 记录一个 hook 的全部命中地址及各自原始字节。
// 代码洞型 hook 只有单地址; 纯字节 patch 型的多命中 AOB 会有多个地址, 全部应用同一 patch。
type caveHookState struct {
	addrs []uintptr
	origs [][]byte
}

type CaveState struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Group    string `json:"group"`
	Enabled  bool   `json:"enabled"`
	HasFloat bool   `json:"hasFloat"`
	HasInt   bool   `json:"hasInt"`
	Flags    int    `json:"flags"`
}

func findCaveDef(id string) *caveDef {
	for i := range combatCaves {
		if combatCaves[i].ID == id {
			return &combatCaves[i]
		}
	}
	return nil
}

func (a *App) scanCaveAobAll(pattern []byte, mask []bool) ([]uintptr, error) {
	moduleSize, err := getRemoteModuleSize(a.hProcess, a.moduleBase)
	if err != nil {
		return nil, err
	}
	const chunkSize uintptr = 0x10000
	var matches []uintptr
	var carry []byte
	var carryBase uintptr
	for off := uintptr(0); off < moduleSize; off += chunkSize {
		size := chunkSize
		if off+size > moduleSize {
			size = moduleSize - off
		}
		buf := make([]byte, size)
		addr := a.moduleBase + off
		if err := readProcessMemory(a.hProcess, addr, unsafe.Pointer(&buf[0]), uintptr(len(buf))); err != nil {
			carry = nil
			continue
		}
		scanBuf, scanBase := buf, addr
		if len(carry) > 0 {
			scanBuf = append(append([]byte{}, carry...), buf...)
			scanBase = carryBase
		}
		matches = append(matches, findPatternMatches(scanBuf, scanBase, pattern, mask)...)
		if len(buf) >= len(pattern)-1 {
			carry = append([]byte{}, buf[len(buf)-len(pattern)+1:]...)
			carryBase = addr + uintptr(len(buf)-len(pattern)+1)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("未找到特征码")
	}
	return matches, nil
}

func (a *App) scanCaveAob(pattern []byte, mask []bool) (uintptr, error) {
	matches, err := a.scanCaveAobAll(pattern, mask)
	if err != nil {
		return 0, err
	}
	if len(matches) > 1 {
		return 0, fmt.Errorf("特征码命中多处，版本可能不匹配")
	}
	return matches[0], nil
}

// caveResolveAobs 解析 cave 全部 AOB。代码洞型要求每个 AOB 唯一命中(跳转锚点须确定);
// 纯字节 patch 型(Code 为空)允许 AOB 多命中, 运行时对所有命中处应用同一 patch。
func (a *App) caveResolveAobs(def *caveDef) (map[string]uintptr, map[string][]uintptr, error) {
	addrs := make(map[string]uintptr, len(def.Aobs))
	multi := make(map[string][]uintptr, len(def.Aobs))
	pureByte := len(def.Code) == 0
	for _, ab := range def.Aobs {
		hits, err := a.scanCaveAobAll(ab.Pattern, ab.Mask)
		if err != nil {
			return nil, nil, fmt.Errorf("%s 定位 %s 失败: %w", def.Name, ab.Sym, err)
		}
		if len(hits) > 1 && !pureByte {
			return nil, nil, fmt.Errorf("%s 定位 %s 失败: 特征码命中多处，版本可能不匹配", def.Name, ab.Sym)
		}
		addrs[ab.Sym] = hits[0]
		multi[ab.Sym] = hits
	}
	return addrs, multi, nil
}

func (a *App) caveState(def *caveDef, enabled bool) CaveState {
	st := CaveState{ID: def.ID, Name: def.Name, Group: def.Group, Enabled: enabled}
	for k := range def.Data {
		switch {
		case hasSuffix(k, "_flt"):
			st.HasFloat = true
		case hasSuffix(k, "_int"):
			st.HasInt = true
		case hasSuffix(k, "_flg"):
			st.Flags++
		}
	}
	return st
}

func hasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}

// caveCaptureHookStates 为每个 hook 记录全部命中地址(纯 patch 型可多命中)及原始字节。
func (a *App) caveCaptureHookStates(def *caveDef, rt *caveRuntime, multi map[string][]uintptr) error {
	pureByte := len(def.Code) == 0
	for _, h := range def.Hooks {
		hits := multi[h.Aob]
		if !pureByte {
			hits = hits[:1] // 代码洞型只用唯一命中(跳转锚点)
		}
		st := caveHookState{}
		for _, addr := range hits {
			orig := make([]byte, h.PatchLen)
			if err := readProcessMemory(a.hProcess, addr, unsafe.Pointer(&orig[0]), uintptr(h.PatchLen)); err != nil {
				return fmt.Errorf("%s 读取 hook 原字节失败: %w", def.Name, err)
			}
			st.addrs = append(st.addrs, addr)
			st.origs = append(st.origs, orig)
		}
		rt.hooks = append(rt.hooks, st)
	}
	return nil
}

func (a *App) caveBuildAndWrite(def *caveDef, addrs map[string]uintptr, multi map[string][]uintptr) (*caveRuntime, error) {
	// 纯字节 patch 型 (Code 为空): 无需代码洞, 仅按 hook 覆盖字节(支持 AOB 多命中)。
	if len(def.Code) == 0 {
		rt := &caveRuntime{aobAddrs: addrs}
		if err := a.caveCaptureHookStates(def, rt, multi); err != nil {
			a.caveRestoreRuntime(rt)
			return nil, err
		}
		if err := a.caveWriteHooks(def, rt); err != nil {
			a.caveRestoreRuntime(rt)
			return nil, err
		}
		return rt, nil
	}

	anchor := addrs[def.Aobs[0].Sym]
	cave, err := virtualAllocRemoteNear(a.hProcess, anchor, 0x1000)
	if err != nil {
		return nil, fmt.Errorf("%s 分配代码洞失败: %w", def.Name, err)
	}

	code := append([]byte{}, def.Code...)

	for _, b := range def.Baks {
		src := addrs[b.Aob]
		if b.Aob == "" {
			continue
		}
		orig := make([]byte, b.Len)
		if err := readProcessMemory(a.hProcess, src, unsafe.Pointer(&orig[0]), uintptr(b.Len)); err != nil {
			_ = virtualFreeRemote(a.hProcess, cave)
			return nil, fmt.Errorf("%s 读取原始字节失败: %w", def.Name, err)
		}
		copy(code[b.Off:b.Off+b.Len], orig)
	}

	for _, r := range def.Relocs {
		var target uintptr
		switch r.Type {
		case "return":
			var hook *caveHook
			for i := range def.Hooks {
				if def.Hooks[i].Aob == r.Aob {
					hook = &def.Hooks[i]
					break
				}
			}
			if hook == nil {
				_ = virtualFreeRemote(a.hProcess, cave)
				return nil, fmt.Errorf("%s reloc 未找到 hook %s", def.Name, r.Aob)
			}
			target = addrs[r.Aob] + uintptr(hook.PatchLen)
		case "call":
			target = addrs[r.Aob]
		default:
			_ = virtualFreeRemote(a.hProcess, cave)
			return nil, fmt.Errorf("%s 未知 reloc 类型 %s", def.Name, r.Type)
		}
		site := cave + uintptr(r.Off)
		rel := int64(target) - int64(site+4)
		if rel < math.MinInt32 || rel > math.MaxInt32 {
			_ = virtualFreeRemote(a.hProcess, cave)
			return nil, fmt.Errorf("%s reloc 距离超过 rel32", def.Name)
		}
		binary.LittleEndian.PutUint32(code[r.Off:r.Off+4], uint32(int32(rel)))
	}

	if err := writeProcessMemory(a.hProcess, cave, unsafe.Pointer(&code[0]), uintptr(len(code))); err != nil {
		_ = virtualFreeRemote(a.hProcess, cave)
		return nil, fmt.Errorf("%s 写入代码洞失败: %w", def.Name, err)
	}

	rt := &caveRuntime{base: cave, aobAddrs: addrs}
	if err := a.caveCaptureHookStates(def, rt, multi); err != nil {
		a.caveRestoreRuntime(rt)
		return nil, err
	}
	if err := a.caveWriteHooks(def, rt); err != nil {
		a.caveRestoreRuntime(rt)
		return nil, err
	}
	return rt, nil
}

func (a *App) caveWriteHooks(def *caveDef, rt *caveRuntime) error {
	for i := range def.Hooks {
		h := def.Hooks[i]
		for _, addr := range rt.hooks[i].addrs {
			patch := make([]byte, h.PatchLen)
			switch {
			case len(h.PatchBytes) > 0:
				copy(patch, h.PatchBytes)
			case h.NopOnly:
				for j := range patch {
					patch[j] = 0x90
				}
			default:
				for j := range patch {
					patch[j] = 0x90
				}
				rel := int64(rt.base+uintptr(h.TargetOff)) - int64(addr+5)
				if rel < math.MinInt32 || rel > math.MaxInt32 {
					return fmt.Errorf("%s hook 跳转超过 rel32", def.Name)
				}
				patch[0] = 0xE9
				binary.LittleEndian.PutUint32(patch[1:5], uint32(int32(rel)))
			}
			if err := writeCodeMemory(a.hProcess, addr, patch); err != nil {
				return fmt.Errorf("%s 写入 hook 失败: %w", def.Name, err)
			}
		}
	}
	return nil
}

func (a *App) caveRestoreRuntime(rt *caveRuntime) {
	for _, h := range rt.hooks {
		for i, addr := range h.addrs {
			_ = writeCodeMemory(a.hProcess, addr, h.origs[i])
		}
	}
}

func (a *App) CaveList() ([]CaveState, error) {
	if err := a.ensureGameProcess(); err != nil {
		return nil, err
	}
	states := make([]CaveState, 0, len(combatCaves))
	for i := range combatCaves {
		def := &combatCaves[i]
		enabled := false
		if rt := a.caveRuntimes[def.ID]; rt != nil && rt.active {
			enabled = true
		}
		states = append(states, a.caveState(def, enabled))
	}
	return states, nil
}

func (a *App) CaveSetEnabled(id string, enabled bool) (CaveState, error) {
	if err := a.ensureGameProcess(); err != nil {
		return CaveState{}, err
	}
	def := findCaveDef(id)
	if def == nil {
		return CaveState{}, fmt.Errorf("未知功能: %s", id)
	}
	if a.caveRuntimes == nil {
		a.caveRuntimes = make(map[string]*caveRuntime)
	}
	rt := a.caveRuntimes[id]
	if enabled {
		if rt != nil && rt.active {
			return a.caveState(def, true), nil
		}
		if rt != nil {
			if err := a.caveWriteHooks(def, rt); err != nil {
				return CaveState{}, err
			}
			rt.active = true
			return a.caveState(def, true), nil
		}
		addrs, multi, err := a.caveResolveAobs(def)
		if err != nil {
			return CaveState{}, err
		}
		newRt, err := a.caveBuildAndWrite(def, addrs, multi)
		if err != nil {
			return CaveState{}, err
		}
		newRt.active = true
		a.caveRuntimes[id] = newRt
		return a.caveState(def, true), nil
	}
	if rt == nil || !rt.active {
		return a.caveState(def, false), nil
	}
	a.caveRestoreRuntime(rt)
	rt.active = false
	return a.caveState(def, false), nil
}

func (a *App) caveDataAddr(id, sym string, extra int) (uintptr, *caveDef, error) {
	def := findCaveDef(id)
	if def == nil {
		return 0, nil, fmt.Errorf("未知功能: %s", id)
	}
	rt := a.caveRuntimes[id]
	if rt == nil || !rt.active {
		return 0, nil, fmt.Errorf("%s 未开启", def.Name)
	}
	off, ok := def.Data[sym]
	if !ok {
		return 0, nil, fmt.Errorf("%s 无数据符号 %s", def.Name, sym)
	}
	return rt.base + uintptr(off+extra), def, nil
}

func (a *App) CaveSetFloat(id, sym string, extra int, value float64) error {
	if err := a.ensureGameProcess(); err != nil {
		return err
	}
	addr, _, err := a.caveDataAddr(id, sym, extra)
	if err != nil {
		return err
	}
	return writeFloat32Remote(a.hProcess, addr, float32(value))
}

func (a *App) CaveSetInt(id, sym string, extra int, value int32) error {
	if err := a.ensureGameProcess(); err != nil {
		return err
	}
	addr, _, err := a.caveDataAddr(id, sym, extra)
	if err != nil {
		return err
	}
	return writeUint32Remote(a.hProcess, addr, uint32(value))
}

func (a *App) CaveSetFlag(id, sym string, byteIdx int, on bool) error {
	if err := a.ensureGameProcess(); err != nil {
		return err
	}
	addr, _, err := a.caveDataAddr(id, sym, byteIdx)
	if err != nil {
		return err
	}
	var b [1]byte
	if on {
		b[0] = 1
	}
	return writeProcessMemory(a.hProcess, addr, unsafe.Pointer(&b[0]), 1)
}

func (a *App) CaveReadPointer(id, sym string) (uint64, error) {
	if err := a.ensureGameProcess(); err != nil {
		return 0, err
	}
	addr, _, err := a.caveDataAddr(id, sym, 0)
	if err != nil {
		return 0, err
	}
	var ptr uint64
	if err := readProcessMemory(a.hProcess, addr, unsafe.Pointer(&ptr), 8); err != nil {
		return 0, err
	}
	return ptr, nil
}

func (a *App) caveRestoreAll() {
	if a.hProcess == 0 || a.moduleBase == 0 {
		return
	}
	for id, rt := range a.caveRuntimes {
		if rt.active {
			a.caveRestoreRuntime(rt)
		}
		delete(a.caveRuntimes, id)
	}
}
