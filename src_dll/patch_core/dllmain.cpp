// dllmain.cpp : 定义 DLL 应用程序的入口点。
#include "pch.h"
#include <Windows.h>
#include <cstdint>
#include <cstdlib>
#include <libmem/libmem.h>
#include <cstdio>
#include <cstring>

struct PatchPoint
{
    const char* id;
    const wchar_t* name;
    lm_address_t rva;
    const lm_byte_t* expected;
    lm_size_t size;
    const lm_byte_t* patch;
    bool hook;
};

static const lm_byte_t kLinkTimeExpected[] = { 0xC4, 0xC1, 0x7A, 0x11, 0x9C, 0x24, 0xB4, 0x01, 0x00, 0x00 };
static const lm_byte_t kLinkTimeDisablePatch[] = { 0xC4, 0xC1, 0x7A, 0x11, 0x84, 0x24, 0xB4, 0x01, 0x00, 0x00 };
static const lm_byte_t kNop10[] = { 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90 };

static const lm_byte_t kDamageExpected[] = { 0x01, 0x91, 0xB8, 0x15, 0x00, 0x00 };
static const lm_byte_t kStunExpected[] = { 0xC4, 0xC1, 0x4A, 0x58, 0x85, 0x20, 0x07, 0x00, 0x00 };

static const lm_byte_t kPurpleExpected[] = { 0xC4, 0xC1, 0x7A, 0x11, 0x85, 0x10, 0x0A, 0x00, 0x00 };
static const lm_byte_t kBlueGrowExpected[] = { 0xC4, 0xC1, 0x7A, 0x11, 0x85, 0x20, 0x07, 0x00, 0x00 };
static const lm_byte_t kBlueDrainExpected[] = { 0xC4, 0xC1, 0x7A, 0x11, 0x85, 0x70, 0x0A, 0x00, 0x00 };
static const lm_byte_t kNop9[] = { 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90 };

static const PatchPoint kMonsterPatches[] = {
    { "link_time_no_drain", L"link time no drain", 0x187228, kLinkTimeExpected, sizeof(kLinkTimeExpected), kNop10, false },
    { "link_time_disable", L"disable link time", 0x187228, kLinkTimeExpected, sizeof(kLinkTimeExpected), kLinkTimeDisablePatch, false },
    { "monster_hp", L"monster hp", 0x1B3F798, kDamageExpected, sizeof(kDamageExpected), nullptr, true },
    { "monster_stun", L"monster stun", 0xA09ADF, kStunExpected, sizeof(kStunExpected), nullptr, true },
    { "purple_drain", L"purple bar drain", 0xA0379A, kPurpleExpected, sizeof(kPurpleExpected), kNop9, false },
    { "blue_grow", L"blue bar grow", 0xA09AF1, kBlueGrowExpected, sizeof(kBlueGrowExpected), kNop9, false },
    { "blue_drain", L"blue bar drain", 0xA03F38, kBlueDrainExpected, sizeof(kBlueDrainExpected), kNop9, false },
};

static bool BytesEqual(const lm_byte_t* a, const lm_byte_t* b, lm_size_t size)
{
    for (lm_size_t i = 0; i < size; ++i)
    {
        if (a[i] != b[i]) return false;
    }
    return true;
}

static bool PatchBytes(lm_address_t target, const lm_byte_t* patch, lm_size_t size)
{
    lm_prot_t oldProt{};
    if (!LM_ProtMemory(target, size, LM_PROT_XRW, &oldProt)) return false;

    bool ok = LM_WriteMemory(target, patch, size) == size;
    LM_ProtMemory(target, size, oldProt, nullptr);
    FlushInstructionCache(GetCurrentProcess(), reinterpret_cast<void*>(target), size);
    return ok;
}

static bool ReadPatchId(char* patchId, DWORD patchIdSize)
{
    wchar_t tempPath[MAX_PATH]{};
    if (!GetTempPathW(_countof(tempPath), tempPath)) return false;

    wchar_t commandPath[MAX_PATH]{};
    swprintf_s(commandPath, L"%sgbfr-player-info-edit\\patch_core_command.txt", tempPath);

    HANDLE file = CreateFileW(commandPath, GENERIC_READ, FILE_SHARE_READ | FILE_SHARE_WRITE, nullptr, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, nullptr);
    if (file == INVALID_HANDLE_VALUE) return false;

    DWORD read = 0;
    BOOL ok = ReadFile(file, patchId, patchIdSize - 1, &read, nullptr);
    CloseHandle(file);
    if (!ok || read == 0) return false;

    patchId[read] = '\0';
    for (DWORD i = 0; i < read; ++i)
    {
        if (patchId[i] == '\r' || patchId[i] == '\n')
        {
            patchId[i] = '\0';
            break;
        }
    }
    return patchId[0] != '\0';
}

