<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { GetLastSavePath, SetLastSavePath } from '../../wailsjs/go/main/App'
import { GetSigilList, GetCompatibleSecondaryTraits, GetAllowedLevels,
         GetPrimaryTraitLevels, GetSecondaryTraitLevels, GetPrimaryTrait,
         GetDefaultSecondaryTrait, LoadSaveFile, GetLoadedSaveInfo,
         GetQueue, AddToQueue, RemoveFromQueue, ClearQueue,
         ApplyQueue, RemoveAllSigils,
         GetExistingSigils, DeleteSelectedSigils,
         SelectSigilInputSave, SelectSigilOutputSave } from '../../wailsjs/go/main/SigilGen'

const emit = defineEmits(['status'])

function showStatus(msg, type) { emit('status', msg, type) }

// ── 状态 ──
const sigils = ref([])
const saveLoaded = ref(false)
const saveInfo = reactive({ path: '', occupiedSigils: 0, maxSlotId: 0 })
const isApplying = ref(false)
const inPlaceEdit = ref(false)
const applyFlash = ref(false)
let applyFlashTimer = 0

// 表单
const selectedSigilID = ref('')
const selectedLevel = ref(0)
const selectedPrimaryLevel = ref(0)
const selectedSecondaryTraitID = ref('')
const selectedSecondaryLevel = ref(0)
const quantity = ref(1)
const outputPath = ref('')

// 下拉选项
const sigilLevels = ref([])
const primaryTraitLevels = ref([])
const secondaryTraits = ref([])
const secondaryTraitLevels = ref([])
const primaryTraitName = ref('')
const supportsSecondary = ref(false)

// 队列
const queue = ref([])

// 已有因子
const existingSigils = ref([])
const selectedForDelete = ref(new Set())
const showExisting = ref(false)
const isDeleting = ref(false)
const loadingExisting = ref(false)

// 搜索
const sigilSearch = ref('')
const secondaryTraitSearch = ref('')
const showSigilDropdown = ref(false)

const filteredSigils = computed(() => {
  if (!sigilSearch.value) return sigils.value
  const q = sigilSearch.value.toLowerCase()
  return sigils.value.filter(s => s.displayName.toLowerCase().includes(q))
})

const filteredSecondaryTraits = computed(() => {
  if (!secondaryTraitSearch.value) return secondaryTraits.value
  const q = secondaryTraitSearch.value.toLowerCase()
  return secondaryTraits.value.filter(t => t.displayName.toLowerCase().includes(q))
})

// ── 加载数据 ──
const dataLoading = ref(true)
const dataError = ref('')
onMounted(async () => {
  try {
    sigils.value = await GetSigilList()
    if (!sigils.value || !sigils.value.length) {
      dataError.value = '因子数据为空'
    }
    const lastPath = await GetLastSavePath()
    if (lastPath) {
      inputPath.value = lastPath
      outputPath.value = defaultOutputPath(lastPath)
    }
  } catch (e) {
    dataError.value = '加载因子数据失败: ' + String(e)
  } finally {
    dataLoading.value = false
  }
})

// ── 存档 ──
const inputPath = ref('')

function defaultOutputPath(path) {
  if (!path) return ''
  if (/\.dat$/i.test(path)) return path.replace(/(\.dat)$/i, '_modified.dat')
  return `${path}_modified.dat`
}

watch(inPlaceEdit, (enabled) => {
  if (enabled) {
    outputPath.value = inputPath.value.trim()
  } else if (outputPath.value.trim() === inputPath.value.trim()) {
    outputPath.value = defaultOutputPath(inputPath.value.trim())
  }
})

async function browseInput() {
  try {
    const path = await SelectSigilInputSave()
    if (!path) return
    inputPath.value = path
    await loadSave()
  } catch (e) { showStatus(String(e), 'error') }
}

async function browseOutput() {
  try {
    const path = await SelectSigilOutputSave(outputPath.value.trim() || defaultOutputPath(inputPath.value.trim()))
    if (path) outputPath.value = path
  } catch (e) { showStatus(String(e), 'error') }
}

