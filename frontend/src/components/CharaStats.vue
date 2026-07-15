<script setup>
import { ref, computed, onMounted } from 'vue'
import { FindSaveFiles, GetCharacterStats } from '../../wailsjs/go/main/App'

const slots = ref([])
const list = ref([])
const savePath = ref('')
const loading = ref(false)
const sortDesc = ref(false)
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
    list.value = await GetCharacterStats(path) || []
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

onMounted(scanSaves)
</script>

<template>
  <div class="root">
    <div class="section">
      <div class="header">
        <span class="title">角色次数统计</span>
        <span class="hint">显示存档角色任务次数（目前芙劳缺失）</span>
      </div>
      <div class="slots">
        <button v-for="s in slots" :key="s.index" class="slot-btn"
          :class="{ on: savePath === s.path }" @click="load(s.path)">
          {{ s.name }}
        </button>
        <button class="btn-refresh" @click="refresh">刷新</button>
      </div>

      <div v-if="loading" class="empty">解析中...</div>
      <div v-else-if="error" class="empty">{{ error }}</div>
      <template v-else-if="list.length">
        <div class="batch-row">
          <button class="btn-sort" @click="sortDesc = !sortDesc">
            {{ sortDesc ? '恢复原序' : '按次数排序' }}
          </button>
        </div>
        <div class="table">
          <div class="row row-head">
            <span class="col-name">角色</span>
            <span class="col-count">次数</span>
          </div>
          <div v-for="c in sorted" :key="c.name" class="row">
            <span class="col-name">{{ c.name }}</span>
            <span class="col-count">{{ c.count }}</span>
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
.slot-btn, .btn-refresh, .btn-sort { padding:6px 14px; border-radius:6px; border:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.05); color:rgba(255,255,255,0.5); font-size:0.78rem; cursor:pointer; transition:background 0.2s; }
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
.empty { font-size:0.78rem; color:rgba(255,255,255,0.3); text-align:center; padding:12px 0; }
</style>