static bool PatchIdEquals(const char* requestedId, const char* pointId)
{
    size_t n = strlen(pointId);
    return strncmp(requestedId, pointId, n) == 0 && (requestedId[n] == '\0' || requestedId[n] == ' ' || requestedId[n] == '\t');
}

static float ReadScale()
{
    char patchId[64]{};
    if (!ReadPatchId(patchId, sizeof(patchId))) return 1.0f;

    char* space = strchr(patchId, ' ');
    if (!space) return 1.0f;
    float scale = static_cast<float>(atof(space + 1));
    if (scale <= 0.0f || scale > 1000.0f) return 1.0f;
    return scale;
}

static lm_address_t AllocNear(lm_address_t target, size_t size)
{
    const uintptr_t granularity = 0x10000;
    const uintptr_t maxDistance = 0x7FFF0000;
    uintptr_t base = target & ~(granularity - 1);

    for (uintptr_t step = 0; step <= maxDistance; step += granularity)
    {
        uintptr_t candidates[2]{};
        int count = 0;
        if (base >= step) candidates[count++] = base - step;
        if (base <= UINTPTR_MAX - step) candidates[count++] = base + step;

        for (int i = 0; i < count; ++i)
        {
            void* ptr = VirtualAlloc(reinterpret_cast<void*>(candidates[i]), size, MEM_COMMIT | MEM_RESERVE, PAGE_EXECUTE_READWRITE);
            if (!ptr) continue;

            int64_t delta = static_cast<int64_t>(reinterpret_cast<uintptr_t>(ptr)) - static_cast<int64_t>(target + 5);
            if (delta >= INT32_MIN && delta <= INT32_MAX)
            {
                return reinterpret_cast<lm_address_t>(ptr);
            }
            VirtualFree(ptr, 0, MEM_RELEASE);
        }
    }

    return LM_ADDRESS_BAD;
}

static bool PatchDamageHook(lm_address_t target, wchar_t* message, size_t messageSize)
{
    float scale = ReadScale();
    lm_address_t cave = AllocNear(target, 128);
    if (cave == LM_ADDRESS_BAD)
    {
        swprintf_s(message, messageSize, L"alloc near failed: monster hp");
        return false;
    }

    lm_byte_t code[64]{};
    size_t i = 0;
    code[i++] = 0x48; code[i++] = 0x83; code[i++] = 0xEC; code[i++] = 0x28;                         // sub rsp,28
    code[i++] = 0x0F; code[i++] = 0x11; code[i++] = 0x04; code[i++] = 0x24;                         // movups [rsp],xmm0
    code[i++] = 0xF3; code[i++] = 0x0F; code[i++] = 0x2A; code[i++] = 0xC2;                         // cvtsi2ss xmm0,edx
    code[i++] = 0xF3; code[i++] = 0x0F; code[i++] = 0x59; code[i++] = 0x05;                         // mulss xmm0,[rip+disp32]
    size_t scaleDisp = i; i += 4;
    code[i++] = 0xF3; code[i++] = 0x0F; code[i++] = 0x2C; code[i++] = 0xD0;                         // cvttss2si edx,xmm0
    code[i++] = 0x0F; code[i++] = 0x10; code[i++] = 0x04; code[i++] = 0x24;                         // movups xmm0,[rsp]
    code[i++] = 0x48; code[i++] = 0x83; code[i++] = 0xC4; code[i++] = 0x28;                         // add rsp,28
    code[i++] = 0x01; code[i++] = 0x91; code[i++] = 0xB8; code[i++] = 0x15; code[i++] = 0x00; code[i++] = 0x00; // add [rcx+15B8],edx
    code[i++] = 0xE9;                                                                               // jmp return
    size_t jmpBackDisp = i; i += 4;
    size_t scaleOffset = i;
    memcpy(code + i, &scale, sizeof(scale)); i += sizeof(scale);

    int64_t scaleDelta = static_cast<int64_t>(cave + scaleOffset) - static_cast<int64_t>(cave + scaleDisp + 4);
    if (scaleDelta < INT32_MIN || scaleDelta > INT32_MAX)
    {
        swprintf_s(message, messageSize, L"scale jump out of range: monster hp");
        return false;
    }
    int32_t relScale = static_cast<int32_t>(scaleDelta);
    memcpy(code + scaleDisp, &relScale, sizeof(relScale));

    int64_t backDelta = static_cast<int64_t>(target + 6) - static_cast<int64_t>(cave + jmpBackDisp + 4);
    if (backDelta < INT32_MIN || backDelta > INT32_MAX)
    {
        swprintf_s(message, messageSize, L"return jump out of range: monster hp");
        return false;
    }
    int32_t relBack = static_cast<int32_t>(backDelta);
    memcpy(code + jmpBackDisp, &relBack, sizeof(relBack));

    if (LM_WriteMemory(cave, code, i) != i)
    {
        swprintf_s(message, messageSize, L"cave write failed: monster hp");
        return false;
    }

    lm_byte_t jmp[6]{};
    jmp[0] = 0xE9;
    int32_t rel = static_cast<int32_t>(cave - (target + 5));
    memcpy(jmp + 1, &rel, sizeof(rel));
    jmp[5] = 0x90;
    if (!PatchBytes(target, jmp, sizeof(jmp)))
    {
        swprintf_s(message, messageSize, L"hook write failed: monster hp");
        return false;
    }
    return true;
}

