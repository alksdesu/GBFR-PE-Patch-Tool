<script setup>
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { CharaAttach, CharaDetach,
         CurrencyGetAll, CurrencySetOne,
         PotionGetAll, PotionSetOne,
         CountdownGetStatus, CountdownScan, CountdownSet,
         FaceAccessoryGetStatus, FaceAccessoryScan, FaceAccessorySetHidden,
         InfiniteChallengeGetStatus, InfiniteChallengeSetEnabled,
         MaterialConsumeGetStatus, MaterialConsumeSetEnabled,
         TerminusDropGetStatus, TerminusDropScan, TerminusDropSetEnabled,
         UnlockAllTrophyGetStatus, UnlockAllTrophyScan, UnlockAllTrophySetEnabled,
         OtherSkinPurpleRuneGetStatus, OtherSkinPurpleRuneSetEnabled,
         MonsterEnhanceSetPatchValueEnabled,
         CaveList, CaveSetEnabled, CaveSetFloat, CaveSetInt, CaveSetFlag, CaveMeta,
         HighlightedItemRead, HighlightedItemUpdate, HighlightedWeaponRead, HighlightedWeaponUpdate,
         DamageMeterGetStatus, DamageMeterReset,
         DamageOverlaySetEnabled, DamageOverlaySetValue, DamageOverlaySetFontSize,
         GetAppVersion, CheckUpdate, OpenReleasePage } from '../../wailsjs/go/main/App'

const emit = defineEmits(['status'])

const connected = ref(false)
const info = reactive({ pid: 0, moduleBase: 0, manager: 0 })
const loading = ref(false)

const countdownValue = ref('30')
const countdownStatus = reactive({ found: false, address: 0, rva: 0, value1: 0, value2: 0, currentBytes: '' })
const countdownLoading = ref(false)
const faceAccessoryStatus = reactive({ found: false, address: 0, rva: 0, hidden: false, jumpOpcode: '', currentBytes: '' })
const faceAccessoryLoading = ref(false)
const infiniteChallengeStatus = reactive({ rva: 0, enabled: false, currentBytes: '' })
const infiniteChallengeLoading = ref(false)
const materialConsumeStatus = reactive({ rva: 0, enabled: false, currentBytes: '' })
const materialConsumeLoading = ref(false)
const terminusDropStatus = reactive({ found: false, address: 0, rva: 0, enabled: false, currentBytes: '' })
const terminusDropLoading = ref(false)
const unlockAllTrophyStatus = reactive({ found: false, address: 0, rva: 0, enabled: false, currentBytes: '' })
const unlockAllTrophyLoading = ref(false)
const showUnlockAllTrophyConfirm = ref(false)
const otherSkinPurpleRuneStatus = reactive({ rva: 0, enabled: false, jumpOpcode: '', currentBytes: '' })
const otherSkinPurpleRuneLoading = ref(false)
const updateInfo = reactive({ currentVersion: 'v1.5.0', latestVersion: '', hasUpdate: false, releaseUrl: '', body: '', assets: [] })
const updateLoading = ref(false)
const damageMeterStatus = reactive({ connected: false, totalDamage: 0, monsterDamage: 0, crocodileDamage: 0 })
const damageMeterLoading = ref(false)
const currencies = ref([])
const currencyInputs = reactive({})
const currencyLoading = ref(false)
const potions = ref([])
const potionInputs = reactive({})
const potionLoading = ref(false)
const damageOverlayEnabled = ref(false)
const damageOverlayFontSize = ref(Number(localStorage.getItem('gbfrDamageOverlayFontSize') || 48))
const showOutdatedFeatures = false
const characterOrder = ['古兰', '卡塔莉娜', '拉卡姆', '伊欧', '欧根', '萝赛塔', '娜露梅', '菲莉', '夏洛特', '尤达拉哈', '巴萨拉卡', '塞达', '冈达葛萨', '巴恩', '伊德', '圣德芬', '希耶提', '索恩', '贝阿朵丽丝', '尤斯提斯', '玛琪拉菲菈', '伽兰查', '芙劳', '菲迪耶尔']
const characterSet = new Set(characterOrder)
const selectedCharacter = ref('')
let damageMeterTimer = 0

const caveList = ref([])
const caveMeta = ref({})
const caveBusy = reactive({})
const caveParams = reactive({})
const caveLoading = ref(false)

const caveToggles = computed(() => caveList.value.filter((c) => {
  if (characterSet.has(c.group)) return false
  return caveMeta.value[c.id]?.kind !== 'modifier'
}))

const caveModifiers = computed(() => caveList.value.filter((c) => {
  if (characterSet.has(c.group)) return false
  const m = caveMeta.value[c.id]
  return m && m.kind === 'modifier'
}))

const expandedModifier = ref('')
function toggleModifierExpand(id) {
  expandedModifier.value = expandedModifier.value === id ? '' : id
}

const modifierActiveCount = computed(() => caveModifiers.value.filter((c) => c.enabled).length)

const caveToggleGroups = computed(() => {
  const map = new Map()
  caveToggles.value.forEach((item) => {
    if (!map.has(item.group)) map.set(item.group, [])
    map.get(item.group).push(item)
  })
  return Array.from(map.entries()).map(([group, items]) => ({ group, items }))
})

const caveActiveCount = computed(() => caveList.value.filter((c) => c.enabled).length)

const characterList = computed(() => {
  const stat = new Map()
  const ensure = (name) => {
    if (!stat.has(name)) stat.set(name, { name, caveCount: 0, activeCount: 0 })
    return stat.get(name)
  }
  caveList.value.forEach((c) => {
    if (!characterSet.has(c.group)) return
    const s = ensure(c.group)
    s.caveCount++
    if (c.enabled) s.activeCount++
  })
  return characterOrder.filter((n) => stat.has(n)).map((n) => stat.get(n))
})

const characterActiveTotal = computed(() => characterList.value.reduce((sum, c) => sum + c.activeCount, 0))

const selectedCharacterPatches = computed(() =>
  caveList.value.filter((c) => c.group === selectedCharacter.value && caveMeta.value[c.id]?.kind !== 'modifier'))

const selectedCharacterModifiers = computed(() =>
  caveList.value.filter((c) => c.group === selectedCharacter.value && caveMeta.value[c.id]?.kind === 'modifier'))

function toggleCharacter(name) {
  selectedCharacter.value = selectedCharacter.value === name ? '' : name
}

function getMonsterEnhanceMultiplier(id) {
  const saved = window.gbfrMonsterEnhanceMultipliers || {}
  const value = parseFloat(saved[id] || '1')
  return isNaN(value) || value <= 0 || value > 9999 ? 1 : value
}

GetAppVersion().then(v => { updateInfo.currentVersion = v }).catch(() => {})

function connect() {
  loading.value = true
  CharaAttach()
    .then((res) => {
      connected.value = true
      Object.assign(info, res)
      if (showOutdatedFeatures) {
        loadCountdownStatus()
        loadFaceAccessoryStatus()
      }
      if (showOutdatedFeatures) loadInfiniteChallengeStatus()
      loadMaterialConsumeStatus()
      loadCaves()
      if (showOutdatedFeatures) {
        loadTerminusDropStatus()
        loadUnlockAllTrophyStatus()
        loadOtherSkinPurpleRuneStatus()
      }
      loadCurrencyValues()
      loadPotionValues()
      if (showOutdatedFeatures) startDamageMeterTimer()
    })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { loading.value = false })
}

function disconnect() {
  CharaDetach()
    .then(() => {
      connected.value = false
      stopDamageMeterTimer()
      Object.assign(info, { pid: 0, moduleBase: 0, manager: 0 })
      Object.assign(countdownStatus, { found: false, address: 0, rva: 0, value1: 0, value2: 0, currentBytes: '' })
      Object.assign(faceAccessoryStatus, { found: false, address: 0, rva: 0, hidden: false, jumpOpcode: '', currentBytes: '' })
      Object.assign(infiniteChallengeStatus, { rva: 0, enabled: false, currentBytes: '' })
      Object.assign(materialConsumeStatus, { rva: 0, enabled: false, currentBytes: '' })
      caveList.value = []
      Object.keys(caveBusy).forEach((key) => delete caveBusy[key])
      expandedCapture.value = ''
      Object.keys(captureBusy).forEach((key) => delete captureBusy[key])
      itemEditor.captured = false
      weaponEditor.captured = false
      selectedCharacter.value = ''
      Object.assign(terminusDropStatus, { found: false, address: 0, rva: 0, enabled: false, currentBytes: '' })
      Object.assign(unlockAllTrophyStatus, { found: false, address: 0, rva: 0, enabled: false, currentBytes: '' })
      Object.assign(otherSkinPurpleRuneStatus, { rva: 0, enabled: false, jumpOpcode: '', currentBytes: '' })
      Object.assign(damageMeterStatus, { connected: false, totalDamage: 0, monsterDamage: 0, crocodileDamage: 0 })
      currencies.value = []
      Object.keys(currencyInputs).forEach((key) => delete currencyInputs[key])
      potions.value = []
      Object.keys(potionInputs).forEach((key) => delete potionInputs[key])
    })
    .catch((err) => emit('status', String(err), 'error'))
}

function formatHex(value) {
  if (!value) return '—'
  return '0x' + Number(value).toString(16).toUpperCase()
}

function formatFloat(value) {
  if (value === undefined || value === null) return '—'
  return Number(value).toFixed(2)
}

function isCountdownActive() {
  return countdownStatus.found && Math.abs(Number(countdownStatus.value1) - 30) > 0.001
}

function applyCountdownStatus(status) {
  Object.assign(countdownStatus, status || { found: false, address: 0, rva: 0, value1: 0, value2: 0, currentBytes: '' })
  if (status && status.found) countdownValue.value = String(Number(status.value1.toFixed(2)))
}