async function loadSave() {
  if (!inputPath.value.trim()) { showStatus('请输入存档路径', 'error'); return }
  try {
    const info = await LoadSaveFile(inputPath.value.trim())
    Object.assign(saveInfo, info)
    saveLoaded.value = true
    outputPath.value = inPlaceEdit.value ? info.path : defaultOutputPath(info.path)
    await SetLastSavePath(info.path)
    showExisting.value = true
    await refreshExisting()
    showStatus(`已加载存档: ${info.occupiedSigils} 个因子`, 'success')
  } catch (e) {
    showExisting.value = false
    showStatus(String(e), 'error')
  }
}

async function refreshExisting() {
  loadingExisting.value = true
  try {
    existingSigils.value = await GetExistingSigils()
    selectedForDelete.value = new Set()
  } catch (e) {
    existingSigils.value = []
    showStatus('读取已有因子失败: ' + String(e), 'error')
  } finally {
    loadingExisting.value = false
  }
}

function toggleSelectAll() {
  if (selectedForDelete.value.size === existingSigils.value.length) {
    selectedForDelete.value = new Set()
  } else {
    selectedForDelete.value = new Set(existingSigils.value.map(s => s.gemUnitId))
  }
}

async function deleteSelected() {
  if (selectedForDelete.value.size === 0) {
    showStatus('未选中任何因子', 'error'); return
  }
  if (!outputPath.value.trim()) {
    showStatus('请填写输出路径', 'error'); return
  }
  if (!confirm(`确定要删除 ${selectedForDelete.value.size} 个因子吗？此操作不可撤销。`)) return
  isDeleting.value = true
  try {
    const ids = Array.from(selectedForDelete.value)
    const result = await DeleteSelectedSigils(ids, outputPath.value.trim())
    if (inPlaceEdit.value) {
      await loadSave()
    } else {
      await refreshExisting()
    }
    showStatus(`已删除 ${result.createdCount} 个因子`, 'success')
  } catch (e) {
    showStatus(String(e), 'error')
  } finally {
    isDeleting.value = false
  }
}

// ── 因子选择变化 ──
watch(selectedSigilID, async (id) => {
  if (!id) return
  const sigil = sigils.value.find(s => s.internalId === id)
  if (!sigil) return

  supportsSecondary.value = sigil.supportsSecondaryTrait

  // 加载等级
  try {
    sigilLevels.value = await GetAllowedLevels(id)
    primaryTraitLevels.value = await GetPrimaryTraitLevels(id)
  } catch (e) { showStatus(String(e), 'error'); return }

  // 主特性
  try {
    const pt = await GetPrimaryTrait(id)
    primaryTraitName.value = pt ? pt.displayName : ''
  } catch (e) { primaryTraitName.value = '' }

  // 副特性
  if (sigil.supportsSecondaryTrait) {
    try {
      secondaryTraits.value = await GetCompatibleSecondaryTraits(id)
      const def = await GetDefaultSecondaryTrait(id)
      selectedSecondaryTraitID.value = def ? def.internalId : ''
      secondaryTraitSearch.value = def ? def.displayName : ''
    } catch (e) {
      secondaryTraits.value = []
      selectedSecondaryTraitID.value = ''
      secondaryTraitSearch.value = ''
    }
  } else {
    secondaryTraits.value = []
    selectedSecondaryTraitID.value = ''
    secondaryTraitLevels.value = []
    selectedSecondaryLevel.value = 0
    secondaryTraitSearch.value = ''
  }

  // 默认等级
  selectedLevel.value = sigilLevels.value[0] || 0
  selectedPrimaryLevel.value = primaryTraitLevels.value[0] || 0
})

watch(selectedSecondaryTraitID, async (id) => {
  if (!id || !selectedSigilID.value) {
    secondaryTraitLevels.value = []
    selectedSecondaryLevel.value = 0
    if (!id) secondaryTraitSearch.value = ''
    return
  }
  const trait = secondaryTraits.value.find(t => t.internalId === id)
  if (trait) secondaryTraitSearch.value = trait.displayName
  try {
    secondaryTraitLevels.value = await GetSecondaryTraitLevels(selectedSigilID.value, id)
    selectedSecondaryLevel.value = secondaryTraitLevels.value[0] || 0
  } catch (e) { secondaryTraitLevels.value = [] }
})

