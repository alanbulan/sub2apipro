<template>
  <section class="card space-y-4 p-6" aria-labelledby="settings-section-feishu">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 id="settings-section-feishu" tabindex="-1" class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('feishu.title') }}</h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('feishu.description') }}</p>
      </div>
      <Toggle :aria-label="t('feishu.title')" :model-value="settings?.enabled ?? false" :disabled="busy || !settings?.app_secret_configured" @update:model-value="toggle" />
    </div>
    <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('feishu.tenantOnly') }}</p>
    <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('feishu.emailSyncHint') }}</p>
    <dl v-if="settings?.app_id" class="grid gap-2 text-sm sm:grid-cols-2">
      <div><dt class="text-gray-500">{{ t('feishu.enterprise') }}</dt><dd>{{ settings.tenant_name || settings.tenant_key }}</dd></div>
      <div><dt class="text-gray-500">App ID</dt><dd class="break-all">{{ settings.app_id }}</dd></div>
      <div class="sm:col-span-2"><dt class="text-gray-500">{{ t('feishu.callback') }}</dt><dd class="break-all">{{ settings.redirect_url }}</dd></div>
    </dl>
    <a v-if="settings?.app_id" :href="`https://open.feishu.cn/app/${encodeURIComponent(settings.app_id)}`" target="_blank" rel="noopener noreferrer" class="text-sm text-primary-600 underline">{{ t('feishu.manageApp') }}</a>
    <div class="flex flex-wrap gap-3">
      <button type="button" class="btn btn-primary" :disabled="busy || polling" @click="start">{{ busy ? t('common.processing') : settings?.app_id ? t('feishu.reconfigure') : t('feishu.scanSetup') }}</button>
      <button v-if="settings?.enabled && settings.app_id" type="button" class="btn btn-secondary" :disabled="busy || polling" @click="enableEmailSync">{{ t('feishu.enableEmailSync') }}</button>
    </div>
    <p v-if="emailSyncSubmitted" role="status" class="rounded-lg bg-green-50 p-3 text-sm text-green-800 dark:bg-green-900/20 dark:text-green-300">{{ t('feishu.emailSyncSubmitted') }}</p>
    <div v-if="setup?.verification_url && (polling || retryID)" class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-600">
      <p class="text-sm">{{ t('feishu.scanHint') }}</p>
      <img v-if="qr" :src="qr" :alt="t('feishu.qrAlt')" class="h-56 w-56 rounded-lg bg-white" />
      <a :href="setup.verification_url" target="_blank" rel="noopener noreferrer" class="text-sm text-primary-600 underline">{{ t('feishu.openOfficial') }}</a>
      <p class="text-sm text-gray-500" role="status">{{ t(polling ? 'feishu.waiting' : 'feishu.retryHint') }}</p>
      <button v-if="polling" type="button" class="btn btn-secondary" @click="stop">{{ t('feishu.stopWaiting') }}</button>
    </div>
    <p v-if="submitted" role="status" class="rounded-lg bg-green-50 p-3 text-sm text-green-800 dark:bg-green-900/20 dark:text-green-300">{{ t('feishu.submitted') }}</p>
    <p v-if="error" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>
    <button v-if="retryID && !polling" type="button" class="btn btn-secondary" :disabled="busy" @click="resume">{{ t('feishu.retry') }}</button>
    <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('feishu.savedSeparately') }}</p>
  </section>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import { useAppStore } from '@/stores'
import Toggle from '@/components/common/Toggle.vue'
import { getFeishuSettings, setFeishuEnabled, startFeishuSetup, pollFeishuSetup, enableFeishuEmailSync, type FeishuSettings, type FeishuSetup } from '@/api/admin/feishu'

const { t } = useI18n()
const appStore = useAppStore()
const settings = ref<FeishuSettings | null>(null)
const setup = ref<FeishuSetup | null>(null)
const qr = ref('')
const busy = ref(false)
const polling = ref(false)
const submitted = ref(false)
const emailSyncSubmitted = ref(false)
const error = ref('')
const retryID = ref('')
let timer: ReturnType<typeof setTimeout> | undefined
let generation = 0
function message(e: unknown): string {
  const err = e as { response?: { data?: { message?: string } }; message?: string }
  return err.response?.data?.message || err.message || t('feishu.failed')
}
function stop() { generation++; polling.value = false; clearTimeout(timer); retryID.value = '' }
function schedule(id: string, interval: number, current: number) {
  timer = setTimeout(async () => {
    if (current !== generation) return
    if (Date.now() >= (setup.value?.expires_at ?? 0) * 1000) { stop(); error.value = t('feishu.expired'); return }
    try {
      const result = await pollFeishuSetup(id)
      if (current !== generation) return
      if (result.status === 'submitted') {
        settings.value = result.settings ?? settings.value
        submitted.value = true; stop(); setup.value = null; qr.value = ''
        void appStore.fetchPublicSettings(true)
        return
      }
      schedule(id, result.interval, current)
    } catch (e) {
      if (current !== generation) return
      stop(); retryID.value = id; error.value = message(e)
    }
  }, Math.max(5, interval) * 1000)
}
async function start() {
  stop(); busy.value = true; error.value = ''; submitted.value = false
  const current = generation
  try {
    const result = await startFeishuSetup()
    if (current !== generation) return
    setup.value = result
    qr.value = await QRCode.toDataURL(result.verification_url || '', { width: 280, margin: 2 })
    if (current !== generation) return
    polling.value = true
    schedule(result.id!, result.interval, current)
  } catch (e) { if (current === generation) error.value = message(e) }
  finally { busy.value = false }
}
function resume() {
  const id = retryID.value
  if (!id) return
  error.value = ''; retryID.value = ''; polling.value = true; generation++
  schedule(id, setup.value?.interval ?? 5, generation)
}
async function toggle(enabled: boolean) {
  busy.value = true; error.value = ''
  try { settings.value = await setFeishuEnabled(enabled); void appStore.fetchPublicSettings(true) } catch (e) { error.value = message(e) }
  finally { busy.value = false }
}
async function enableEmailSync() {
  busy.value = true; error.value = ''; emailSyncSubmitted.value = false
  try { settings.value = await enableFeishuEmailSync(); emailSyncSubmitted.value = true }
  catch (e) { error.value = message(e) }
  finally { busy.value = false }
}
onMounted(async () => { try { settings.value = await getFeishuSettings() } catch (e) { error.value = message(e) } })
onBeforeUnmount(stop)
</script>
