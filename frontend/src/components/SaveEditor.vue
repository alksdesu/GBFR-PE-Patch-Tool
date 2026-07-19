<script setup>
import { ref, computed } from 'vue'
import { FindSaveFiles, GetBadgeUnlockStatus, GetQuests, LoadSave, UnlockAllBadges } from '../../wailsjs/go/main/App'

const slots = ref([])
const quests = ref([])
const total = ref(0)
const loading = ref(false)
const savePath = ref('')
const sortDesc = ref(true)
const badgeStatus = ref(null)
const markViewed = ref(false)
const unlocking = ref(false)
const backupPath = ref('')
const emit = defineEmits(['status'])

const sortedQuests = computed(() => {
  if (!sortDesc.value) return quests.value
  return [...quests.value].sort((a, b) => b.clears - a.clears)
})

async function scanSaves() {
  slots.value = await FindSaveFiles() || []
}

async function load(path) {
  loading.value = true
  savePath.value = path
  try {
    const [summary, qs, badges] = await Promise.all([LoadSave(path), GetQuests(path), GetBadgeUnlockStatus(path)])
    quests.value = qs || []
    total.value = summary?.questTotalClears || 0
    badgeStatus.value = badges
    backupPath.value = ''
  } catch (err) { console.error(err) } finally { loading.value = false }
}

async function unlockBadges() {
  if (!savePath.value || unlocking.value) return
  if (!window.confirm('请先完全退出游戏。将永久解锁 1615 个有效称号，并自动备份当前存档。确认继续吗？')) return
  unlocking.value = true
  try {
    const result = await UnlockAllBadges(savePath.value, markViewed.value)
    badgeStatus.value = result.status
    backupPath.value = result.backupPath || ''
    const changed = result.changed + result.viewedChanged
    emit('status', changed ? `称号解锁完成，修改 ${changed} 项` : '全部称号已经解锁', 'success')
  } catch (err) {
    emit('status', String(err), 'error')
  } finally {
    unlocking.value = false
  }
}

scanSaves()
</script>

