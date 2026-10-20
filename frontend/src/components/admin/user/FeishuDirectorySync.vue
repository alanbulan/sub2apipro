<template>
  <div class="min-w-0">
    <button class="btn btn-secondary w-full" :disabled="busy || starting" data-test="feishu-sync" @click="start">
      {{ t(busy || starting ? 'admin.users.feishuSync.running' : 'admin.users.feishuSync.action') }}
    </button>
    <p v-if="job" class="mt-1 max-w-xs text-xs text-gray-500" role="status" data-test="sync-status">
      {{ t(`admin.users.feishuSync.${job.status}`, job) }}
      <span v-if="job.error_code">{{ t(`admin.users.feishuSync.errors.${job.error_code}`) }}</span>
    </p>
    <button v-if="pollFailed" class="mt-1 text-xs text-red-600 underline" @click="poll">
      {{ t('admin.users.feishuSync.retryStatus') }}
    </button>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getFeishuDirectorySync, startFeishuDirectorySync, type FeishuDirectoryJob } from '@/api/admin/feishu'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
const emit = defineEmits<{ synced: [] }>()
const { t } = useI18n()
const app = useAppStore()
const job = ref<FeishuDirectoryJob | null>(null)
const starting = ref(false)
const pollFailed = ref(false)
const busy = computed(() => !!job.value && ['preparing', 'reading', 'syncing'].includes(job.value.status))
let timer: ReturnType<typeof setTimeout> | undefined
let disposed = false
function schedule() {
  clearTimeout(timer)
  if (!disposed && busy.value) timer = setTimeout(poll, 2000)
}
async function poll() {
  try {
    const previous = job.value
    const current = await getFeishuDirectorySync()
    if (disposed) return
    job.value = current
    pollFailed.value = false
    if (previous && current && !busy.value && previous.status !== current.status) emit('synced')
    schedule()
  } catch { if (!disposed) pollFailed.value = true }
}
async function start() {
  if (starting.value || busy.value) return
  if (!window.confirm(t('admin.users.feishuSync.confirm'))) return
  starting.value = true
  try {
    const current = await startFeishuDirectorySync()
    if (!disposed) { job.value = current; pollFailed.value = false; schedule() }
  } catch (error) {
    app.showError(extractApiErrorMessage(error, t('admin.users.feishuSync.failed')))
  } finally { starting.value = false }
}
onMounted(poll)
onUnmounted(() => { disposed = true; clearTimeout(timer) })
</script>