function loadCountdownStatus() {
  if (!connected.value) return
  countdownLoading.value = true
  CountdownGetStatus()
    .then(applyCountdownStatus)
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { countdownLoading.value = false })
}

function scanCountdown() {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  countdownLoading.value = true
  CountdownScan()
    .then((status) => { applyCountdownStatus(status); emit('status', '倒计时特征定位成功', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { countdownLoading.value = false })
}

function setCountdown() {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  const v = parseFloat(countdownValue.value)
  if (isNaN(v) || v < 0 || v > 9999) { emit('status', '请输入 0 到 9999 之间的数值', 'error'); return }
  countdownLoading.value = true
  CountdownSet(v)
    .then((status) => { applyCountdownStatus(status); emit('status', '倒计时写入成功', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { countdownLoading.value = false })
}

function applyFaceAccessoryStatus(status) {
  Object.assign(faceAccessoryStatus, status || { found: false, address: 0, rva: 0, hidden: false, jumpOpcode: '', currentBytes: '' })
}

function loadFaceAccessoryStatus() {
  if (!connected.value) return
  faceAccessoryLoading.value = true
  FaceAccessoryGetStatus()
    .then(applyFaceAccessoryStatus)
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { faceAccessoryLoading.value = false })
}

function scanFaceAccessory() {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  faceAccessoryLoading.value = true
  FaceAccessoryScan()
    .then((status) => { applyFaceAccessoryStatus(status); emit('status', '脸部符文特征定位成功', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { faceAccessoryLoading.value = false })
}

function setFaceAccessoryHidden(hidden) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  faceAccessoryLoading.value = true
  FaceAccessorySetHidden(hidden)
    .then((status) => { applyFaceAccessoryStatus(status); emit('status', hidden ? '已隐藏脸部符文' : '已恢复脸部符文显示', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { faceAccessoryLoading.value = false })
}

function applyInfiniteChallengeStatus(status) {
  Object.assign(infiniteChallengeStatus, status || { rva: 0, enabled: false, currentBytes: '' })
}

function loadInfiniteChallengeStatus() {
  if (!connected.value) return
  infiniteChallengeLoading.value = true
  InfiniteChallengeGetStatus()
    .then(applyInfiniteChallengeStatus)
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { infiniteChallengeLoading.value = false })
}

function setInfiniteChallengeEnabled(enabled) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  infiniteChallengeLoading.value = true
  InfiniteChallengeSetEnabled(enabled)
    .then((status) => { applyInfiniteChallengeStatus(status); emit('status', enabled ? '已开启无限挑战' : '已恢复挑战次数递增', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { infiniteChallengeLoading.value = false })
}

function loadCaves() {
  if (!connected.value) return
  caveLoading.value = true
  Promise.all([CaveMeta(), CaveList()])
    .then(([meta, list]) => {
      const map = {}
      ;(Array.isArray(meta) ? meta : []).forEach((m) => { map[m.id] = m })
      caveMeta.value = map
      caveList.value = Array.isArray(list) ? list : []
      caveList.value.forEach((c) => {
        const m = map[c.id]
        if (!m) return
        ;(m.floats || []).forEach((f, i) => {
          const key = `${c.id}|flt|${i}`
          if (caveParams[key] === undefined) caveParams[key] = String(f.default)
        })
        ;(m.ints || []).forEach((f, i) => {
          const key = `${c.id}|int|${i}`
          if (caveParams[key] === undefined) caveParams[key] = String(f.default)
        })
      })
    })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { caveLoading.value = false })
}

function toggleCave(item) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  if (caveBusy[item.id]) return
  const next = !item.enabled
  caveBusy[item.id] = true
  CaveSetEnabled(item.id, next)
    .then((state) => {
      const index = caveList.value.findIndex((entry) => entry.id === state.id)
      if (index >= 0) caveList.value.splice(index, 1, state)
      if (next) applyCaveParams(item.id)
      emit('status', `${state.name}已${next ? '开启' : '关闭'}`, 'success')
    })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { delete caveBusy[item.id] })
}

function applyCaveParams(id) {
  const m = caveMeta.value[id]
  if (!m) return
  ;(m.floats || []).forEach((f, i) => {
    const v = parseFloat(caveParams[`${id}|flt|${i}`])
    if (!isNaN(v)) CaveSetFloat(id, f.sym, f.extra, v).catch(() => {})
  })
  ;(m.ints || []).forEach((f, i) => {
    const v = parseInt(caveParams[`${id}|int|${i}`], 10)
    if (!isNaN(v)) CaveSetInt(id, f.sym, f.extra, v).catch(() => {})
  })
}

function setCaveFloat(id, sym, extra, key) {
  const v = parseFloat(caveParams[key])
  if (isNaN(v)) { emit('status', '请输入有效数值', 'error'); return }
  CaveSetFloat(id, sym, extra, v)
    .then(() => emit('status', '数值已写入', 'success'))
    .catch((err) => emit('status', String(err), 'error'))
}

function setCaveInt(id, sym, extra, key) {
  const v = parseInt(caveParams[key], 10)
  if (isNaN(v)) { emit('status', '请输入有效整数', 'error'); return }
  CaveSetInt(id, sym, extra, v)
    .then(() => emit('status', '数值已写入', 'success'))
    .catch((err) => emit('status', String(err), 'error'))
}

function toggleCaveFlag(id, sym, byte, key) {
  const next = !caveParams[key]
  caveParams[key] = next
  CaveSetFlag(id, sym, byte, next)
    .then(() => emit('status', next ? '子功能已开启' : '子功能已关闭', 'success'))
    .catch((err) => { caveParams[key] = !next; emit('status', String(err), 'error') })
}

const captureEditors = { highlighted_item: HighlightedItemRead, highlighted_weapon: HighlightedWeaponRead }
const expandedCapture = ref('')
const captureBusy = reactive({})
const itemEditor = reactive({ captured: false, id: 0, amount: '', state: '' })
const weaponEditor = reactive({
  captured: false, hasImbued: false,
  id: 0, skin: '', level: '', hp: '', attack: '', stunPower: '', critChance: '',
  exp: '', uncapLevel: '', mirage: '', awakenedLevel: '',
  traits: [], imbuedStone: '', imbuedTraits: [],
})

function hasCaptureEditor(id) { return Object.prototype.hasOwnProperty.call(captureEditors, id) }

function toggleCaptureExpand(id) {
  const next = expandedCapture.value === id ? '' : id
  expandedCapture.value = next
  if (next) readCapture(id)
}

function readCapture(id) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  captureBusy[id] = true
  captureEditors[id]()
    .then((data) => {
      if (id === 'highlighted_item') fillItemEditor(data)
      else fillWeaponEditor(data)
      emit('status', '已读取选中对象', 'success')
    })
    .catch((err) => {
      if (id === 'highlighted_item') itemEditor.captured = false
      else weaponEditor.captured = false
      emit('status', String(err), 'error')
    })
    .finally(() => { delete captureBusy[id] })
}

function fillItemEditor(d) {
  itemEditor.captured = true
  itemEditor.id = d.id >>> 0
  itemEditor.amount = String(d.amount >>> 0)
  itemEditor.state = String(d.state >>> 0)
}

function fillWeaponEditor(d) {
  weaponEditor.captured = true
  weaponEditor.hasImbued = !!d.hasImbued
  weaponEditor.id = (d.id >>> 0).toString(16).toUpperCase()
  weaponEditor.skin = (d.skin >>> 0).toString(16).toUpperCase()
  weaponEditor.level = String(d.level >>> 0)
  weaponEditor.hp = String(d.hp >>> 0)
  weaponEditor.attack = String(d.attack >>> 0)
  weaponEditor.stunPower = String(d.stunPower >>> 0)
  weaponEditor.critChance = String(d.critChance >>> 0)
  weaponEditor.exp = String(d.exp >>> 0)
  weaponEditor.uncapLevel = String(d.uncapLevel >>> 0)
  weaponEditor.mirage = String(d.mirage >>> 0)
  weaponEditor.awakenedLevel = String(d.awakenedLevel >>> 0)
  weaponEditor.traits = (d.traits || []).map((t) => ({ id: (t.id >>> 0).toString(16).toUpperCase(), level: String(t.level >>> 0) }))
  weaponEditor.imbuedStone = (d.imbuedStone >>> 0).toString(16).toUpperCase()
  weaponEditor.imbuedTraits = (d.imbuedTraits || []).map((t) => ({ id: (t.id >>> 0).toString(16).toUpperCase(), level: String(t.level >>> 0) }))
}

function u32(v) { const n = parseInt(v, 10); return isNaN(n) || n < 0 ? 0 : n >>> 0 }
function hex32(v) { const n = parseInt(String(v).trim(), 16); return isNaN(n) || n < 0 ? 0 : n >>> 0 }

function writeItem() {
  captureBusy.highlighted_item = true
  HighlightedItemUpdate(u32(itemEditor.amount), u32(itemEditor.state))
    .then((data) => { fillItemEditor(data); emit('status', '物品已写入', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { delete captureBusy.highlighted_item })
}

function writeWeapon() {
  captureBusy.highlighted_weapon = true
  HighlightedWeaponUpdate({
    skin: hex32(weaponEditor.skin),
    level: u32(weaponEditor.level),
    hp: u32(weaponEditor.hp),
    attack: u32(weaponEditor.attack),
    stunPower: u32(weaponEditor.stunPower),
    critChance: u32(weaponEditor.critChance),
    exp: u32(weaponEditor.exp),
    uncapLevel: u32(weaponEditor.uncapLevel),
    mirage: u32(weaponEditor.mirage),
    awakenedLevel: u32(weaponEditor.awakenedLevel),
    traits: weaponEditor.traits.map((t) => ({ id: hex32(t.id), level: u32(t.level) })),
    imbuedStone: hex32(weaponEditor.imbuedStone),
    imbuedTraits: weaponEditor.imbuedTraits.map((t) => ({ id: hex32(t.id), level: u32(t.level) })),
  })
    .then((data) => { fillWeaponEditor(data); emit('status', '武器已写入', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { delete captureBusy.highlighted_weapon })
}

function applyMaterialConsumeStatus(status) {
  Object.assign(materialConsumeStatus, status || { rva: 0, enabled: false, currentBytes: '' })
}

function loadMaterialConsumeStatus() {
  if (!connected.value) return
  materialConsumeLoading.value = true
  MaterialConsumeGetStatus()
    .then(applyMaterialConsumeStatus)
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { materialConsumeLoading.value = false })
}

function setMaterialConsumeEnabled(enabled) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  materialConsumeLoading.value = true
  MaterialConsumeSetEnabled(enabled)
    .then((status) => { applyMaterialConsumeStatus(status); emit('status', enabled ? '已开启升级/强化不材料消耗' : '已恢复升级/强化材料变化', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { materialConsumeLoading.value = false })
}

function applyTerminusDropStatus(status) {
  Object.assign(terminusDropStatus, status || { found: false, address: 0, rva: 0, enabled: false, currentBytes: '' })
}

function loadTerminusDropStatus() {
  if (!connected.value) return
  terminusDropLoading.value = true
  TerminusDropGetStatus()
    .then(applyTerminusDropStatus)
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { terminusDropLoading.value = false })
}

function scanTerminusDrop() {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  terminusDropLoading.value = true
  TerminusDropScan()
    .then((status) => { applyTerminusDropStatus(status); emit('status', '巴武掉落特征定位成功', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { terminusDropLoading.value = false })
}

function setTerminusDropEnabled(enabled) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  terminusDropLoading.value = true
  TerminusDropSetEnabled(enabled)
    .then((status) => { applyTerminusDropStatus(status); emit('status', enabled ? '已开启巴武掉落 100%' : '已恢复巴武默认掉率', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { terminusDropLoading.value = false })
}

function applyUnlockAllTrophyStatus(status) {
  Object.assign(unlockAllTrophyStatus, status || { found: false, address: 0, rva: 0, enabled: false, currentBytes: '' })
}

function loadUnlockAllTrophyStatus() {
  if (!connected.value) return
  unlockAllTrophyLoading.value = true
  UnlockAllTrophyGetStatus()
    .then(applyUnlockAllTrophyStatus)
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { unlockAllTrophyLoading.value = false })
}

function scanUnlockAllTrophy() {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  unlockAllTrophyLoading.value = true
  UnlockAllTrophyScan()
    .then((status) => { applyUnlockAllTrophyStatus(status); emit('status', '全称号解锁特征定位成功', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { unlockAllTrophyLoading.value = false })
}

function setUnlockAllTrophyEnabled(enabled) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  if (enabled) { showUnlockAllTrophyConfirm.value = true; return }
  applyUnlockAllTrophyEnabled(false)
}

function confirmUnlockAllTrophy() {
  showUnlockAllTrophyConfirm.value = false
  applyUnlockAllTrophyEnabled(true)
}

function applyUnlockAllTrophyEnabled(enabled) {
  unlockAllTrophyLoading.value = true
  UnlockAllTrophySetEnabled(enabled)
    .then((status) => { applyUnlockAllTrophyStatus(status); emit('status', enabled ? '已开启游戏内全称号解锁' : '已恢复称号默认判断', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { unlockAllTrophyLoading.value = false })
}

function applyOtherSkinPurpleRuneStatus(status) {
  Object.assign(otherSkinPurpleRuneStatus, status || { rva: 0, enabled: false, jumpOpcode: '', currentBytes: '' })
}

function loadOtherSkinPurpleRuneStatus() {
  if (!connected.value) return
  otherSkinPurpleRuneLoading.value = true
  OtherSkinPurpleRuneGetStatus()
    .then(applyOtherSkinPurpleRuneStatus)
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { otherSkinPurpleRuneLoading.value = false })
}

function setOtherSkinPurpleRuneEnabled(enabled) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  otherSkinPurpleRuneLoading.value = true
  OtherSkinPurpleRuneSetEnabled(enabled)
    .then((status) => { applyOtherSkinPurpleRuneStatus(status); emit('status', enabled ? '已开启其他皮肤紫色符文显示' : '已恢复其他皮肤紫色符文判断', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { otherSkinPurpleRuneLoading.value = false })
}

function formatDamage(value) {
  return Number(value || 0).toLocaleString()
}

function formatInt(value) {
  return Number(value || 0).toLocaleString()
}

function applyCurrencyValues(items) {
  currencies.value = Array.isArray(items) ? items : []
  currencies.value.forEach((item) => {
    currencyInputs[item.id] = String(item.value)
  })
}

function loadCurrencyValues() {
  if (!connected.value) return
  currencyLoading.value = true
  CurrencyGetAll()
    .then(applyCurrencyValues)
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { currencyLoading.value = false })
}

function setCurrency(item) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  const value = Number(currencyInputs[item.id])
  if (!Number.isInteger(value) || value < 0 || value > 2147483647) { emit('status', '请输入 0 到 2147483647 之间的整数', 'error'); return }
  currencyLoading.value = true
  CurrencySetOne(item.id, value)
    .then((updated) => {
      const index = currencies.value.findIndex((entry) => entry.id === updated.id)
      if (index >= 0) currencies.value.splice(index, 1, updated)
      currencyInputs[updated.id] = String(updated.value)
      emit('status', `${updated.name}写入成功`, 'success')
    })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { currencyLoading.value = false })
}

function formatOffsets(offsets) {
  return (offsets || []).map(formatHex).join(' + ')
}

function applyPotionValues(items) {
  potions.value = Array.isArray(items) ? items : []
  potions.value.forEach((item) => {
    potionInputs[item.id] = String(item.value)
  })
}

function loadPotionValues() {
  if (!connected.value) return
  potionLoading.value = true
  PotionGetAll()
    .then(applyPotionValues)
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { potionLoading.value = false })
}

function setPotion(item) {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  const value = Number(potionInputs[item.id])
  if (!Number.isInteger(value) || value < 0 || value > 2147483647) { emit('status', '请输入 0 到 2147483647 之间的整数', 'error'); return }
  potionLoading.value = true
  PotionSetOne(item.id, value)
    .then((updated) => {
      const index = potions.value.findIndex((entry) => entry.id === updated.id)
      if (index >= 0) potions.value.splice(index, 1, updated)
      potionInputs[updated.id] = String(updated.value)
      emit('status', `${updated.name}写入成功`, 'success')
    })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { potionLoading.value = false })
}

function applyDamageMeterStatus(status) {
  Object.assign(damageMeterStatus, {
    connected: !!(status && status.connected),
    totalDamage: Number((status && status.totalDamage) || 0),
    monsterDamage: Number((status && status.monsterDamage) || 0),
    crocodileDamage: Number((status && status.crocodileDamage) || 0),
  })
  if (damageOverlayEnabled.value) DamageOverlaySetValue(displayDamage()).catch(() => {})
}

function displayDamage() {
  return Math.round(damageMeterStatus.monsterDamage * getMonsterEnhanceMultiplier('monster_hp') + damageMeterStatus.crocodileDamage * getMonsterEnhanceMultiplier('crocodile_damage'))
}

function startDamageMeterTimer() {
  stopDamageMeterTimer()
  loadDamageMeterStatus()
  damageMeterTimer = window.setInterval(() => loadDamageMeterStatus(true), 500)
}

function stopDamageMeterTimer() {
  if (!damageMeterTimer) return
  window.clearInterval(damageMeterTimer)
  damageMeterTimer = 0
}

function loadDamageMeterStatus(silent = false) {
  if (!connected.value) return
  if (!silent) damageMeterLoading.value = true
  DamageMeterGetStatus()
    .then(applyDamageMeterStatus)
    .catch((err) => { if (!silent) emit('status', String(err), 'error') })
    .finally(() => { if (!silent) damageMeterLoading.value = false })
}

function enableDamageMeter() {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  damageMeterLoading.value = true
  MonsterEnhanceSetPatchValueEnabled('monster_hp', true, getMonsterEnhanceMultiplier('monster_hp'))
    .then(() => MonsterEnhanceSetPatchValueEnabled('crocodile_damage', true, getMonsterEnhanceMultiplier('crocodile_damage')))
    .then(() => DamageMeterGetStatus())
    .then((status) => {
      applyDamageMeterStatus(status)
      emit('status', '伤害记录已开启，已自动开启怪物多倍血和鳄鱼多倍血', 'success')
    })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { damageMeterLoading.value = false })
}

function resetDamageMeter() {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  damageMeterLoading.value = true
  DamageMeterReset()
    .then((status) => { applyDamageMeterStatus(status); emit('status', '团队伤害已清零', 'success') })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { damageMeterLoading.value = false })
}

function clampOverlayFontSize(value) {
  return Math.min(120, Math.max(18, Number(value) || 48))
}

function setDamageOverlayFontSize(value) {
  damageOverlayFontSize.value = clampOverlayFontSize(value)
  localStorage.setItem('gbfrDamageOverlayFontSize', String(damageOverlayFontSize.value))
  DamageOverlaySetFontSize(damageOverlayFontSize.value).catch(() => {})
}

function enableDamageOverlay() {
  if (!connected.value) { emit('status', '请先连接游戏进程', 'error'); return }
  DamageOverlaySetFontSize(damageOverlayFontSize.value)
    .then(() => DamageOverlaySetValue(displayDamage()))
    .then(() => DamageOverlaySetEnabled(true))
    .then(() => {
      damageOverlayEnabled.value = true
      startDamageMeterTimer()
      emit('status', '伤害悬浮窗已开启', 'success')
    })
    .catch((err) => emit('status', String(err), 'error'))
}

function disableDamageOverlay() {
  DamageOverlaySetEnabled(false).catch(() => {})
  damageOverlayEnabled.value = false
  emit('status', '伤害悬浮窗已关闭', 'success')
}

function toggleDamageOverlay() {
  if (damageOverlayEnabled.value) disableDamageOverlay()
  else enableDamageOverlay()
}

function checkUpdate() {
  updateLoading.value = true
  CheckUpdate()
    .then((info) => {
      Object.assign(updateInfo, info)
      emit('status', info.hasUpdate ? `发现新版本 ${info.latestVersion}` : '当前已是最新版本', info.hasUpdate ? 'success' : 'success')
    })
    .catch((err) => emit('status', String(err), 'error'))
    .finally(() => { updateLoading.value = false })
}

function openReleasePage() {
  OpenReleasePage(updateInfo.releaseUrl || '')
    .catch((err) => emit('status', String(err), 'error'))
}

onBeforeUnmount(() => {
  stopDamageMeterTimer()
  if (damageOverlayEnabled.value) disableDamageOverlay()
})

</script>

<template>
  <div class="root">
    <div class="section">
      <div class="header">
        <span class="title">杂项（隐藏了不可用功能/新增了一些新功能）</span>
        <span class="info-dot" title="这些功能会修改游戏运行时内存，不写入存档；重启游戏或切换版本后需要重新连接并设置。">!</span>
        <span class="hint">需游戏运行中使用 · 重启游戏后需重新设置</span>
      </div>
      <div class="connect-row">
        <button v-if="!connected" class="btn-connect" @click="connect" :disabled="loading">
          {{ loading ? '连接中...' : '连接游戏进程' }}
        </button>
        <button v-else class="btn-disconnect" @click="disconnect">断开连接</button>
        <span v-if="connected" class="pid">PID: {{ info.pid }}</span>
      </div>

      <div class="memory-card">
        <div class="memory-header">
          <span class="memory-title">检查更新</span>
          <span class="memory-hint">当前版本 {{ updateInfo.currentVersion }}</span>
        </div>
        <div class="memory-info">
          <span>最新版本: {{ updateInfo.latestVersion || '未检查' }}</span>
          <span v-if="updateInfo.hasUpdate" class="update-new">有新版本</span>
          <span v-else-if="updateInfo.latestVersion">已是最新</span>
        </div>
        <div v-if="updateInfo.body" class="update-body">{{ updateInfo.body }}</div>
        <div class="memory-row">
          <button class="btn-batch" @click="checkUpdate" :disabled="updateLoading">{{ updateLoading ? '检查中...' : '检查更新' }}</button>
          <button class="btn-refresh" @click="openReleasePage">打开 Release 页面</button>
        </div>
      </div>

      <template v-if="connected">
        <div class="memory-card" :class="{ active: currencies.length }">
          <div class="memory-header">
            <span class="memory-title">货币编辑</span>
            <span class="memory-hint">稳定指针读取/写入 int32</span>
          </div>
          <div class="currency-grid">
            <div v-for="item in currencies" :key="item.id" class="currency-row">
              <div class="currency-name">{{ item.name }}</div>
              <div class="currency-meta">{{ formatInt(item.value) }} · {{ formatHex(item.rva) }} + {{ formatHex(item.offset) }}</div>
              <input v-model="currencyInputs[item.id]" type="number" min="0" max="2147483647" step="1" class="batch-input currency-input" />
              <button class="btn-batch" @click="setCurrency(item)" :disabled="currencyLoading">写入</button>
            </div>
          </div>
          <div class="memory-row">
            <button class="btn-refresh" @click="loadCurrencyValues" :disabled="currencyLoading">刷新货币</button>
          </div>
        </div>

        <div class="memory-card" :class="{ active: potions.length }">
          <div class="memory-header">
            <span class="memory-title">药神（进入副本后点刷新看到药水数量正常后设置即可）</span>
            <span class="memory-hint">稳定指针链读取/写入 int32</span>
          </div>
          <div class="currency-grid">
            <div v-for="item in potions" :key="item.id" class="currency-row">
              <div class="currency-name">{{ item.name }}</div>
              <div class="currency-meta">{{ formatInt(item.value) }} · {{ formatHex(item.rva) }} + {{ formatOffsets(item.offsets) }}</div>
              <input v-model="potionInputs[item.id]" type="number" min="0" max="2147483647" step="1" class="batch-input currency-input" />
              <button class="btn-batch" @click="setPotion(item)" :disabled="potionLoading">写入</button>
            </div>
          </div>
          <div class="memory-row">
            <button class="btn-refresh" @click="loadPotionValues" :disabled="potionLoading">刷新药水</button>
          </div>
        </div>

        <div v-if="showOutdatedFeatures" class="memory-card" :class="{ active: damageMeterStatus.connected && damageMeterStatus.totalDamage > 0 }">
          <div class="memory-header">
            <span class="memory-title">团队伤害记录</span>
            <span class="memory-hint">依赖怪物增强中倍率血量，本功能自动开启默认1倍</span>
          </div>
          <div class="memory-info damage-meter-info">
            <span>状态: {{ damageMeterStatus.connected ? '记录中' : '等待共享内存' }}</span>
            <span>原始扣血点会按怪物增强倍率折算显示</span>
          </div>
          <div class="damage-meter-value">{{ formatDamage(displayDamage()) }}</div>
          <div class="damage-meter-raw">原始: {{ formatDamage(damageMeterStatus.totalDamage) }}</div>
          <div class="memory-row">
            <button class="btn-batch" @click="enableDamageMeter" :disabled="damageMeterLoading">开启记录</button>
            <button class="btn-refresh" @click="toggleDamageOverlay" :disabled="damageMeterLoading || !damageMeterStatus.connected">{{ damageOverlayEnabled ? '关闭悬浮窗' : '开启悬浮窗' }}</button>
            <button class="btn-refresh" @click="loadDamageMeterStatus" :disabled="damageMeterLoading">刷新</button>
            <button class="btn-refresh" @click="resetDamageMeter" :disabled="damageMeterLoading">清零</button>
            <button class="btn-sort" @click="setDamageOverlayFontSize(damageOverlayFontSize - 4)" :disabled="!damageOverlayEnabled">字号 -</button>
            <button class="btn-sort" @click="setDamageOverlayFontSize(damageOverlayFontSize + 4)" :disabled="!damageOverlayEnabled">字号 +</button>
          </div>
        </div>

        <div v-if="showOutdatedFeatures" class="memory-card" :class="{ active: isCountdownActive() }">
          <div class="memory-header">
            <span class="memory-title">任务结算倒计时/零帧开箱</span>
            <span class="info-dot" title="任务结算倒计时超过30s会导致进度条消失，但计时正常；零帧开箱需设置为0s。">!</span>
            <span class="memory-hint">AOB 定位后动态写入两个 float 值</span>
          </div>
          <div class="memory-info">
            <span>RVA: {{ formatHex(countdownStatus.rva) }}</span>
            <span>状态: {{ isCountdownActive() ? '开启' : '默认' }}</span>
            <span>当前: {{ formatFloat(countdownStatus.value1) }} / {{ formatFloat(countdownStatus.value2) }}</span>
          </div>
          <div class="memory-row">
            <input v-model="countdownValue" type="number" min="0" max="9999" step="0.1" class="batch-input countdown-input" placeholder="秒数" />
            <button class="btn-batch" @click="setCountdown" :disabled="countdownLoading">设置倒计时</button>
            <button class="btn-refresh" @click="loadCountdownStatus" :disabled="countdownLoading">刷新</button>
            <button class="btn-sort" @click="scanCountdown" :disabled="countdownLoading">重新扫描</button>
          </div>
          <div class="memory-bytes">{{ countdownStatus.currentBytes || '未定位' }}</div>
        </div>

        <div v-if="showOutdatedFeatures" class="memory-card" :class="{ active: faceAccessoryStatus.hidden }">
          <div class="memory-header">
            <span class="memory-title">脸部符文显示(紫色皮肤包)</span>
            <span class="memory-hint">切换 JE/JNE 控制渲染判断</span>
          </div>
          <div class="memory-info">
            <span>RVA: {{ formatHex(faceAccessoryStatus.rva) }}</span>
            <span>状态: {{ faceAccessoryStatus.hidden ? '隐藏' : '显示' }}</span>
            <span>跳转: {{ faceAccessoryStatus.jumpOpcode || '—' }}</span>
          </div>
          <div class="memory-row">
            <button class="btn-batch" @click="setFaceAccessoryHidden(true)" :disabled="faceAccessoryLoading || faceAccessoryStatus.hidden">隐藏脸部符文</button>
            <button class="btn-refresh" @click="setFaceAccessoryHidden(false)" :disabled="faceAccessoryLoading || !faceAccessoryStatus.hidden">恢复符文显示</button>
            <button class="btn-refresh" @click="loadFaceAccessoryStatus" :disabled="faceAccessoryLoading">刷新</button>
            <button class="btn-sort" @click="scanFaceAccessory" :disabled="faceAccessoryLoading">重新扫描</button>
          </div>
          <div class="memory-bytes">{{ faceAccessoryStatus.currentBytes || '未定位' }}</div>
        </div>

        <div v-if="showOutdatedFeatures" class="memory-card" :class="{ active: infiniteChallengeStatus.enabled }">
          <div class="memory-header">
            <span class="memory-title">无限挑战</span>
            <span class="memory-hint">NOP 挑战次数递增</span>
          </div>
          <div class="memory-info">
            <span>RVA: {{ formatHex(infiniteChallengeStatus.rva) }}</span>
            <span>状态: {{ infiniteChallengeStatus.enabled ? '开启' : '默认' }}</span>
          </div>
          <div class="memory-row">
            <button class="btn-batch" @click="setInfiniteChallengeEnabled(true)" :disabled="infiniteChallengeLoading || infiniteChallengeStatus.enabled">开启无限挑战</button>
            <button class="btn-refresh" @click="setInfiniteChallengeEnabled(false)" :disabled="infiniteChallengeLoading || !infiniteChallengeStatus.enabled">恢复默认</button>
            <button class="btn-refresh" @click="loadInfiniteChallengeStatus" :disabled="infiniteChallengeLoading">刷新</button>
          </div>
          <div class="memory-bytes">{{ infiniteChallengeStatus.currentBytes || '未读取' }}</div>
        </div>

        <div class="memory-card" :class="{ active: materialConsumeStatus.enabled }">
          <div class="memory-header">
            <span class="memory-title">升级/强化/练成不材料消耗</span>
            <span class="info-dot" title="开启后材料数量不会减少；同一指令也会阻止材料增加。">!</span>
            <span class="memory-hint">NOP add [r14+04],esi</span>
          </div>
          <div class="memory-info">
            <span>RVA: {{ formatHex(materialConsumeStatus.rva) }}</span>
            <span>状态: {{ materialConsumeStatus.enabled ? '开启' : '默认' }}</span>
          </div>
          <div class="memory-row">
            <button class="btn-batch" @click="setMaterialConsumeEnabled(true)" :disabled="materialConsumeLoading || materialConsumeStatus.enabled">开启不消耗</button>
            <button class="btn-refresh" @click="setMaterialConsumeEnabled(false)" :disabled="materialConsumeLoading || !materialConsumeStatus.enabled">恢复默认</button>
            <button class="btn-refresh" @click="loadMaterialConsumeStatus" :disabled="materialConsumeLoading">刷新</button>
          </div>
          <div class="memory-bytes">{{ materialConsumeStatus.currentBytes || '未读取' }}</div>
        </div>

        <div class="memory-card" :class="{ active: caveActiveCount > 0 }">
          <div class="memory-header">
            <span class="memory-title">高级功能（代码注入 · 开关类）</span>
            <span class="info-dot" title="需分配代码洞并 Hook 游戏函数，含玩家/物品指针捕获、自动连招等。断开连接自动还原。">!</span>
            <span class="memory-hint">已开启 {{ caveActiveCount }} / {{ caveList.length }}</span>
          </div>
          <div v-if="!caveToggles.length" class="combat-empty">{{ caveLoading ? '加载中...' : '无可用功能' }}</div>
          <div v-for="grp in caveToggleGroups" :key="grp.group" class="combat-group">
            <div class="combat-group-title">{{ grp.group }}</div>
            <div class="combat-toggle-grid">
              <button
                v-for="item in grp.items"
                :key="item.id"
                class="combat-toggle"
                :class="{ on: item.enabled }"
                :disabled="caveBusy[item.id]"
                @click="toggleCave(item)"
              >
                <span class="combat-dot"></span>
                <span class="combat-label">{{ item.name }}</span>
              </button>
            </div>
            <div v-for="item in grp.items.filter((c) => hasCaptureEditor(c.id) && c.enabled)" :key="item.id + '-edit'" class="capture-edit" :class="{ open: expandedCapture === item.id }">
              <div class="capture-edit-head" @click="toggleCaptureExpand(item.id)">
                <span class="capture-edit-title">{{ item.name }} · 编辑</span>
                <button class="btn-batch" :disabled="captureBusy[item.id]" @click.stop="readCapture(item.id)">读取</button>
                <span class="mod-caret" :class="{ open: expandedCapture === item.id }">▸</span>
              </div>

              <div v-if="expandedCapture === item.id" class="capture-edit-body">
                <template v-if="item.id === 'highlighted_item'">
                  <div v-if="!itemEditor.captured" class="mod-hint">在游戏内选中物品后点「读取」</div>
                  <template v-else>
                    <div class="capture-field-grid">
                      <label class="capture-field"><span>ID</span><input :value="itemEditor.id.toString(16).toUpperCase()" class="batch-input" readonly /></label>
                      <label class="capture-field"><span>数量</span><input v-model="itemEditor.amount" type="number" min="0" class="batch-input" /></label>
                      <label class="capture-field"><span>状态</span><input v-model="itemEditor.state" type="number" min="0" class="batch-input" /></label>
                    </div>
                    <button class="btn-batch capture-save" :disabled="captureBusy.highlighted_item" @click="writeItem">写入物品</button>
                  </template>
                </template>

                <template v-else>
                  <div v-if="!weaponEditor.captured" class="mod-hint">在游戏内选中武器后点「读取」</div>
                  <template v-else>
                    <div class="capture-note">写入后会调用游戏保存函数持久化全部字段；HP/攻击/暴击等派生属性以角色实际装备时重算为准。</div>
                    <div class="capture-field-grid">
                      <label class="capture-field"><span>ID(hex)</span><input :value="weaponEditor.id" class="batch-input" readonly /></label>
                      <label class="capture-field"><span>皮肤(hex)</span><input v-model="weaponEditor.skin" class="batch-input" /></label>
                      <label class="capture-field"><span>等级</span><input v-model="weaponEditor.level" type="number" min="0" class="batch-input" /></label>
                      <label class="capture-field"><span>突破等级</span><input v-model="weaponEditor.uncapLevel" type="number" min="0" class="batch-input" /></label>
                      <label class="capture-field"><span>觉醒等级</span><input v-model="weaponEditor.awakenedLevel" type="number" min="0" class="batch-input" /></label>
                      <label class="capture-field"><span>经验</span><input v-model="weaponEditor.exp" type="number" min="0" class="batch-input" /></label>
                      <label class="capture-field"><span>HP</span><input v-model="weaponEditor.hp" type="number" min="0" class="batch-input" /></label>
                      <label class="capture-field"><span>攻击</span><input v-model="weaponEditor.attack" type="number" min="0" class="batch-input" /></label>
                      <label class="capture-field"><span>眩晕值</span><input v-model="weaponEditor.stunPower" type="number" min="0" class="batch-input" /></label>
                      <label class="capture-field"><span>暴击率</span><input v-model="weaponEditor.critChance" type="number" min="0" class="batch-input" /></label>
                      <label class="capture-field"><span>幻影弹</span><input v-model="weaponEditor.mirage" type="number" min="0" class="batch-input" /></label>
                    </div>

                    <div class="capture-subtitle">词条</div>
                    <div v-for="(t, i) in weaponEditor.traits" :key="'wt'+i" class="capture-trait-row">
                      <span class="capture-trait-idx">[{{ i+1 }}]</span>
                      <input v-model="t.id" placeholder="Trait ID (hex)" class="batch-input capture-trait-id" />
                      <input v-model="t.level" type="number" min="0" placeholder="Lv" class="batch-input capture-trait-lv" />
                    </div>

                    <template v-if="weaponEditor.hasImbued">
                      <div class="capture-subtitle">附魔（附魔祝福石）</div>
                      <label class="capture-field capture-field-wide"><span>祝福石(hex)</span><input v-model="weaponEditor.imbuedStone" class="batch-input" /></label>
                      <div v-for="(t, i) in weaponEditor.imbuedTraits" :key="'it'+i" class="capture-trait-row">
                        <span class="capture-trait-idx">[{{ i+1 }}]</span>
                        <input v-model="t.id" placeholder="Trait ID (hex)" class="batch-input capture-trait-id" />
                        <input v-model="t.level" type="number" min="0" placeholder="Lv" class="batch-input capture-trait-lv" />
                      </div>
                    </template>

                    <button class="btn-batch capture-save" :disabled="captureBusy.highlighted_weapon" @click="writeWeapon">写入武器</button>
                  </template>
                </template>
              </div>
            </div>
          </div>
          <div class="memory-row">
            <button class="btn-refresh" @click="loadCaves" :disabled="caveLoading">刷新状态</button>
          </div>
        </div>

        <div class="memory-card" :class="{ active: modifierActiveCount > 0 }" v-if="caveModifiers.length">
          <div class="memory-header">
            <span class="memory-title">参数化修改</span>
            <span class="info-dot" title="带倍率/等级/子开关的通用修改。点击一行展开其参数；有参数的功能需先开启再展开设置。">!</span>
            <span class="memory-hint">已开启 {{ modifierActiveCount }} / {{ caveModifiers.length }}</span>
          </div>
          <div class="mod-list">
            <div v-for="item in caveModifiers" :key="item.id" class="mod-item" :class="{ active: item.enabled, open: expandedModifier === item.id }">
              <div class="mod-row" @click="toggleModifierExpand(item.id)">
                <span class="mod-dot" :class="{ on: item.enabled }"></span>
                <span class="mod-name">{{ item.name }}</span>
                <span class="mod-group">{{ item.group }}</span>
                <button
                  class="mod-toggle-btn"
                  :class="{ on: item.enabled }"
                  :disabled="caveBusy[item.id]"
                  @click.stop="toggleCave(item)"
                >{{ item.enabled ? '关闭' : '开启' }}</button>
                <span class="mod-caret" :class="{ open: expandedModifier === item.id }">▸</span>
              </div>
              <div v-if="expandedModifier === item.id" class="mod-params">
                <div v-if="!item.enabled" class="mod-hint">先点「开启」再设置参数</div>
                <template v-else>
                  <div v-for="(f, i) in (caveMeta[item.id]?.floats || [])" :key="'f'+i" class="cave-param-row">
                    <span class="cave-param-label">{{ f.label }}</span>
                    <input v-model="caveParams[item.id + '|flt|' + i]" type="number" step="0.1" class="batch-input cave-input" />
                    <button class="btn-batch" @click="setCaveFloat(item.id, f.sym, f.extra, item.id + '|flt|' + i)">写入</button>
                  </div>
                  <div v-for="(f, i) in (caveMeta[item.id]?.ints || [])" :key="'i'+i" class="cave-param-row">
                    <span class="cave-param-label">{{ f.label }}</span>
                    <input v-model="caveParams[item.id + '|int|' + i]" type="number" step="1" class="batch-input cave-input" />
                    <button class="btn-batch" @click="setCaveInt(item.id, f.sym, f.extra, item.id + '|int|' + i)">写入</button>
                  </div>
                  <div v-if="(caveMeta[item.id]?.flags || []).length" class="combat-toggle-grid cave-flag-grid">
                    <button
                      v-for="(fl, i) in caveMeta[item.id].flags"
                      :key="'fl'+i"
                      class="combat-toggle"
                      :class="{ on: caveParams[item.id + '|flg|' + i] }"
                      @click="toggleCaveFlag(item.id, fl.sym, fl.byte, item.id + '|flg|' + i)"
                    >
                      <span class="combat-dot"></span>
                      <span class="combat-label">{{ fl.label }}</span>
                    </button>
                  </div>
                  <div v-if="!(caveMeta[item.id]?.floats || []).length && !(caveMeta[item.id]?.ints || []).length && !(caveMeta[item.id]?.flags || []).length" class="mod-hint">此功能无额外参数</div>
                </template>
              </div>
            </div>
          </div>
        </div>

        <div class="memory-card" :class="{ active: characterActiveTotal > 0 }">
          <div class="memory-header">
            <span class="memory-title">角色专属修改</span>
            <span class="info-dot" title="按角色收纳的专属机制。点击角色展开其全部开关与参数修改。">!</span>
            <span class="memory-hint">已开启 {{ characterActiveTotal }}</span>
          </div>
          <div v-if="!characterList.length" class="combat-empty">{{ caveLoading ? '加载中...' : '无可用角色功能' }}</div>
          <div class="char-grid">
            <button
              v-for="c in characterList"
              :key="c.name"
              class="char-chip"
              :class="{ selected: selectedCharacter === c.name, lit: c.activeCount > 0 }"
              @click="toggleCharacter(c.name)"
            >
              <span class="char-name">{{ c.name }}</span>
              <span class="char-badge" v-if="c.activeCount > 0">{{ c.activeCount }}</span>
            </button>
          </div>

          <div v-if="selectedCharacter" class="char-panel">
            <div class="char-panel-title">{{ selectedCharacter }} · 专属修改</div>
            <div v-if="selectedCharacterPatches.length" class="combat-toggle-grid">
              <button
                v-for="item in selectedCharacterPatches"
                :key="item.id"
                class="combat-toggle"
                :class="{ on: item.enabled }"
                :disabled="caveBusy[item.id]"
                @click="toggleCave(item)"
              >
                <span class="combat-dot"></span>
                <span class="combat-label">{{ item.name }}</span>
              </button>
            </div>

            <div v-for="item in selectedCharacterModifiers" :key="item.id" class="char-modifier" :class="{ active: item.enabled }">
              <div class="char-modifier-head">
                <span class="char-modifier-name">{{ item.name }}</span>
                <button v-if="!item.enabled" class="btn-batch" @click="toggleCave(item)" :disabled="caveBusy[item.id]">开启</button>
                <button v-else class="btn-refresh" @click="toggleCave(item)" :disabled="caveBusy[item.id]">关闭</button>
              </div>
              <template v-if="item.enabled">
                <div v-for="(f, i) in (caveMeta[item.id]?.floats || [])" :key="'f'+i" class="cave-param-row">
                  <span class="cave-param-label">{{ f.label }}</span>
                  <input v-model="caveParams[item.id + '|flt|' + i]" type="number" step="0.1" class="batch-input cave-input" />
                  <button class="btn-batch" @click="setCaveFloat(item.id, f.sym, f.extra, item.id + '|flt|' + i)">写入</button>
                </div>
                <div v-for="(f, i) in (caveMeta[item.id]?.ints || [])" :key="'i'+i" class="cave-param-row">
                  <span class="cave-param-label">{{ f.label }}</span>
                  <input v-model="caveParams[item.id + '|int|' + i]" type="number" step="1" class="batch-input cave-input" />
                  <button class="btn-batch" @click="setCaveInt(item.id, f.sym, f.extra, item.id + '|int|' + i)">写入</button>
                </div>
                <div v-if="(caveMeta[item.id]?.flags || []).length" class="combat-toggle-grid cave-flag-grid">
                  <button
                    v-for="(fl, i) in caveMeta[item.id].flags"
                    :key="'fl'+i"
                    class="combat-toggle"
                    :class="{ on: caveParams[item.id + '|flg|' + i] }"
                    @click="toggleCaveFlag(item.id, fl.sym, fl.byte, item.id + '|flg|' + i)"
                  >
                    <span class="combat-dot"></span>
                    <span class="combat-label">{{ fl.label }}</span>
                  </button>
                </div>
              </template>
            </div>
          </div>
        </div>

        <div class="memory-card" :class="{ active: terminusDropStatus.enabled }">
          <div class="memory-header">
            <span class="memory-title">巴武掉落 100%</span>
            <span class="info-dot" title="仅让原型巴哈姆特任务的巴武 lot 不再被 80% 排除；仍保留未拥有、角色已解锁等游戏原始检查。">!</span>
            <span class="memory-hint">AOB 定位后 NOP 巴武 lot 排除跳转</span>
          </div>
          <div class="memory-info">
            <span>RVA: {{ formatHex(terminusDropStatus.rva) }}</span>
            <span>状态: {{ terminusDropStatus.enabled ? '开启' : '默认' }}</span>
          </div>
          <div class="memory-row">
            <button class="btn-batch" @click="setTerminusDropEnabled(true)" :disabled="terminusDropLoading || terminusDropStatus.enabled">开启巴武 100%</button>
            <button class="btn-refresh" @click="setTerminusDropEnabled(false)" :disabled="terminusDropLoading || !terminusDropStatus.enabled">恢复默认</button>
            <button class="btn-refresh" @click="loadTerminusDropStatus" :disabled="terminusDropLoading">刷新</button>
            <button class="btn-sort" @click="scanTerminusDrop" :disabled="terminusDropLoading">重新扫描</button>
          </div>
          <div class="memory-bytes">{{ terminusDropStatus.currentBytes || '未定位' }}</div>
        </div>

        <div v-if="showOutdatedFeatures" class="memory-card" :class="{ active: unlockAllTrophyStatus.enabled }">
          <div class="memory-header">
            <span class="memory-title">游戏内全称号解锁</span>
            <span class="memory-hint">AOB 定位后切换 SETNE/SETNO</span>
          </div>
          <div class="memory-info">
            <span>RVA: {{ formatHex(unlockAllTrophyStatus.rva) }}</span>
            <span>状态: {{ unlockAllTrophyStatus.enabled ? '开启' : '默认' }}</span>
          </div>
          <div class="memory-row">
            <button class="btn-batch" @click="setUnlockAllTrophyEnabled(true)" :disabled="unlockAllTrophyLoading || unlockAllTrophyStatus.enabled">开启全称号</button>
            <button class="btn-refresh" @click="setUnlockAllTrophyEnabled(false)" :disabled="unlockAllTrophyLoading || !unlockAllTrophyStatus.enabled">恢复默认</button>
            <button class="btn-refresh" @click="loadUnlockAllTrophyStatus" :disabled="unlockAllTrophyLoading">刷新</button>
            <button class="btn-sort" @click="scanUnlockAllTrophy" :disabled="unlockAllTrophyLoading">重新扫描</button>
          </div>
          <div class="memory-bytes">{{ unlockAllTrophyStatus.currentBytes || '未定位' }}</div>
        </div>

        <div v-if="showOutdatedFeatures" class="memory-card" :class="{ active: otherSkinPurpleRuneStatus.enabled }">
          <div class="memory-header">
            <span class="memory-title">在其他皮肤显示紫色符文</span>
            <span class="memory-hint">固定 RVA 切换 JNE/JE</span>
          </div>
          <div class="memory-info">
            <span>RVA: {{ formatHex(otherSkinPurpleRuneStatus.rva) }}</span>
            <span>状态: {{ otherSkinPurpleRuneStatus.enabled ? '开启' : '默认' }}</span>
            <span>跳转: {{ otherSkinPurpleRuneStatus.jumpOpcode || '—' }}</span>
          </div>
          <div class="memory-row">
            <button class="btn-batch" @click="setOtherSkinPurpleRuneEnabled(true)" :disabled="otherSkinPurpleRuneLoading || otherSkinPurpleRuneStatus.enabled">开启显示</button>
            <button class="btn-refresh" @click="setOtherSkinPurpleRuneEnabled(false)" :disabled="otherSkinPurpleRuneLoading || !otherSkinPurpleRuneStatus.enabled">恢复默认</button>
            <button class="btn-refresh" @click="loadOtherSkinPurpleRuneStatus" :disabled="otherSkinPurpleRuneLoading">刷新</button>
          </div>
          <div class="memory-bytes">{{ otherSkinPurpleRuneStatus.currentBytes || '未读取' }}</div>
        </div>

      </template>
      <div v-else class="empty">请先连接游戏进程</div>
    </div>
    <div v-if="showUnlockAllTrophyConfirm" class="confirm-overlay" @click.self="showUnlockAllTrophyConfirm = false">
      <div class="confirm-dialog">
        <div class="confirm-title">确认开启游戏内全称号解锁</div>
        <div class="confirm-body">目前存档时机尚不明确，可以领取任务奖励、佩戴选定称号、选择佩戴界面有多个“未设置”是正常现象</div>
        <div class="confirm-actions">
          <button class="btn-refresh" @click="showUnlockAllTrophyConfirm = false">取消</button>
          <button class="btn-warn" @click="confirmUnlockAllTrophy" :disabled="unlockAllTrophyLoading">确认开启</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.root { display:flex; flex-direction:column; gap:10px; width:100%; max-width:720px; margin:0 auto; padding-bottom:40px; }
.section {
  border-radius:16px; padding:16px 18px;
  background:linear-gradient(135deg, rgba(56,189,248,0.12) 0%, rgba(103,232,249,0.06) 100%);
  border:1px solid rgba(103,232,249,0.15);
  display:flex; flex-direction:column; gap:10px;
}
.header { display:flex; align-items:center; justify-content:space-between; gap:8px; }
.title { font-size:0.88rem; font-weight:600; color:rgba(255,255,255,0.65); letter-spacing:1px; }
.info-dot { display:inline-flex; align-items:center; justify-content:center; width:15px; height:15px; border-radius:50%; border:1px solid rgba(103,232,249,0.35); color:#67e8f9; background:rgba(103,232,249,0.08); font-size:0.68rem; font-weight:700; cursor:help; flex-shrink:0; }
.hint { font-size:0.68rem; color:rgba(255,255,255,0.25); margin-left:auto; }
.connect-row { display:flex; align-items:center; gap:10px; }
.btn-connect {
  padding:8px 18px; border-radius:8px; border:1px solid rgba(34,197,94,0.4);
  background:rgba(34,197,94,0.12); color:#4ade80; font-size:0.82rem; font-weight:600; cursor:pointer;
  transition:background 0.2s,transform 0.15s;
}
.btn-connect:not(:disabled):hover { background:rgba(34,197,94,0.22); transform:scale(1.02); }
.btn-connect:disabled { opacity:0.5; cursor:not-allowed; }
.btn-disconnect {
  padding:8px 18px; border-radius:8px; border:1px solid rgba(239,68,68,0.4);
  background:rgba(239,68,68,0.12); color:#f87171; font-size:0.82rem; font-weight:600; cursor:pointer;
  transition:background 0.2s;
}
.btn-disconnect:hover { background:rgba(239,68,68,0.22); }
.pid { font-size:0.72rem; color:rgba(255,255,255,0.35); font-family:'Courier New',monospace; }
.memory-card {
  position:relative; overflow:hidden; z-index:0;
  border-radius:12px; padding:12px;
  background:rgba(255,255,255,0.045); border:1px solid rgba(165,180,252,0.16);
  box-shadow:0 10px 26px rgba(0,0,0,0.18);
  display:flex; flex-direction:column; gap:8px;
  transition:border-color 0.3s, box-shadow 0.3s, transform 0.3s;
}
.memory-card::after {
  content:""; position:absolute; inset:0; z-index:-1; border-radius:12px;
  background:#abd373; transform:translateY(calc(-100% - 2px));
  transition:transform 0.5s ease;
}
.memory-card.active { border-color:rgba(171,211,115,0.55); box-shadow:0 14px 34px rgba(171,211,115,0.18); }
.memory-card.active::after { transform:translateY(0); }
.memory-card.active .memory-title { color:#1f2937; }
.memory-card.active .memory-hint,
.memory-card.active .memory-info,
.memory-card.active .memory-bytes { color:rgba(31,41,55,0.72); }
.memory-card.active .info-dot { border-color:rgba(31,41,55,0.28); color:#1f2937; background:rgba(31,41,55,0.08); }
.memory-card.active .btn-batch { border-color:rgba(31,41,55,0.22); background:rgba(31,41,55,0.12); color:#1f2937; }
.memory-card.active .btn-refresh,
.memory-card.active .btn-sort { border-color:rgba(31,41,55,0.16); background:rgba(255,255,255,0.18); color:rgba(31,41,55,0.72); }
.memory-card.active .batch-input { border-color:rgba(31,41,55,0.22); background:rgba(255,255,255,0.22); color:#1f2937; }
.memory-header, .memory-info, .memory-row { display:flex; align-items:center; gap:8px; flex-wrap:wrap; }
.memory-header { justify-content:flex-start; }
.memory-header .memory-hint { margin-left:auto; }
.memory-title { font-size:0.8rem; font-weight:600; color:rgba(255,255,255,0.62); }
.memory-hint, .memory-info { font-size:0.68rem; color:rgba(255,255,255,0.32); }
.memory-bytes { font-size:0.66rem; color:rgba(255,255,255,0.24); font-family:'Courier New',monospace; word-break:break-all; }
.damage-meter-info { justify-content:space-between; }
.damage-meter-value { font-size:1.8rem; font-weight:700; color:#67e8f9; line-height:1; }
.damage-meter-raw { margin-top:-4px; font-size:0.72rem; color:rgba(255,255,255,0.28); }
.memory-card.active .damage-meter-value { color:#1f2937; }
.memory-card.active .damage-meter-raw { color:rgba(31,41,55,0.56); }
.currency-grid { display:flex; flex-direction:column; gap:8px; }
.currency-row { display:grid; grid-template-columns:90px 1fr 120px auto; align-items:center; gap:8px; }
.currency-name { font-size:0.78rem; font-weight:600; color:rgba(255,255,255,0.62); }
.currency-meta { font-size:0.66rem; color:rgba(255,255,255,0.28); font-family:'Courier New',monospace; }
.currency-input { width:120px; }
.memory-card.active .currency-name { color:#1f2937; }
.memory-card.active .currency-meta { color:rgba(31,41,55,0.56); }
.update-new { color:#4ade80; }
.update-body { max-height:86px; overflow-y:auto; padding:8px 10px; border-radius:8px; background:rgba(255,255,255,0.03); color:rgba(255,255,255,0.36); font-size:0.7rem; line-height:1.45; white-space:pre-wrap; scrollbar-width:thin; scrollbar-color:rgba(255,255,255,0.12) transparent; }
.batch-input {
  width:80px; padding:6px 10px; border-radius:6px; border:1px solid rgba(255,255,255,0.15);
  background:rgba(255,255,255,0.07); color:#fff; font-size:0.82rem; outline:none;
}
.countdown-input { width:96px; }
.batch-input:focus { border-color:rgba(103,232,249,0.5); }
.batch-input::-webkit-outer-spin-button, .batch-input::-webkit-inner-spin-button { -webkit-appearance:none; margin:0; }
.btn-batch {
  padding:6px 14px; border-radius:6px; border:1px solid rgba(165,180,252,0.3);
  background:rgba(165,180,252,0.1); color:#a5b4fc; font-size:0.78rem; font-weight:600; cursor:pointer;
  transition:background 0.2s; white-space:nowrap;
}
.btn-batch:not(:disabled):hover { background:rgba(165,180,252,0.2); }
.btn-batch:disabled { opacity:0.4; cursor:not-allowed; }
.btn-refresh, .btn-sort {
  padding:6px 14px; border-radius:6px; border:1px solid rgba(255,255,255,0.12);
  background:rgba(255,255,255,0.05); color:rgba(255,255,255,0.5); font-size:0.78rem; font-weight:600; cursor:pointer;
  transition:background 0.2s;
}
.btn-refresh:hover, .btn-sort:hover { background:rgba(255,255,255,0.1); color:rgba(255,255,255,0.7); }
.btn-refresh:disabled, .btn-sort:disabled { opacity:0.4; cursor:not-allowed; }
.empty { font-size:0.78rem; color:rgba(255,255,255,0.3); text-align:center; padding:12px 0; }
.od-select {
  padding:6px 10px; border-radius:6px; border:1px solid rgba(255,255,255,0.15);
  background:rgba(255,255,255,0.07); color:#fff; font-size:0.8rem; outline:none; cursor:pointer;
}
.od-select:focus { border-color:rgba(103,232,249,0.5); }
.od-select option { background:#1a1a2e; color:#fff; }
.od-indicator {
  font-size:0.72rem; padding:4px 10px; border-radius:6px; text-align:center;
  background:rgba(255,255,255,0.05); color:rgba(255,255,255,0.35);
  transition:all 0.3s;
}
.od-mode-active { background:rgba(250,204,21,0.15); color:#facc15; border:1px solid rgba(250,204,21,0.25); }
.od-burst-active { background:rgba(239,68,68,0.15); color:#ef4444; border:1px solid rgba(239,68,68,0.25); animation:od-burst-pulse 1s infinite alternate; }
@keyframes od-burst-pulse { from { opacity:0.7; } to { opacity:1; } }
.burst-timer { color:#facc15; font-weight:600; font-family:'Courier New',monospace; }
.confirm-overlay { position:fixed; inset:0; z-index:20; display:flex; align-items:center; justify-content:center; padding:20px; background:rgba(0,0,0,0.48); }
.confirm-dialog { width:min(420px, 100%); border-radius:12px; padding:16px; background:linear-gradient(135deg, rgba(251,191,36,0.22) 0%, rgba(239,68,68,0.16) 100%); border:1px solid rgba(251,191,36,0.34); box-shadow:0 12px 40px rgba(0,0,0,0.42); display:flex; flex-direction:column; gap:12px; }
.confirm-title { font-size:0.9rem; font-weight:700; color:#facc15; }
.confirm-body { font-size:0.78rem; line-height:1.65; color:rgba(255,255,255,0.72); }
.confirm-actions { display:flex; justify-content:flex-end; gap:8px; flex-wrap:wrap; }
.btn-warn { padding:6px 14px; border-radius:6px; border:1px solid rgba(251,191,36,0.45); background:rgba(251,191,36,0.16); color:#facc15; font-size:0.78rem; font-weight:600; cursor:pointer; transition:background 0.2s; white-space:nowrap; }
.btn-warn:not(:disabled):hover { background:rgba(251,191,36,0.26); }
.btn-warn:disabled { opacity:0.4; cursor:not-allowed; }
.combat-empty { font-size:0.74rem; color:rgba(255,255,255,0.3); padding:6px 0; }
.combat-group { display:flex; flex-direction:column; gap:6px; }
.combat-group-title {
  font-size:0.7rem; font-weight:700; letter-spacing:0.5px;
  color:rgba(103,232,249,0.7); padding:2px 0;
}
.combat-toggle-grid { display:grid; grid-template-columns:repeat(auto-fill, minmax(150px, 1fr)); gap:6px; }
.combat-toggle {
  display:flex; align-items:center; gap:8px; text-align:left;
  padding:7px 10px; border-radius:8px; cursor:pointer;
  border:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.05);
  transition:background 0.2s, border-color 0.2s;
}
.combat-toggle:not(:disabled):hover { background:rgba(255,255,255,0.1); }
.combat-toggle:disabled { opacity:0.5; cursor:progress; }
.combat-dot {
  width:9px; height:9px; border-radius:50%; flex-shrink:0;
  background:rgba(255,255,255,0.22); border:1px solid rgba(255,255,255,0.28);
  transition:background 0.2s, border-color 0.2s, box-shadow 0.2s;
}
.combat-label { font-size:0.76rem; font-weight:600; color:rgba(255,255,255,0.72); line-height:1.2; }
.combat-toggle.on { border-color:rgba(74,222,128,0.5); background:rgba(74,222,128,0.14); }
.combat-toggle.on .combat-dot { background:#4ade80; border-color:#4ade80; box-shadow:0 0 6px rgba(74,222,128,0.6); }
.combat-toggle.on .combat-label { color:#4ade80; }
.combat-toggle.warn { border-color:rgba(251,191,36,0.5); }
.combat-toggle.warn .combat-dot { background:#facc15; border-color:#facc15; }
.memory-card.active .combat-group-title { color:rgba(31,41,55,0.68); }
.memory-card.active .combat-toggle { border-color:rgba(31,41,55,0.18); background:rgba(255,255,255,0.28); }
.memory-card.active .combat-toggle:not(:disabled):hover { background:rgba(255,255,255,0.42); }
.memory-card.active .combat-label { color:rgba(31,41,55,0.78); }
.memory-card.active .combat-dot { background:rgba(31,41,55,0.2); border-color:rgba(31,41,55,0.3); }
.memory-card.active .combat-toggle.on { border-color:rgba(21,128,61,0.55); background:rgba(21,128,61,0.18); }
.memory-card.active .combat-toggle.on .combat-dot { background:#15803d; border-color:#15803d; box-shadow:0 0 6px rgba(21,128,61,0.5); }
.memory-card.active .combat-toggle.on .combat-label { color:#14532d; }
.cave-param-row { display:flex; align-items:center; gap:8px; }
.cave-param-label { font-size:0.76rem; font-weight:600; color:rgba(255,255,255,0.62); min-width:120px; }
.cave-input { width:110px; }
.cave-flag-grid { margin-top:2px; }
.memory-card.active .cave-param-label { color:rgba(31,41,55,0.78); }
.char-grid { display:grid; grid-template-columns:repeat(auto-fill, minmax(88px, 1fr)); gap:6px; }
.char-chip {
  position:relative; display:flex; align-items:center; justify-content:center;
  padding:8px 6px; border-radius:8px; cursor:pointer;
  border:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.05);
  transition:background 0.2s, border-color 0.2s;
}
.char-chip:hover { background:rgba(255,255,255,0.1); }
.char-name { font-size:0.78rem; font-weight:600; color:rgba(255,255,255,0.7); }
.char-chip.lit { border-color:rgba(74,222,128,0.4); }
.char-chip.lit .char-name { color:#4ade80; }
.char-chip.selected { border-color:rgba(103,232,249,0.55); background:rgba(103,232,249,0.14); }
.char-chip.selected .char-name { color:#67e8f9; }
.char-badge {
  position:absolute; top:-5px; right:-5px; min-width:15px; height:15px; padding:0 4px;
  display:flex; align-items:center; justify-content:center;
  border-radius:8px; background:#4ade80; color:#1f2937;
  font-size:0.6rem; font-weight:700; line-height:1;
}
.char-panel {
  margin-top:2px; padding:12px; border-radius:8px;
  background:rgba(103,232,249,0.05); border:1px solid rgba(103,232,249,0.14);
  display:flex; flex-direction:column; gap:10px;
}
.char-panel-title { font-size:0.74rem; font-weight:700; letter-spacing:0.5px; color:#67e8f9; }
.char-modifier {
  border-radius:8px; padding:9px 10px;
  border:1px solid rgba(255,255,255,0.1); background:rgba(255,255,255,0.03);
  display:flex; flex-direction:column; gap:8px;
}
.char-modifier.active { border-color:rgba(74,222,128,0.35); background:rgba(74,222,128,0.06); }
.char-modifier-head { display:flex; align-items:center; justify-content:space-between; gap:8px; }
.char-modifier-name { font-size:0.78rem; font-weight:600; color:rgba(255,255,255,0.68); }
.char-modifier.active .char-modifier-name { color:#4ade80; }
.memory-card.active .char-name { color:rgba(31,41,55,0.72); }
.memory-card.active .char-chip { border-color:rgba(31,41,55,0.18); background:rgba(255,255,255,0.28); }
.memory-card.active .char-chip.selected { border-color:rgba(21,128,61,0.5); background:rgba(21,128,61,0.16); }
.memory-card.active .char-chip.selected .char-name { color:#14532d; }
.mod-list { display:flex; flex-direction:column; gap:5px; }
.mod-item { border-radius:8px; border:1px solid rgba(255,255,255,0.09); background:rgba(255,255,255,0.03); overflow:hidden; transition:border-color 0.2s; }
.mod-item.active { border-color:rgba(74,222,128,0.32); }
.mod-item.open { border-color:rgba(103,232,249,0.35); }
.mod-row { display:flex; align-items:center; gap:9px; padding:8px 11px; cursor:pointer; }
.mod-row:hover { background:rgba(255,255,255,0.04); }
.mod-dot { width:8px; height:8px; border-radius:50%; flex-shrink:0; background:rgba(255,255,255,0.2); border:1px solid rgba(255,255,255,0.28); transition:background 0.2s, box-shadow 0.2s; }
.mod-dot.on { background:#4ade80; border-color:#4ade80; box-shadow:0 0 6px rgba(74,222,128,0.6); }
.mod-name { font-size:0.8rem; font-weight:600; color:rgba(255,255,255,0.72); }
.mod-item.active .mod-name { color:#4ade80; }
.mod-group { font-size:0.66rem; color:rgba(255,255,255,0.3); margin-left:auto; }
.mod-toggle-btn { padding:4px 12px; border-radius:6px; border:1px solid rgba(165,180,252,0.3); background:rgba(165,180,252,0.1); color:#a5b4fc; font-size:0.74rem; font-weight:600; cursor:pointer; white-space:nowrap; transition:background 0.2s; }
.mod-toggle-btn:not(:disabled):hover { background:rgba(165,180,252,0.2); }
.mod-toggle-btn:disabled { opacity:0.4; cursor:not-allowed; }
.mod-toggle-btn.on { border-color:rgba(248,113,113,0.35); background:rgba(248,113,113,0.1); color:#f87171; }
.mod-caret { font-size:0.7rem; color:rgba(255,255,255,0.3); transition:transform 0.2s; }
.mod-caret.open { transform:rotate(90deg); }
.mod-params { padding:4px 11px 11px 28px; display:flex; flex-direction:column; gap:8px; }
.mod-hint { font-size:0.72rem; color:rgba(255,255,255,0.35); }
.memory-card.active .mod-name { color:rgba(31,41,55,0.78); }
.memory-card.active .mod-group { color:rgba(31,41,55,0.5); }
.memory-card.active .mod-item { border-color:rgba(31,41,55,0.14); background:rgba(255,255,255,0.22); }
.capture-edit { margin-top:8px; border:1px solid rgba(103,232,249,0.2); border-radius:6px; background:rgba(103,232,249,0.05); overflow:hidden; }
.capture-edit-head { display:flex; align-items:center; gap:8px; padding:8px 11px; cursor:pointer; }
.capture-edit-title { font-size:0.76rem; font-weight:600; color:rgba(255,255,255,0.7); }
.capture-edit-head .btn-batch { margin-left:auto; }
.capture-edit-body { padding:4px 11px 11px; display:flex; flex-direction:column; gap:8px; }
.capture-note { font-size:0.7rem; line-height:1.5; color:rgba(103,232,249,0.72); }
.capture-field-grid { display:grid; grid-template-columns:repeat(auto-fill,minmax(150px,1fr)); gap:8px; }
.capture-field { display:flex; flex-direction:column; gap:3px; }
.capture-field.capture-field-wide { grid-column:1 / -1; max-width:260px; }
.capture-field span { font-size:0.68rem; color:rgba(255,255,255,0.4); }
.capture-subtitle { font-size:0.7rem; font-weight:600; color:rgba(255,255,255,0.5); margin-top:4px; }
.capture-trait-row { display:flex; align-items:center; gap:8px; }
.capture-trait-idx { font-size:0.7rem; color:rgba(255,255,255,0.32); font-family:'Courier New',monospace; min-width:28px; }
.capture-trait-id { flex:1; }
.capture-trait-lv { width:72px; }
.capture-save { align-self:flex-start; margin-top:4px; }
.memory-card.active .capture-edit { border-color:rgba(31,41,55,0.18); background:rgba(255,255,255,0.18); }
.memory-card.active .capture-edit-title { color:rgba(31,41,55,0.78); }
.memory-card.active .capture-note { color:rgba(31,41,55,0.6); }
.memory-card.active .capture-field span,
.memory-card.active .capture-subtitle { color:rgba(31,41,55,0.6); }
</style>