static bool PatchStunHook(lm_address_t target, wchar_t* message, size_t messageSize)
{
    float scale = ReadScale();
    lm_address_t cave = AllocNear(target, 128);
    if (cave == LM_ADDRESS_BAD)
    {
        swprintf_s(message, messageSize, L"alloc near failed: monster stun");
        return false;
    }

    lm_byte_t code[64]{};
    size_t i = 0;
    code[i++] = 0x50;                                                                               // push rax
    code[i++] = 0x48; code[i++] = 0x83; code[i++] = 0xEC; code[i++] = 0x20;                         // sub rsp,20
    code[i++] = 0x0F; code[i++] = 0x11; code[i++] = 0x34; code[i++] = 0x24;                         // movups [rsp],xmm6
    code[i++] = 0xF3; code[i++] = 0x0F; code[i++] = 0x59; code[i++] = 0x35;                         // mulss xmm6,[rip+disp32]
    size_t scaleDisp = i; i += 4;
    code[i++] = 0x49; code[i++] = 0x8D; code[i++] = 0x85; code[i++] = 0x20; code[i++] = 0x07; code[i++] = 0x00; code[i++] = 0x00; // lea rax,[r13+720]
    code[i++] = 0xC5; code[i++] = 0xCA; code[i++] = 0x58; code[i++] = 0x00;                         // vaddss xmm0,xmm6,[rax]
    code[i++] = 0x0F; code[i++] = 0x10; code[i++] = 0x34; code[i++] = 0x24;                         // movups xmm6,[rsp]
    code[i++] = 0x48; code[i++] = 0x83; code[i++] = 0xC4; code[i++] = 0x20;                         // add rsp,20
    code[i++] = 0x58;                                                                               // pop rax
    code[i++] = 0xE9;                                                                               // jmp return
    size_t jmpBackDisp = i; i += 4;
    size_t scaleOffset = i;
    memcpy(code + i, &scale, sizeof(scale)); i += sizeof(scale);

    int64_t scaleDelta = static_cast<int64_t>(cave + scaleOffset) - static_cast<int64_t>(cave + scaleDisp + 4);
    if (scaleDelta < INT32_MIN || scaleDelta > INT32_MAX)
    {
        swprintf_s(message, messageSize, L"scale jump out of range: monster stun");
        return false;
    }
    int32_t relScale = static_cast<int32_t>(scaleDelta);
    memcpy(code + scaleDisp, &relScale, sizeof(relScale));

    int64_t backDelta = static_cast<int64_t>(target + 9) - static_cast<int64_t>(cave + jmpBackDisp + 4);
    if (backDelta < INT32_MIN || backDelta > INT32_MAX)
    {
        swprintf_s(message, messageSize, L"return jump out of range: monster stun");
        return false;
    }
    int32_t relBack = static_cast<int32_t>(backDelta);
    memcpy(code + jmpBackDisp, &relBack, sizeof(relBack));

    if (LM_WriteMemory(cave, code, i) != i)
    {
        swprintf_s(message, messageSize, L"cave write failed: monster stun");
        return false;
    }

    lm_byte_t jmp[9]{ 0xE9, 0, 0, 0, 0, 0x90, 0x90, 0x90, 0x90 };
    int64_t hookDelta = static_cast<int64_t>(cave) - static_cast<int64_t>(target + 5);
    if (hookDelta < INT32_MIN || hookDelta > INT32_MAX)
    {
        swprintf_s(message, messageSize, L"hook jump out of range: monster stun");
        return false;
    }
    int32_t rel = static_cast<int32_t>(hookDelta);
    memcpy(jmp + 1, &rel, sizeof(rel));
    if (!PatchBytes(target, jmp, sizeof(jmp)))
    {
        swprintf_s(message, messageSize, L"hook write failed: monster stun");
        return false;
    }
    return true;
}

