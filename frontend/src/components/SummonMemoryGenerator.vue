<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { SummonMemoryGetOptions, SummonMemoryList, SummonMemoryUpdate } from '../../wailsjs/go/main/App'

const emit = defineEmits(['status'])
const options = reactive({ summons: [], traits: [], subTraits: [], maxTraitLevel: 15 })
const entries = ref([])
const selected = ref(null)
const form = reactive({ index: -1, typeHash: 0, primaryHash: 0, primaryLevel: 15, secondaryHash: 0, secondaryParam: 0, rank: 1 })
const loading = ref(false)
const applying = ref(false)
const typeSearch = ref('')
const primarySearch = ref('')
const secondarySearch = ref('')

function show(msg, type) { emit('status', msg, type) }
function hex(v) { return `0x${(Number(v) >>> 0).toString(16).toUpperCase().padStart(8, '0')}` }

const subByHash = computed(() => {
  const m = {}
  options.subTraits.forEach(s => { m[s.hash >>> 0] = s })
  return m
})
const traitByHash = computed(() => {
  const m = {}
  options.traits.forEach(t => { m[t.hash >>> 0] = t })
  return m
})

function filterList(items, query) {
  const q = query.trim().toLowerCase()
  return !q ? items : items.filter(i => i.displayName.toLowerCase().includes(q) || hex(i.hash).toLowerCase().includes(q))
}
const filteredTypes = computed(() => filterList(options.summons, typeSearch.value))
const filteredPrimary = computed(() => filterList(options.traits, primarySearch.value))
const filteredSecondary = computed(() => filterList(options.subTraits, secondarySearch.value))

const currentSub = computed(() => subByHash.value[form.secondaryHash >>> 0] || null)
const subParamText = computed(() => {
  const sub = currentSub.value
  if (!sub || !sub.values || !sub.values.length) return ''
  const idx = Math.min(Math.max(form.secondaryParam, 0), sub.values.length - 1)
  const val = sub.values[idx]
  const max = sub.values[sub.values.length - 1]
  const suffix = sub.isPercent ? '%' : ''
  return `→ ${val}${suffix}（最高 ${max}${suffix}）`
})
const subMaxParam = computed(() => currentSub.value ? currentSub.value.maxLevel : 9)

async function loadOptions() {
  try { Object.assign(options, await SummonMemoryGetOptions()) }
  catch (e) { show('读取召唤石数据失败: ' + String(e), 'error') }
}
async function refresh() {
  loading.value = true
  try {
    entries.value = await SummonMemoryList()
    show(`已读取 ${entries.value.length} 个召唤石`, 'success')
    if (selected.value != null) {
      const still = entries.value.find(e => e.index === selected.value)
      if (still) selectEntry(still)
      else clearSelection()
    }
  } catch (e) { entries.value = []; clearSelection(); show(String(e), 'error') }
  finally { loading.value = false }
}
function clearSelection() {
  selected.value = null
  form.index = -1
}
function selectEntry(entry) {
  selected.value = entry.index
  form.index = entry.index
  form.typeHash = entry.typeHash >>> 0
  form.primaryHash = entry.primaryHash >>> 0
  form.primaryLevel = entry.primaryLevel >>> 0
  form.secondaryHash = entry.secondaryHash >>> 0
  form.secondaryParam = entry.secondaryParam >>> 0
  form.rank = entry.rank >>> 0
  typeSearch.value = entry.typeName || ''
  primarySearch.value = entry.primaryName || ''
  secondarySearch.value = entry.secondaryName || ''
}
async function write() {
  if (form.index < 0) { show('请先在左侧选择一个召唤石', 'error'); return }
  applying.value = true
  try {
    const next = await SummonMemoryUpdate({
      index: form.index,
      typeHash: form.typeHash >>> 0,
      primaryHash: form.primaryHash >>> 0,
      primaryLevel: form.primaryLevel >>> 0,
      secondaryHash: form.secondaryHash >>> 0,
      secondaryParam: form.secondaryParam >>> 0,
      rank: form.rank >>> 0,
    })
    const i = entries.value.findIndex(e => e.index === next.index)
    if (i >= 0) entries.value.splice(i, 1, next)
    selectEntry(next)
    show(`已写入并保存: ${next.typeName}`, 'success')
  } catch (e) { show(String(e), 'error') }
  finally { applying.value = false }
}
function chooseType(item) { if (item) { form.typeHash = item.hash >>> 0; typeSearch.value = item.displayName } }
function choosePrimary(item) { if (item) { form.primaryHash = item.hash >>> 0; primarySearch.value = item.displayName } }
function chooseSecondary(item) { if (item) { form.secondaryHash = item.hash >>> 0; secondarySearch.value = item.displayName } }