<template>
  <div class="root">
    <!-- 存档选择 -->
    <div class="slots">
      <button v-for="s in slots" :key="s.index" class="slot-btn"
        :class="{ on: savePath === s.path }" @click="load(s.path)">
        {{ s.name }}
      </button>
      <button class="refresh" @click="scanSaves">刷新</button>
    </div>


    <section v-if="badgeStatus" class="badge-card">
      <div class="badge-summary">
        <div><strong>{{ badgeStatus.unlocked }} / {{ badgeStatus.total }}</strong><span>已解锁称号</span></div>
        <button class="unlock-btn" :disabled="unlocking || badgeStatus.allUnlocked" @click="unlockBadges">
          {{ unlocking ? '正在写入…' : badgeStatus.allUnlocked ? '已全部解锁' : '一键解锁全部称号' }}
        </button>
      </div>
      <label class="view-option"><input v-model="markViewed" type="checkbox"> 同时标记为已查看</label>
      <p class="badge-note">仅适配 2.0.2 存档；不会修改称号奖励领取状态。操作前请退出游戏。</p>
      <p v-if="backupPath" class="backup">备份：{{ backupPath }}</p>
    </section>

    <div v-if="loading" class="loading">解析中...</div>

    <div v-else-if="quests.length" class="quests">
      <div class="head">
        <span>{{ quests.length }} 个任务 · {{ total }} 次挑战</span>
        <button class="refresh" @click="load(savePath)">刷新</button>
        <button class="sort" @click="sortDesc = !sortDesc">{{ sortDesc ? '↓次数' : '↑默认' }}</button>
      </div>
      <div class="list">
        <div v-for="q in sortedQuests" :key="q.questId" class="row">
          <span class="id">{{ q.questId }}</span>
          <span class="name">{{ q.questNameCn || q.questName }}</span>
          <span class="count" :class="{ hot: q.clears > 100 }">{{ q.clears }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.root { display:flex; flex-direction:column; gap:10px; width:100%; max-width:720px; height:100%; min-height:0; margin:0 auto; padding-bottom:0; box-sizing:border-box; }
.slots { display:flex; gap:8px; flex-wrap:wrap; justify-content:center; align-items:center; }
.slot-btn {
  padding:10px 20px; border-radius:10px; border:1px solid rgba(255,255,255,0.1);
  background:rgba(255,255,255,0.04); color:rgba(255,255,255,0.45);
  font-size:0.82rem; font-family:inherit; cursor:pointer; transition:all 0.2s;
}
.slot-btn:hover { border-color:rgba(103,232,249,0.2); color:rgba(255,255,255,0.7); }
.slot-btn.on { border-color:rgba(103,232,249,0.4); background:rgba(103,232,249,0.1); color:#67e8f9; }
.refresh {
  padding:6px 14px; border-radius:6px; border:1px solid rgba(255,255,255,0.08);
  background:transparent; color:rgba(255,255,255,0.3); font-size:0.75rem; cursor:pointer;
}
.refresh:hover { color:rgba(255,255,255,0.6); border-color:rgba(255,255,255,0.15); }

.loading { text-align:center; color:#67e8f9; font-size:0.82rem; padding:16px; }

.quests { border-radius:12px; border:1px solid rgba(255,255,255,0.06); background:rgba(255,255,255,0.02); overflow:hidden; flex:1; min-height:0; display:flex; flex-direction:column; }
.head { display:flex; align-items:center; padding:10px 14px; background:rgba(255,255,255,0.03); border-bottom:1px solid rgba(255,255,255,0.05); gap:10px; }
.head span { font-size:0.75rem; color:rgba(255,255,255,0.35); flex:1; }
.sort {
  padding:3px 10px; border-radius:4px; border:1px solid rgba(255,255,255,0.1);
  background:transparent; color:rgba(255,255,255,0.3); font-size:0.7rem; cursor:pointer;
}
.sort:hover { color:#67e8f9; border-color:rgba(103,232,249,0.3); }

.list { flex:1; min-height:0; overflow-y:auto; scrollbar-width:thin; scrollbar-color:rgba(255,255,255,0.08) transparent; }
.row { display:flex; align-items:center; gap:8px; padding:7px 14px; border-bottom:1px solid rgba(255,255,255,0.02); }
.row:hover { background:rgba(255,255,255,0.02); }
.id { width:48px; font-size:0.68rem; color:rgba(255,255,255,0.2); font-family:'Courier New',monospace; flex-shrink:0; }
.name { flex:1; font-size:0.8rem; color:rgba(255,255,255,0.5); overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.count { width:40px; text-align:right; font-size:0.8rem; font-weight:600; color:rgba(255,255,255,0.35); font-family:'Courier New',monospace; flex-shrink:0; }
.count.hot { color:#fbbf24; }
.badge-card { padding:14px; border:1px solid var(--badge-border, rgba(103,232,249,.25)); border-radius:12px; background:var(--badge-bg, rgba(103,232,249,.06)); }
.badge-summary { display:flex; align-items:center; justify-content:space-between; gap:16px; }
.badge-summary div { display:flex; flex-direction:column; gap:3px; }
.badge-summary strong { color:#67e8f9; font-size:1.15rem; }
.badge-summary span,.view-option,.badge-note,.backup { color:rgba(255,255,255,.65); font-size:.75rem; }
.unlock-btn { padding:9px 16px; border:1px solid rgba(103,232,249,.45); border-radius:8px; background:rgba(103,232,249,.16); color:#cffafe; cursor:pointer; }
.unlock-btn:disabled { cursor:not-allowed; opacity:.5; }
.view-option { display:flex; align-items:center; gap:7px; margin-top:12px; }
.badge-note,.backup { margin:8px 0 0; line-height:1.5; overflow-wrap:anywhere; }
.backup { color:#86efac; }

</style>
