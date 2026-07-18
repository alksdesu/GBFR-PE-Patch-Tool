package main

import (
	"encoding/binary"
	"testing"
)

func TestBuildWrightstoneMemoryCavePreservesR10(t *testing.T) {
	const cave = uintptr(0x10000000)
	original := []byte{0x8B, 0x02, 0x39, 0x06, 0x75, 0x04, 0x33, 0xC0}

	code, err := buildWrightstoneMemoryCave(cave, cave+0x100, original)
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := []byte{0x41, 0x52, 0x49, 0xBA}
	for i, b := range wantPrefix {
		if code[i] != b {
			t.Fatalf("byte %d = 0x%02X, want 0x%02X", i, code[i], b)
		}
	}
	if got := uintptr(binary.LittleEndian.Uint64(code[4:12])); got != cave+wrightstoneMemoryCaveDataOffset {
		t.Fatalf("saved pointer address = 0x%X, want 0x%X", got, cave+wrightstoneMemoryCaveDataOffset)
	}
	if code[12] != 0x49 || code[13] != 0x89 || code[14] != 0x12 || code[15] != 0x41 || code[16] != 0x5A {
		t.Fatalf("hook does not restore r10: % X", code[12:17])
	}
	for i, b := range original {
		if got := code[17+i]; got != b {
			t.Fatalf("original byte %d = 0x%02X, want 0x%02X", i, got, b)
		}
	}
}
