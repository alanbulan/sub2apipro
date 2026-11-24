<template>
  <div class="min-w-0">
    <div class="flex items-center gap-1">
      <button class="btn btn-secondary whitespace-nowrap" data-test="feishu-key-open" @click="open" @contextmenu.prevent="settingsOpen = true">
        <Icon name="mail" size="sm" class="mr-1.5" />{{ t('feishuKeyDelivery.action') }}
      </button>
      <button class="btn btn-secondary px-3" :title="t('feishuKeyDelivery.settings')" :aria-label="t('feishuKeyDelivery.settings')" data-test="feishu-key-settings" @click="settingsOpen = true"><Icon name="cog" size="sm" /></button>
    </div>
    <button v-if="job" class="mt-1 max-w-xs text-left text-xs text-gray-500 hover:underline" @click="show = true">
      {{ t(`feishuKeyDelivery.jobs.${job.status}`) }} · {{ t('feishuKeyDelivery.progress', job) }}
    </button>
    <BaseDialog :show="show" :title="t('feishuKeyDelivery.action')" width="extra-wide" @close="show = false">
      <div class="space-y-4">
        <p class="text-sm text-gray-500">{{ t('feishuKeyDelivery.hint') }}</p>
        <div v-if="!settings?.smtp_configured" class="rounded-xl bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-950/30 dark:text-amber-300">
          {{ t('feishuKeyDelivery.smtpMissing') }} <a class="underline" href="/admin/settings?tab=email">{{ t('feishuKeyDelivery.mailSettings') }}</a>
        </div>
        <div v-if="!running" class="flex flex-wrap items-end gap-3">
          <div class="min-w-64 flex-1">
            <label class="input-label" for="feishu-key-scope">{{ t('feishuKeyDelivery.scope') }}</label>
            <Select id="feishu-key-scope" v-model="scope" :options="scopeOptions" :disabled="busy" @update:model-value="preview = null" />
          </div>
          <button class="btn btn-secondary" :disabled="busy || !settings?.config.group_ids.length" data-test="feishu-key-preview" @click="prepare">{{ t(busy ? 'common.loading' : 'feishuKeyDelivery.preview') }}</button>
          <button class="btn btn-secondary" :disabled="busy" @click="settingsOpen = true">{{ t('feishuKeyDelivery.settings') }}</button>
        </div>
        <template v-if="display">
          <div class="flex flex-wrap gap-3 rounded-xl bg-gray-50 p-3 text-sm dark:bg-dark-800" role="status">
            <span>{{ t('feishuKeyDelivery.ready', { count: display.ready }) }}</span>
            <span>{{ t('feishuKeyDelivery.skipped', { count: display.skipped }) }}</span>
            <span>{{ t('feishuKeyDelivery.totalGrant', { amount: money(display.total_grant) }) }}</span>
            <span v-if="display.status !== 'preview'">{{ t('feishuKeyDelivery.progress', display) }}</span>
          </div>
          <div class="max-h-[45vh] overflow-auto rounded-xl border border-gray-200 dark:border-dark-700">
            <table class="w-full text-left text-sm">
              <thead class="sticky top-0 bg-gray-50 text-gray-500 dark:bg-dark-800"><tr>
                <th class="p-3">{{ t('feishuKeyDelivery.employee') }}</th><th class="p-3">{{ t('feishuKeyDelivery.recipient') }}</th>
                <th class="p-3">{{ t('feishuKeyDelivery.keyAndGroups') }}</th><th class="p-3">{{ t('feishuKeyDelivery.grant') }}</th><th class="p-3">{{ t('feishuKeyDelivery.status') }}</th>
              </tr></thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700"><tr v-for="(row, index) in display.rows" :key="`${row.user_id}-${index}`">
                <td class="p-3">{{ row.name || `ID ${row.user_id}` }}</td><td class="max-w-64 break-all p-3">{{ row.email || '—' }}</td>
                <td class="p-3"><span>{{ t(row.reuse ? 'feishuKeyDelivery.reuse' : 'feishuKeyDelivery.create') }}</span><p class="max-w-64 text-xs text-gray-500">{{ row.group_names?.join('、') || '—' }}</p></td>
                <td class="p-3">${{ money(row.grant) }}</td><td class="p-3"><span :class="row.status === 'sent' ? 'text-emerald-600' : row.reason ? 'text-amber-600' : ''">{{ t(`feishuKeyDelivery.states.${row.status}`) }}</span><p v-if="row.reason" class="max-w-64 text-xs text-gray-500">{{ t(`feishuKeyDelivery.reasons.${row.reason}`) }}</p></td>
              </tr></tbody>
            </table>
          </div>
        </template>
        <p v-if="preview" class="text-xs text-gray-500">{{ t('feishuKeyDelivery.previewHint') }}</p>
        <p v-if="display && display.status !== 'preview'" class="text-xs text-gray-500">{{ t('feishuKeyDelivery.acceptedHint') }}</p>
        <p v-if="pollFailed" class="text-sm text-amber-600">{{ t('feishuKeyDelivery.pollFailed') }} <button class="underline" @click="poll">{{ t('feishuKeyDelivery.retryStatus') }}</button></p>
        <label v-if="job?.uncertain && !running" class="flex items-center gap-2 text-sm"><Toggle v-model="retryUncertain" />{{ t('feishuKeyDelivery.retryUncertain') }}</label>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="show = false">{{ t('common.close') }}</button>
        <button v-if="preview && !running" class="btn btn-primary" data-test="feishu-key-send" :disabled="busy || !preview.ready || !settings?.smtp_configured || expired" @click="send">{{ t('feishuKeyDelivery.sendCount', { count: preview.ready }) }}</button>
        <button v-else-if="job && !running && (job.failed || (job.uncertain && retryUncertain))" class="btn btn-primary" :disabled="busy || !settings?.smtp_configured" data-test="feishu-key-retry" @click="retry">{{ t('feishuKeyDelivery.retryFailed') }}</button>
      </template>
    </BaseDialog>
    <FeishuKeyDeliverySettings :show="settingsOpen" @close="settingsOpen = false" @saved="onSaved" />
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import FeishuKeyDeliverySettings from './FeishuKeyDeliverySettings.vue'
import { currentFeishuKeyJob, getFeishuKeyConfig, previewFeishuKeys, retryFeishuKeys, startFeishuKeys, type FeishuKeyJob, type FeishuKeySettings } from '@/api/admin/feishuKeyDelivery'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
const props = defineProps<{ selectedIds: number[] }>()
const emit = defineEmits<{ updated: [] }>()
const { t } = useI18n()
const app = useAppStore()
const show = ref(false)
const settingsOpen = ref(false)
const settings = ref<FeishuKeySettings | null>(null)
const job = ref<FeishuKeyJob | null>(null)
const preview = ref<FeishuKeyJob | null>(null)
const scope = ref('all')
const busy = ref(false)
const pollFailed = ref(false)
const retryUncertain = ref(false)
const running = computed(() => job.value?.status === 'running')
const display = computed(() => running.value ? job.value : preview.value || job.value)
const expired = ref(false)
const scopeOptions = computed(() => [{ value: 'all', label: t('feishuKeyDelivery.all') }, { value: 'selected', label: t('feishuKeyDelivery.selected', { count: props.selectedIds.length }), disabled: !props.selectedIds.length }])
let timer: ReturnType<typeof setTimeout> | undefined
let disposed = false
function money(amount: number) { return Number(amount || 0).toFixed(8).replace(/\.?0+$/, '') || '0' }
function schedule() { clearTimeout(timer); if (!disposed && running.value) timer = setTimeout(poll, 2000) }
async function poll() {
  try {
    const current = await currentFeishuKeyJob()
    if (disposed) return
    const before = job.value
    job.value = current
    pollFailed.value = false
    if (before?.status === 'running' && current?.status !== 'running') emit('updated')
    schedule()
  } catch { if (!disposed) { pollFailed.value = true; schedule() } }
}
async function open() {
  show.value = true
  if (!running.value) { scope.value = props.selectedIds.length ? 'selected' : 'all'; preview.value = null }
  try { settings.value = await getFeishuKeyConfig(); if (!settings.value.config.group_ids.length) settingsOpen.value = true }
  catch (error) { report(error) }
}
function onSaved(value: FeishuKeySettings) { settings.value = value; preview.value = null }
function report(error: unknown) { app.showError(extractApiErrorMessage(error, t('feishuKeyDelivery.failed'))) }
async function prepare() {
  if (busy.value || running.value) return
  busy.value = true
  preview.value = null
  try { preview.value = await previewFeishuKeys(scope.value === 'selected' ? props.selectedIds : [], scope.value === 'all'); expired.value = false }
  catch (error) { report(error) }
  finally { busy.value = false }
}
async function send() {
  if (!preview.value || busy.value || running.value) return
  if (preview.value.expires_at * 1000 <= Date.now()) { expired.value = true; return }
  busy.value = true
  try { job.value = await startFeishuKeys(preview.value.id); preview.value = null; schedule() }
  catch (error) { report(error) }
  finally { busy.value = false }
}
async function retry() {
  if (!job.value || busy.value || running.value) return
  busy.value = true
  try { job.value = await retryFeishuKeys(job.value.id, retryUncertain.value); retryUncertain.value = false; schedule() }
  catch (error) { report(error) }
  finally { busy.value = false }
}
watch(() => props.selectedIds.slice(), () => { preview.value = null; if (!props.selectedIds.length) scope.value = 'all' })
onMounted(async () => { await poll(); try { const value = await getFeishuKeyConfig(); if (!disposed) settings.value = value } catch { /* Open retries and reports configuration errors. */ } })
onUnmounted(() => { disposed = true; clearTimeout(timer) })
</script>