// ── 队列操作 ──
async function addToQueue() {
  if (!selectedSigilID.value) { showStatus('请选择因子', 'error'); return }
  try {
    await AddToQueue({
      sigilId: selectedSigilID.value,
      sigilName: '',
      level: selectedLevel.value,
      primaryTraitId: '',
      primaryTraitName: '',
      primaryLevel: selectedPrimaryLevel.value,
      secondaryTraitId: supportsSecondary.value ? selectedSecondaryTraitID.value : '',
      secondaryTraitName: '',
      secondaryLevel: selectedSecondaryLevel.value,
      quantity: quantity.value,
    })
    queue.value = await GetQueue()
    showStatus('已添加到队列', 'success')
  } catch (e) { showStatus(String(e), 'error') }
}

async function removeFromQueue(index) {
  try {
    await RemoveFromQueue(index)
    queue.value = await GetQueue()
  } catch (e) { showStatus(String(e), 'error') }
}

async function clearQueueAll() {
  await ClearQueue()
  queue.value = []
}

function flashApplySuccess() {
  applyFlash.value = false
  clearTimeout(applyFlashTimer)
  requestAnimationFrame(() => {
    applyFlash.value = true
    applyFlashTimer = window.setTimeout(() => { applyFlash.value = false }, 900)
  })
}

async function applyQueueToSave() {
  if (!outputPath.value.trim()) { showStatus('请输入输出路径', 'error'); return }
  isApplying.value = true
  try {
    const result = await ApplyQueue(outputPath.value.trim())
    queue.value = []
    if (inPlaceEdit.value) await loadSave()
    flashApplySuccess()
    showStatus(`已写入 ${result.createdCount} 个因子 (验证 ${result.verifiedCount})`, 'success')
  } catch (e) { showStatus(String(e), 'error') }
  finally { isApplying.value = false }
}

async function removeAll() {
  if (!inputPath.value.trim() || !outputPath.value.trim()) {
    showStatus('请先填写输入和输出路径', 'error'); return
  }
  if (!confirm('这将清除输出存档中的所有因子，确定继续？')) return
  try {
    const result = await RemoveAllSigils(inputPath.value.trim(), outputPath.value.trim())
    if (inPlaceEdit.value) {
      await loadSave()
    }
    showStatus(`已清除 ${result.createdCount} 个因子 (剩余 ${result.verifiedCount})`, 'success')
  } catch (e) { showStatus(String(e), 'error') }
}

function onSigilSelect() {
  const sigil = sigils.value.find(s => s.internalId === selectedSigilID.value)
  if (sigil) {
    sigilSearch.value = sigil.displayName
  }
}

function onSecondaryTraitSelect() {
  const trait = secondaryTraits.value.find(t => t.internalId === selectedSecondaryTraitID.value)
  if (trait) {
    secondaryTraitSearch.value = trait.displayName
  }
}
</script>