static bool ShouldApply(const char* requestedId, const PatchPoint& point)
{
    return strcmp(requestedId, "all") == 0 || PatchIdEquals(requestedId, point.id);
}

static bool ApplyMonsterPatches(wchar_t* message, size_t messageSize)
{
    char patchId[64]{};
    if (!ReadPatchId(patchId, sizeof(patchId)))
    {
        strcpy_s(patchId, "all");
    }

    lm_module_t module{};
    if (!LM_FindModule("granblue_fantasy_relink.exe", &module))
    {
        swprintf_s(message, messageSize, L"LM_FindModule failed");
        return false;
    }

    int patched = 0;
    int already = 0;
    int selected = 0;

    for (const auto& point : kMonsterPatches)
    {
        if (!ShouldApply(patchId, point)) continue;
        ++selected;

        lm_address_t target = module.base + point.rva;
        lm_byte_t current[16]{};
        if (point.size > sizeof(current) || LM_ReadMemory(target, current, point.size) != point.size)
        {
            swprintf_s(message, messageSize, L"read failed: %s at +%llX", point.name, static_cast<unsigned long long>(point.rva));
            return false;
        }

        if (point.hook && current[0] == 0xE9)
        {
            ++already;
            continue;
        }
        if (!point.hook && BytesEqual(current, point.patch, point.size))
        {
            ++already;
            continue;
        }

        if (!BytesEqual(current, point.expected, point.size))
        {
            swprintf_s(message, messageSize, L"unexpected bytes: %s at +%llX", point.name, static_cast<unsigned long long>(point.rva));
            return false;
        }

        if (point.hook)
        {
            if (strcmp(point.id, "monster_stun") == 0)
            {
                if (!PatchStunHook(target, message, messageSize)) return false;
            }
            else if (!PatchDamageHook(target, message, messageSize)) return false;
        }
        else if (!PatchBytes(target, point.patch, point.size))
        {
            swprintf_s(message, messageSize, L"write failed: %s at +%llX", point.name, static_cast<unsigned long long>(point.rva));
            return false;
        }
        ++patched;
    }

    if (selected == 0)
    {
        swprintf_s(message, messageSize, L"unknown patch id: %S", patchId);
        return false;
    }

    swprintf_s(message, messageSize, L"monster enhance ok: id %S patched %d, already %d", patchId, patched, already);
    return true;
}

static DWORD WINAPI InitThread(LPVOID)
{
    wchar_t message[256]{};
    bool ok = ApplyMonsterPatches(message, _countof(message));

    wchar_t debugMessage[320]{};
    swprintf_s(debugMessage, L"[patch_core] %s\n", message);
    OutputDebugStringW(debugMessage);

    MessageBoxW(nullptr, message, ok ? L"patch_core success" : L"patch_core failed", ok ? MB_OK | MB_ICONINFORMATION : MB_OK | MB_ICONERROR);
    return 0;
}

BOOL APIENTRY DllMain( HMODULE hModule,
                       DWORD  ul_reason_for_call,
                       LPVOID lpReserved
                     )
{
    switch (ul_reason_for_call)
    {
    case DLL_PROCESS_ATTACH:
        DisableThreadLibraryCalls(hModule);
        if (HANDLE thread = CreateThread(nullptr, 0, InitThread, nullptr, 0, nullptr))
        {
            CloseHandle(thread);
        }
        break;
    case DLL_PROCESS_DETACH:
        break;
    }
    return TRUE;
}

