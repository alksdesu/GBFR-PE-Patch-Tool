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
	Aob       string
	TargetOff int
	PatchLen  int
	NopOnly   bool
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

type caveHookState struct {
	addr     uintptr
	orig     []byte
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

func (a *App) scanCaveAob(pattern []byte, mask []bool) (uintptr, error) {
	moduleSize, err := getRemoteModuleSize(a.hProcess, a.moduleBase)
	if err != nil {
		return 0, err
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
		if len(matches) > 1 {
			return 0, fmt.Errorf("特征码命中多处，版本可能不匹配")
		}
	}
	if len(matches) == 0 {
		return 0, fmt.Errorf("未找到特征码")
	}
	return matches[0], nil
}

func (a *App) caveResolveAobs(def *caveDef) (map[string]uintptr, error) {
	addrs := make(map[string]uintptr, len(def.Aobs))
	for _, ab := range def.Aobs {
		addr, err := a.scanCaveAob(ab.Pattern, ab.Mask)
		if err != nil {
			return nil, fmt.Errorf("%s 定位 %s 失败: %w", def.Name, ab.Sym, err)
		}
		addrs[ab.Sym] = addr
	}
	return addrs, nil
}

func (a *App) caveIsEnabled(def *caveDef) (bool, map[string]uintptr, error) {
	addrs, err := a.caveResolveAobs(def)
	if err != nil {
		return false, nil, err
	}
	for _, h := range def.Hooks {
		addr := addrs[h.Aob]
		buf := make([]byte, 1)
		if err := readProcessMemory(a.hProcess, addr, unsafe.Pointer(&buf[0]), 1); err != nil {
			return false, addrs, err
		}
		if h.NopOnly {
			if buf[0] != 0x90 {
				return false, addrs, nil
			}
		} else if buf[0] != 0xE9 {
			return false, addrs, nil
		}
	}
	return true, addrs, nil
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

func (a *App) caveBuildAndWrite(def *caveDef, addrs map[string]uintptr) (*caveRuntime, error) {
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
	for _, h := range def.Hooks {
		addr := addrs[h.Aob]
		orig := make([]byte, h.PatchLen)
		if err := readProcessMemory(a.hProcess, addr, unsafe.Pointer(&orig[0]), uintptr(h.PatchLen)); err != nil {
			a.caveRestoreRuntime(rt)
			return nil, fmt.Errorf("%s 读取 hook 原字节失败: %w", def.Name, err)
		}
		rt.hooks = append(rt.hooks, caveHookState{addr: addr, orig: orig})
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
		addr := rt.hooks[i].addr
		patch := make([]byte, h.PatchLen)
		for j := range patch {
			patch[j] = 0x90
		}
		if !h.NopOnly {
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
	return nil
}

func (a *App) caveRestoreRuntime(rt *caveRuntime) {
	for _, h := range rt.hooks {
		_ = writeCodeMemory(a.hProcess, h.addr, h.orig)
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
		addrs, err := a.caveResolveAobs(def)
		if err != nil {
			return CaveState{}, err
		}
		newRt, err := a.caveBuildAndWrite(def, addrs)
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