<template>
  <div class="sigil-container">
    <!-- 存档选择 -->
    <div class="section">
      <div class="section-title">存档文件</div>
      <div class="input-row">
        <input v-model="inputPath" type="text" class="text-input flex-1"
          placeholder="GBFR 存档文件 (.dat | C:\Users\UserName\AppData\Local\GBFR\Saved\SaveGames\)" />
        <button class="btn-action btn-cyan" @click="browseInput">浏览</button>
        <button class="btn-action btn-green" @click="loadSave">加载</button>
      </div>
      <div v-if="saveLoaded" class="save-info">
        已加载 · {{ saveInfo.occupiedSigils }} 个因子 · 最大槽位 {{ saveInfo.maxSlotId }}
      </div>
    </div>

    <!-- 已有因子 -->
    <div v-if="showExisting" class="section">
      <div class="section-title">
        已有因子 {{ loadingExisting ? '加载中...' : `(${existingSigils.length})` }}
        <div class="existing-actions">
          <button class="btn-link" @click="toggleSelectAll"
            :disabled="loadingExisting">
            {{ selectedForDelete.size === existingSigils.length ? '取消全选' : '全选' }}
          </button>
          <button class="btn-link" @click="refreshExisting" :disabled="loadingExisting">刷新</button>
          <button class="btn-action btn-red btn-sm"
            @click="deleteSelected"
            :disabled="isDeleting || loadingExisting || selectedForDelete.size === 0">
            {{ isDeleting ? '删除中...' : `删除选中 (${selectedForDelete.size})` }}
          </button>
        </div>
      </div>
      <div v-if="loadingExisting" class="loading-hint">正在读取已有因子，数量较多时请耐心等待...</div>
      <div v-else-if="saveInfo.occupiedSigils > 500" class="warning-hint">
        注意：当前存档有 {{ saveInfo.occupiedSigils }} 个因子，目前批量编辑处于测试阶段，不建议使用
      </div>
      <div v-if="!loadingExisting && existingSigils.length === 0" class="empty-hint">暂无已有因子或读取失败</div>
      <div v-else class="existing-table">
        <div class="existing-row existing-header">
          <span class="ex-col-cb"><input type="checkbox" :checked="selectedForDelete.size === existingSigils.length && existingSigils.length > 0" @change="toggleSelectAll" /></span>
          <span class="ex-col-name">因子</span>
          <span class="ex-col-level">等级</span>
          <span class="ex-col-trait">特性</span>
        </div>
        <div v-for="s in existingSigils" :key="s.gemUnitId" class="existing-row">
          <span class="ex-col-cb">
            <input type="checkbox" :checked="selectedForDelete.has(s.gemUnitId)"
              @change="selectedForDelete.has(s.gemUnitId) ? selectedForDelete.delete(s.gemUnitId) : selectedForDelete.add(s.gemUnitId)" />
          </span>
          <span class="ex-col-name">{{ s.sigilName }}</span>
          <span class="ex-col-level">Lv {{ s.level }}</span>
          <span class="ex-col-trait">
            {{ s.primaryTraitName }} Lv {{ s.primaryLevel }}
            <template v-if="s.secondaryTraitName"> / {{ s.secondaryTraitName }} Lv {{ s.secondaryLevel }}</template>
          </span>
        </div>
      </div>
    </div>

    <!-- 因子配置 -->
    <div class="section">
      <div class="section-title">因子配置</div>

      <!-- 因子搜索 -->
      <div class="field">
        <label>因子 {{ dataLoading ? '(加载中...)' : dataError ? '(加载失败)' : '' }}</label>
        <div v-if="dataError" class="data-error">{{ dataError }}</div>
        <input v-model="sigilSearch" type="text" class="text-input"
          placeholder="输入关键词过滤..." @input="showSigilDropdown = true" />
        <select v-model="selectedSigilID" class="select-input sigil-select"
          size="6" @change="onSigilSelect">
          <option value="">— 请先搜索并选择因子 —</option>
          <option v-for="s in filteredSigils" :key="s.internalId" :value="s.internalId">
            {{ s.displayName }}<template v-if="s.supportsSecondaryTrait"> [V+]</template>
          </option>
        </select>
      </div>

      <!-- 因子等级 -->
      <div class="field">
        <label>因子等级</label>
        <div class="readonly-field">Lv {{ selectedLevel || '—' }}</div>
      </div>

      <!-- 主特性 -->
      <div class="field">
        <label>主特性</label>
        <div class="readonly-field">{{ primaryTraitName || '—' }}</div>
      </div>

      <div class="field">
        <label>主特性等级</label>
        <select v-model="selectedPrimaryLevel" class="select-input" :disabled="!primaryTraitLevels.length">
          <option v-for="l in primaryTraitLevels" :key="l" :value="l">Lv {{ l }}</option>
        </select>
      </div>

      <!-- 副特性 -->
      <template v-if="supportsSecondary">
        <div class="field">
          <label>副特性</label>
          <input v-model="secondaryTraitSearch" type="text" class="text-input"
            placeholder="输入关键词过滤副特性..." />
          <select v-model="selectedSecondaryTraitID" class="select-input sigil-select"
            size="6" :disabled="!secondaryTraits.length" @change="onSecondaryTraitSelect">
            <option value="">— 不选择 —</option>
            <option v-for="t in filteredSecondaryTraits" :key="t.internalId" :value="t.internalId">
              {{ t.displayName }}
            </option>
          </select>
        </div>
        <div class="field">
          <label>副特性等级</label>
          <select v-model="selectedSecondaryLevel" class="select-input"
            :disabled="!secondaryTraitLevels.length">
            <option v-for="l in secondaryTraitLevels" :key="l" :value="l">Lv {{ l }}</option>
          </select>
        </div>
      </template>

      <!-- 数量 + 添加 -->
      <div class="input-row">
        <div class="field flex-1">
          <label>数量</label>
          <input v-model.number="quantity" type="number" min="1" max="999" class="text-input" />
        </div>
        <button class="btn-action btn-purple add-btn" @click="addToQueue"
          :disabled="!selectedSigilID">
          添加到队列
        </button>
      </div>
    </div>

    <!-- 队列 -->
    <div class="section">
      <div class="section-title">
        队列 ({{ queue.length }})
        <button v-if="queue.length" class="btn-link" @click="clearQueueAll">清空</button>
      </div>
      <div v-if="!queue.length" class="empty-hint">暂无待写入因子，请先添加</div>
      <div v-else class="queue-list">
        <div v-for="(item, i) in queue" :key="i" class="queue-item">
          <div class="queue-info">
            <span class="queue-name">{{ item.sigilName }}</span>
            <span class="queue-detail">
              Lv {{ item.level }} · {{ item.primaryTraitName }} Lv {{ item.primaryLevel }}
              <template v-if="item.secondaryTraitId">
                / {{ item.secondaryTraitName }} Lv {{ item.secondaryLevel }}
              </template>
              · x{{ item.quantity }}
            </span>
          </div>
          <button class="btn-icon" @click="removeFromQueue(i)" title="移除">✕</button>
        </div>
      </div>
    </div>

    <!-- 输出 + 应用 -->
    <div class="section apply-section" :class="{ 'apply-flash': applyFlash }">
      <div class="section-title">输出</div>
      <div class="input-row">
        <input v-model="outputPath" type="text" class="text-input flex-1"
          :class="{ 'danger-path': inPlaceEdit }" :readonly="inPlaceEdit"
          placeholder="输出存档路径..." />
        <button class="btn-action btn-cyan" @click="browseOutput" :disabled="inPlaceEdit">浏览</button>
        <button class="btn-action btn-cyan" @click="applyQueueToSave"
          :disabled="isApplying || !queue.length">
          {{ isApplying ? '写入中...' : '应用写入' }}
        </button>
      </div>
      <label class="toggle-row">
        <input v-model="inPlaceEdit" type="checkbox" />
        <span>启用原地修改（直接覆盖输入存档）</span>
      </label>
      <div v-if="inPlaceEdit" class="danger-hint">警告：启用后，应用写入将直接覆盖当前输入存档，建议先备份。</div>
    </div>

    <!-- 清除所有 -->
    <div class="section section-danger">
      <div class="section-title">危险操作</div>
      <button class="btn-action btn-red" @click="removeAll"
        :disabled="!inputPath.trim() || !outputPath.trim()">
        清除输出存档中所有因子
      </button>
    </div>
  </div>