function entryTraitText(e) {
  const parts = []
  if (e.primaryName) parts.push(`${e.primaryName} Lv${e.primaryLevel}`)
  if (e.secondaryName) parts.push(e.secondaryName)
  return parts.join(' · ')
}

onMounted(async () => { await loadOptions() })
</script>

<template>
  <div class="memory-summon">
    <div class="section">
      <div class="section-title">内存连接</div>
      <div class="actions">
        <button class="btn btn-cyan" :disabled="loading" @click="refresh">{{ loading ? '读取中...' : '读取召唤石列表' }}</button>
      </div>
      <div class="hint">在游戏内进入召唤石界面后点击读取。修改会写入内存并触发游戏保存，建议再手动存档一次。</div>
    </div>

    <div class="grid">
      <div class="section list-section">
        <div class="section-title">召唤石列表 <span class="count" v-if="entries.length">{{ entries.length }}</span></div>
        <div v-if="!entries.length" class="empty">尚未读取，或未解锁召唤石</div>
        <div v-else class="entry-list">
          <button
            v-for="e in entries"
            :key="e.index"
            class="entry"
            :class="{ active: selected === e.index }"
            @click="selectEntry(e)"
          >
            <div class="entry-head"><span class="entry-idx">#{{ e.index }}</span><span class="entry-name">{{ e.typeName }}</span></div>
            <div class="entry-sub">{{ entryTraitText(e) }}</div>
          </button>
        </div>
      </div>

      <div class="section" :class="{ muted: form.index < 0 }">
        <div class="section-title">编辑 <span v-if="form.index >= 0" class="count">#{{ form.index }}</span></div>
        <template v-if="form.index >= 0">
          <div class="field">
            <label>种类（本体）</label>
            <input v-model="typeSearch" class="input" placeholder="搜索召唤石种类或 0x 哈希" />
            <select class="select list" size="5" :value="form.typeHash" @change="chooseType(filteredTypes.find(v => (v.hash >>> 0) === Number($event.target.value)))">
              <option v-for="item in filteredTypes" :key="item.hash" :value="item.hash >>> 0">{{ item.displayName }} · {{ hex(item.hash) }}</option>
            </select>
          </div>
          <div class="field">
            <label>阶级</label>
            <div class="rank-row">
              <!-- 阶级 0 是部分召唤石(莉莉丝/路西法等)的合法内部值, 读到时显示该档位以便保持 -->
              <button v-if="form.rank === 0" class="rank-btn on" disabled>0</button>
              <button v-for="r in [1,2,3]" :key="r" class="rank-btn" :class="{ on: form.rank === r }" @click="form.rank = r">
                {{ ['Ⅰ','Ⅱ','Ⅲ'][r-1] }}
              </button>
            </div>
          </div>
          <div class="field">
            <label>主因子</label>
            <input v-model="primarySearch" class="input" placeholder="搜索主因子或 0x 哈希" />
            <select class="select list" size="5" :value="form.primaryHash" @change="choosePrimary(filteredPrimary.find(v => (v.hash >>> 0) === Number($event.target.value)))">
              <option v-for="item in filteredPrimary" :key="item.hash" :value="item.hash >>> 0">{{ item.displayName }} · {{ hex(item.hash) }}</option>
            </select>
          </div>
          <div class="level-row">
            <label>主因子等级</label>
            <input v-model.number="form.primaryLevel" class="input level" type="number" min="1" :max="options.maxTraitLevel" />
            <span class="lv-hint">上限 {{ options.maxTraitLevel }}</span>
          </div>
          <div class="field">
            <label>副特性</label>
            <input v-model="secondarySearch" class="input" placeholder="搜索副特性" />
            <select class="select list" size="5" :value="form.secondaryHash" @change="chooseSecondary(filteredSecondary.find(v => (v.hash >>> 0) === Number($event.target.value)))">
              <option v-for="item in filteredSecondary" :key="item.hash" :value="item.hash >>> 0">{{ item.displayName }} · {{ hex(item.hash) }}</option>
            </select>
          </div>
          <div class="level-row">
            <label>副特性参数</label>
            <input v-model.number="form.secondaryParam" class="input level" type="number" min="0" :max="subMaxParam" />
            <span class="lv-hint">{{ subParamText }}</span>
          </div>
          <button class="btn btn-green write" :disabled="applying" @click="write">{{ applying ? '写入中...' : '写入召唤石' }}</button>
        </template>
        <div v-else class="empty">从左侧选择一个召唤石开始编辑</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.memory-summon { width:100%; display:flex; flex-direction:column; gap:14px; }
