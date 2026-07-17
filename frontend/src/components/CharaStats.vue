<script setup>
import { ref, computed, onMounted } from 'vue'
import { FindSaveFiles, GetCharacterStats, UpdateCharacterStat } from '../../wailsjs/go/main/App'

const slots = ref([])
const list = ref([])
const savePath = ref('')
const loading = ref(false)
const sortDesc = ref(false)
const newSave = ref(false)
const editMode = ref(false)
const editingSlot = ref(null)
const editValue = ref(0)
const showEditConfirm = ref(false)
const savingSlot = ref(null)
const error = ref('')

const sorted = computed(() => {
  if (!sortDesc.value) return list.value
  return [...list.value].sort((a, b) => b.count - a.count)
})

async function scanSaves() {
  slots.value = await FindSaveFiles() || []
}

async function load(path) {
  loading.value = true
  savePath.value = path
  error.value = ''
  try {
    list.value = await GetCharacterStats(path, newSave.value) || []
  } catch (err) {
    list.value = []
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

async function refresh() {
  await scanSaves()
  if (savePath.value) await load(savePath.value)
}

function switchVersion(value) {
  if (newSave.value === value) return
  newSave.value = value
  editMode.value = false
  editingSlot.value = null
  if (savePath.value) load(savePath.value)
}

function toggleEditMode() {
  if (editMode.value) {
    editMode.value = false
    editingSlot.value = null
    return
  }
  showEditConfirm.value = true
}

function confirmEditMode() {
  showEditConfirm.value = false
  editMode.value = true
}

function startEdit(character) {
  editingSlot.value = character.slot
  editValue.value = character.count
}

function cancelEdit() {
  editingSlot.value = null
}

async function saveEdit(character) {
  if (!Number.isInteger(editValue.value) || editValue.value < 0) {
    error.value = '请输入有效的非负整数'
    return
  }
  savingSlot.value = character.slot
  error.value = ''
  try {
    await UpdateCharacterStat(savePath.value, newSave.value, character.slot, editValue.value)
    character.count = editValue.value
    editingSlot.value = null
  } catch (err) {
    error.value = String(err || '写入失败')
  } finally {
    savingSlot.value = null
  }
}

onMounted(scanSaves)
</script>

<template>
  <div class="root">
    <div class="section">
      <div class="header">
        <span class="title">角色次数统计</span>
        <span class="hint">显示存档角色任务次数(新-DLC更新后创建的存档/旧-DLC更新前创建并转换过来的存档)</span>
      </div>
      <div class="slots">
        <button v-for="s in slots" :key="s.index" class="slot-btn"
          :class="{ on: savePath === s.path }" @click="load(s.path)">
          {{ s.name }}
        </button>
        <button class="btn-refresh" @click="refresh">刷新</button>
      </div>
      <div class="version-row">
        <span class="version-label">存档版本</span>
        <div class="version-switch">
          <button :class="{ on: !newSave }" @click="switchVersion(false)">旧版转换存档</button>
          <button :class="{ on: newSave }" @click="switchVersion(true)">DLC更新后新建存档</button>
        </div>
        <button v-if="list.length" class="btn-sort" @click="sortDesc = !sortDesc">
          {{ sortDesc ? '恢复原序' : '按次数排序' }}
        </button>
        <button v-if="list.length" class="btn-sort" :class="{ on: editMode }" @click="toggleEditMode">
          {{ editMode ? '退出编辑' : '编辑模式' }}
        </button>
      </div>

      <div v-if="showEditConfirm" class="confirm-backdrop" @click.self="showEditConfirm = false">
        <section class="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="character-edit-confirm-title">
          <h2 id="character-edit-confirm-title">确认进入编辑模式</h2>
          <p>编辑前务必备份你的存档，注意你存档版本是否选择正确，并不能保证编辑结果是一定正确的。</p>
          <div class="confirm-actions">
            <button class="confirm-cancel" @click="showEditConfirm = false">取消</button>
            <button class="confirm-apply" @click="confirmEditMode">确认进入</button>
          </div>
        </section>
      </div>

      <div v-if="loading" class="empty">解析中...</div>
      <div v-else-if="error" class="empty">{{ error }}</div>
      <template v-else-if="list.length">
        <div class="table">
          <div class="row row-head">
            <span class="col-name">角色</span>
            <span class="col-count">次数</span>
          </div>
          <div v-for="c in sorted" :key="c.name" class="row">
            <span class="col-name">{{ c.name }}</span>
            <span v-if="!editMode || editingSlot !== c.slot" class="col-count">{{ c.count }}</span>
            <div v-else class="count-edit">
              <input v-model.number="editValue" type="number" min="0" :disabled="savingSlot === c.slot" @keyup.enter="saveEdit(c)" />
              <button class="edit-action save" @click="saveEdit(c)" :disabled="savingSlot === c.slot">{{ savingSlot === c.slot ? '...' : '保存' }}</button>
              <button class="edit-action" @click="cancelEdit" :disabled="savingSlot === c.slot">取消</button>
            </div>
            <button v-if="editMode && editingSlot !== c.slot" class="edit-action" @click="startEdit(c)">编辑</button>
          </div>
        </div>
      </template>
      <div v-else-if="savePath" class="empty">未找到当前档案角色次数</div>
    </div>
  </div>
</template>

<style scoped>
.root { display:flex; flex-direction:column; gap:10px; width:100%; max-width:720px; margin:0 auto; padding-bottom:40px; }
.section { border-radius:12px; padding:14px 16px; background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.06); display:flex; flex-direction:column; gap:10px; }
.header { display:flex; align-items:center; justify-content:space-between; gap:10px; }
.title { font-size:0.88rem; font-weight:600; color:rgba(255,255,255,0.65); letter-spacing:1px; }
.hint { font-size:0.68rem; color:rgba(255,255,255,0.25); text-align:right; }
.slots { display:flex; gap:8px; flex-wrap:wrap; align-items:center; }
.version-row { display:flex; align-items:center; gap:10px; }
.version-label { font-size:0.76rem; color:rgba(255,255,255,0.4); }
.version-switch { display:flex; border:1px solid rgba(255,255,255,0.12); border-radius:6px; overflow:hidden; }
.version-switch button { padding:6px 10px; border:0; border-right:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.03); color:rgba(255,255,255,0.45); font-size:0.72rem; cursor:pointer; }
.version-switch button:last-child { border-right:0; }
.version-switch button.on { background:rgba(103,232,249,0.12); color:#67e8f9; }
.slot-btn, .btn-refresh, .btn-sort { padding:6px 14px; border-radius:6px; border:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.05); color:rgba(255,255,255,0.5); font-size:0.78rem; cursor:pointer; transition:background 0.2s; }
.btn-sort.on { border-color:rgba(103,232,249,0.4); background:rgba(103,232,249,0.1); color:#67e8f9; }
.slot-btn { padding:8px 14px; }
.slot-btn:hover, .btn-refresh:hover, .btn-sort:hover { background:rgba(255,255,255,0.1); color:rgba(255,255,255,0.7); }
.slot-btn.on { border-color:rgba(103,232,249,0.4); background:rgba(103,232,249,0.1); color:#67e8f9; }
.batch-row { display:flex; gap:8px; align-items:center; }
.table { display:flex; flex-direction:column; background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.06); border-radius:12px; overflow:hidden; }
.row { display:flex; align-items:center; padding:7px 14px; gap:8px; border-bottom:1px solid rgba(255,255,255,0.02); }
.row:hover { background:rgba(255,255,255,0.02); }
.row-head { background:rgba(255,255,255,0.03); border-bottom:1px solid rgba(255,255,255,0.05); font-size:0.7rem; color:rgba(255,255,255,0.3); font-weight:600; }
.col-name { flex:1; font-size:0.8rem; color:rgba(255,255,255,0.6); overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.col-count { width:64px; text-align:right; font-size:0.8rem; color:#67e8f9; font-family:'Courier New',monospace; flex-shrink:0; }
.count-edit { display:flex; align-items:center; justify-content:flex-end; gap:5px; width:192px; }
.count-edit input { width:70px; padding:4px 6px; border:1px solid rgba(103,232,249,0.28); border-radius:4px; background:rgba(255,255,255,0.06); color:#fff; font:0.75rem 'Courier New',monospace; text-align:right; }
.edit-action { padding:3px 8px; border:1px solid rgba(255,255,255,0.14); border-radius:4px; background:transparent; color:rgba(255,255,255,0.55); font-size:0.68rem; cursor:pointer; }
.edit-action.save { border-color:rgba(103,232,249,0.3); color:#67e8f9; }
.edit-action:disabled { opacity:0.45; cursor:not-allowed; }
.confirm-backdrop { position:fixed; inset:0; z-index:10; display:flex; align-items:center; justify-content:center; padding:20px; background:rgba(0,0,0,0.62); }
.confirm-dialog { width:min(420px, 100%); padding:18px; border:1px solid rgba(103,232,249,0.24); border-radius:8px; background:#17222f; box-shadow:0 18px 48px rgba(0,0,0,0.55); }
.confirm-dialog h2 { margin:0 0 10px; color:#e8f8fb; font-size:0.95rem; font-weight:600; }
.confirm-dialog p { margin:0; color:rgba(255,255,255,0.66); font-size:0.8rem; line-height:1.65; }
.confirm-actions { display:flex; justify-content:flex-end; gap:8px; margin-top:16px; }
.confirm-actions button { padding:7px 14px; border-radius:5px; font-size:0.78rem; cursor:pointer; }
.confirm-cancel { border:1px solid rgba(255,255,255,0.15); background:transparent; color:rgba(255,255,255,0.62); }
.confirm-apply { border:1px solid rgba(103,232,249,0.35); background:rgba(103,232,249,0.14); color:#67e8f9; }

.empty { font-size:0.78rem; color:rgba(255,255,255,0.3); text-align:center; padding:12px 0; }
</style>