</template>

<style scoped>
.sigil-container {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
}

.section {
  border-radius: 12px;
  padding: 14px 16px;
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.06);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.apply-section {
  position: relative;
  overflow: hidden;
  z-index: 0;
  transition: border-color 0.3s, box-shadow 0.3s;
}
.apply-section > * { position: relative; z-index: 1; }
.apply-section::after {
  content: "";
  position: absolute;
  inset: 0;
  z-index: 0;
  border-radius: 12px;
  background: #abd373;
  transform: translateY(calc(-100% - 2px));
  transition: transform 0.5s ease;
}
.apply-section.apply-flash { border-color: rgba(171,211,115,0.55); box-shadow: 0 14px 34px rgba(171,211,115,0.18); }
.apply-section.apply-flash::after { transform: translateY(0); }
.apply-section.apply-flash .section-title,
.apply-section.apply-flash .toggle-row { color: #1f2937; }
.apply-section.apply-flash .text-input { border-color: rgba(31,41,55,0.22); background: rgba(255,255,255,0.22); color: #1f2937; }
.apply-section.apply-flash .btn-cyan { border-color: rgba(31,41,55,0.22); background: rgba(31,41,55,0.12); color: #1f2937; }
.apply-section.apply-flash .danger-hint { border-color: rgba(31,41,55,0.18); background: rgba(255,255,255,0.18); color: rgba(31,41,55,0.78); }

.section-danger {
  border-color: rgba(239,68,68,0.15);
  background: rgba(239,68,68,0.04);
}

.section-title {
  font-size: 0.78rem;
  font-weight: 600;
  color: rgba(255,255,255,0.35);
  letter-spacing: 1px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.field { display: flex; flex-direction: column; gap: 4px; }
.field label {
  font-size: 0.7rem;
  color: rgba(255,255,255,0.3);
}

.text-input, .select-input {
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.12);
  background: rgba(255,255,255,0.06);
  color: #fff;
  font-size: 0.82rem;
  font-family: inherit;
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.select-input option {
  background: transparent;
  color: #fff;
}

.text-input:focus, .select-input:focus {
  border-color: rgba(103,232,249,0.4);
  background: transparent;
}

.select-input {
  cursor: pointer;
  appearance: none;
  background-color: transparent;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='8' height='5'%3E%3Cpath d='M0 0l4 5 4-5z' fill='rgba(255,255,255,0.3)'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 10px center;
  padding-right: 28px;
}

.select-input:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.readonly-field {
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.08);
  background: rgba(255,255,255,0.03);
  color: rgba(255,255,255,0.45);
  font-size: 0.82rem;
}

.input-row {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.flex-1 { flex: 1; }

.btn-action {
  padding: 8px 16px;
  border-radius: 8px;
  border: none;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: transform 0.15s, opacity 0.2s;
}

.btn-action:not(:disabled):hover { transform: scale(1.03); }
.btn-action:disabled { opacity: 0.35; cursor: not-allowed; }

.btn-green {
  background: rgba(34,197,94,0.18);
  color: #4ade80;
  border: 1px solid rgba(34,197,94,0.3);
}
.btn-green:not(:disabled):hover { background: rgba(34,197,94,0.28); }

.btn-purple {
  background: rgba(165,180,252,0.15);
  color: #a5b4fc;
  border: 1px solid rgba(165,180,252,0.3);
}
.btn-purple:not(:disabled):hover { background: rgba(165,180,252,0.25); }

.btn-cyan {
  background: rgba(103,232,249,0.15);
  color: #67e8f9;
  border: 1px solid rgba(103,232,249,0.3);
}
.btn-cyan:not(:disabled):hover { background: rgba(103,232,249,0.25); }

.btn-red {
  background: rgba(239,68,68,0.15);
  color: #f87171;
  border: 1px solid rgba(239,68,68,0.3);
}
.btn-red:not(:disabled):hover { background: rgba(239,68,68,0.25); }

.add-btn { padding-top: 8px; padding-bottom: 8px; align-self: flex-end; }

.btn-link {
  background: none;
  border: none;
  color: rgba(255,255,255,0.3);
  font-size: 0.72rem;
  cursor: pointer;
  padding: 0 4px;
}
.btn-link:hover { color: rgba(239,68,68,0.7); }

.btn-icon {
  background: none;
  border: none;
  color: rgba(255,255,255,0.3);
  cursor: pointer;
  font-size: 0.85rem;
  padding: 2px 6px;
  border-radius: 4px;
  transition: color 0.15s;
}
.btn-icon:hover { color: #f87171; }

.save-info {
  font-size: 0.72rem;
  color: rgba(74,222,128,0.6);
}

.empty-hint {
  font-size: 0.75rem;
  color: rgba(255,255,255,0.2);
  text-align: center;
  padding: 8px 0;
}

.loading-hint {
  font-size: 0.78rem;
  color: #67e8f9;
  text-align: center;
  padding: 12px 0;
}

.warning-hint {
  font-size: 0.72rem;
  color: rgba(251,191,36,0.8);
  text-align: center;
  padding: 8px 12px;
  background: rgba(251,191,36,0.08);
  border: 1px solid rgba(251,191,36,0.15);
  border-radius: 6px;
  line-height: 1.5;
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.78rem;
  color: rgba(255,255,255,0.75);
}

.danger-hint {
  font-size: 0.72rem;
  color: rgba(248,113,113,0.95);
  text-align: center;
  padding: 8px 12px;
  background: rgba(239,68,68,0.08);
  border: 1px solid rgba(239,68,68,0.2);
  border-radius: 6px;
  line-height: 1.5;
}

.danger-path {
  background: rgba(239,68,68,0.14) !important;
  border-color: rgba(239,68,68,0.55) !important;
  color: #fecaca;
}

/* 因子选择列表 */
.sigil-select {
  height: auto;
  min-height: 120px;
  overflow-y: auto;
  cursor: pointer;
  appearance: auto !important;
  background-image: none !important;
  padding-right: 6px;
}
.sigil-select option {
  padding: 5px 8px;
  color: #fff;
  background: transparent;
  font-size: 0.82rem;
}
.sigil-select option:checked {
  background: rgba(103,232,249,0.25);
  color: #67e8f9;
}

/* 暗色滚动条 */
.sigil-select::-webkit-scrollbar {
  width: 6px;
}
.sigil-select::-webkit-scrollbar-track {
  background: rgba(0,0,0,0.2);
  border-radius: 3px;
}
.sigil-select::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,0.15);
  border-radius: 3px;
}
.sigil-select::-webkit-scrollbar-thumb:hover {
  background: rgba(255,255,255,0.25);
}

.data-error {
  font-size: 0.72rem;
  color: #f87171;
  padding: 4px 0;
}

/* 已有因子列表 */
.existing-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}
.btn-sm {
  padding: 4px 10px !important;
  font-size: 0.7rem !important;
}
.existing-table {
  display: flex;
  flex-direction: column;
  gap: 1px;
  background: rgba(255,255,255,0.04);
  border-radius: 8px;
  overflow: hidden;
  max-height: 250px;
  overflow-y: auto;
}
.existing-table::-webkit-scrollbar { width: 5px; }
.existing-table::-webkit-scrollbar-track { background: transparent; }
.existing-table::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.12); border-radius: 3px; }
.existing-row {
  display: flex;
  align-items: center;
  padding: 5px 10px;
  gap: 6px;
  background: rgba(27,38,54,0.6);
  font-size: 0.76rem;
}
.existing-header {
  background: rgba(255,255,255,0.06);
  font-size: 0.7rem;
  color: rgba(255,255,255,0.3);
  font-weight: 600;
  padding: 4px 10px;
}
.existing-row input[type="checkbox"] { accent-color: #67e8f9; cursor: pointer; }
.ex-col-cb { width: 20px; flex-shrink: 0; text-align: center; }
.ex-col-name { flex: 1; color: rgba(255,255,255,0.6); min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ex-col-level { width: 40px; text-align: right; color: rgba(255,255,255,0.35); flex-shrink: 0; }
.ex-col-trait { width: 160px; color: rgba(255,255,255,0.3); font-size: 0.7rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex-shrink: 0; }

/* 队列列表 */
.queue-list { display: flex; flex-direction: column; gap: 6px; }
.queue-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-radius: 8px;
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.06);
  gap: 8px;
}
.queue-info { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.queue-name { font-size: 0.8rem; color: rgba(255,255,255,0.6); font-weight: 600; }
.queue-detail { font-size: 0.7rem; color: rgba(255,255,255,0.3); }
</style>
