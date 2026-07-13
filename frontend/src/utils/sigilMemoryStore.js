import { ref, watch } from 'vue'

const TEMPLATES_KEY = 'gbfr-pe-patch-tool.sigilMemory.templates'
const HISTORY_KEY = 'gbfr-pe-patch-tool.sigilMemory.history'
const HISTORY_MAX = 20

function load(key) {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function save(key, value) {
  try { localStorage.setItem(key, JSON.stringify(value)) } catch {}
}

export const templates = ref(load(TEMPLATES_KEY))
export const history = ref(load(HISTORY_KEY))

watch(templates, (v) => save(TEMPLATES_KEY, v), { deep: true })
watch(history, (v) => save(HISTORY_KEY, v), { deep: true })

function newId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
}

function toEntry(form) {
  return {
    sigilHash: form.sigilHash >>> 0,
    sigilLevel: form.sigilLevel >>> 0,
    primaryTraitHash: form.primaryTraitHash >>> 0,
    primaryTraitLevel: form.primaryTraitLevel >>> 0,
    secondaryTraitHash: form.secondaryTraitHash >>> 0,
    secondaryTraitLevel: form.secondaryTraitLevel >>> 0,
  }
}

export function saveTemplate(name, form) {
  const trimmed = String(name || '').trim()
  if (!trimmed) return null
  const entry = { id: newId(), name: trimmed, createdAt: Date.now(), ...toEntry(form) }
  templates.value = [entry, ...templates.value]
  return entry
}

export function renameTemplate(id, name) {
  const trimmed = String(name || '').trim()
  if (!trimmed) return
  templates.value = templates.value.map(t => t.id === id ? { ...t, name: trimmed } : t)
}

export function deleteTemplate(id) {
  templates.value = templates.value.filter(t => t.id !== id)
}

export function pushHistory(form) {
  const entry = { id: newId(), createdAt: Date.now(), ...toEntry(form) }
  const next = [entry, ...history.value]
  if (next.length > HISTORY_MAX) next.length = HISTORY_MAX
  history.value = next
}