.section { padding:14px 16px; border:1px solid rgba(255,255,255,.08); border-radius:8px; background:rgba(255,255,255,.04); display:flex; flex-direction:column; gap:10px; }
.section.muted { opacity:.55; }
.section-title { color:rgba(255,255,255,.4); font-size:.78rem; font-weight:600; letter-spacing:1px; display:flex; align-items:center; gap:8px; }
.count { color:#67e8f9; font-size:.72rem; }
.actions, .level-row, .rank-row { display:flex; gap:8px; align-items:center; }
.hint { font-size:.74rem; color:rgba(255,255,255,.35); line-height:1.5; }
.grid { display:grid; grid-template-columns:minmax(220px,1fr) minmax(280px,1.3fr); gap:14px; align-items:start; }
.list-section { max-height:520px; }
.entry-list { display:flex; flex-direction:column; gap:6px; overflow-y:auto; max-height:460px; scrollbar-width:thin; scrollbar-color:rgba(255,255,255,.12) transparent; }
.entry { text-align:left; padding:8px 10px; border:1px solid rgba(255,255,255,.1); border-radius:6px; background:rgba(255,255,255,.04); color:#fff; cursor:pointer; display:flex; flex-direction:column; gap:3px; transition:border-color .2s, background .2s; }
.entry:hover { background:rgba(255,255,255,.08); }
.entry.active { border-color:rgba(103,232,249,.5); background:rgba(103,232,249,.1); }
.entry-head { display:flex; align-items:baseline; gap:8px; }
.entry-idx { font-size:.68rem; color:rgba(255,255,255,.3); font-family:'Courier New',monospace; }
.entry-name { font-size:.82rem; font-weight:600; color:rgba(255,255,255,.78); }
.entry-sub { font-size:.7rem; color:rgba(255,255,255,.4); }
.field { display:flex; flex-direction:column; gap:5px; }
.field label, .level-row label { color:rgba(255,255,255,.35); font-size:.72rem; }
.level-row label { flex:0 0 auto; min-width:72px; }
.readonly { padding:8px 10px; border:1px solid rgba(255,255,255,.08); border-radius:6px; background:rgba(255,255,255,.03); color:rgba(255,255,255,.68); font-size:.8rem; }
.input, .select { box-sizing:border-box; width:100%; padding:8px 10px; border:1px solid rgba(255,255,255,.12); border-radius:6px; background:rgba(255,255,255,.06); color:#fff; outline:none; font:inherit; font-size:.8rem; }
.input:focus, .select:focus { border-color:rgba(103,232,249,.5); }
.select option { background:#1b2636; color:#fff; }
.list { min-height:106px; padding:3px; }
.list option { padding:4px 6px; }
.level { width:96px; }
.lv-hint { font-size:.72rem; color:rgba(255,255,255,.4); }
.rank-btn { flex:1; padding:7px 0; border:1px solid rgba(255,255,255,.14); border-radius:6px; background:rgba(255,255,255,.05); color:rgba(255,255,255,.6); font-size:.82rem; font-weight:600; cursor:pointer; transition:all .2s; }
.rank-btn:hover { background:rgba(255,255,255,.1); }
.rank-btn.on { border-color:rgba(103,232,249,.45); background:rgba(103,232,249,.12); color:#67e8f9; }
.btn { padding:8px 14px; border:1px solid rgba(255,255,255,.16); border-radius:6px; background:rgba(255,255,255,.06); color:rgba(255,255,255,.75); font-size:.8rem; font-weight:600; cursor:pointer; }
.btn:disabled { opacity:.4; cursor:not-allowed; }
.btn-cyan { border-color:rgba(103,232,249,.35); color:#67e8f9; background:rgba(103,232,249,.1); }
.btn-green { border-color:rgba(74,222,128,.35); color:#4ade80; background:rgba(74,222,128,.1); }
.write { align-self:flex-end; }
.empty { font-size:.76rem; color:rgba(255,255,255,.3); text-align:center; padding:20px 0; }
</style>
