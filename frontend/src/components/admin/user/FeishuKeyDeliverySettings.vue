<template>
  <BaseDialog :show="show" :title="t('feishuKeyDelivery.settings')" width="wide" @close="close">
    <form v-if="draft" id="feishu-key-settings" class="space-y-5" @submit.prevent="save">
      <p class="text-sm text-gray-500">{{ t('feishuKeyDelivery.settingsHint') }}</p>
      <div v-if="!smtpConfigured" class="rounded-xl bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-950/30 dark:text-amber-300">
        {{ t('feishuKeyDelivery.smtpMissing') }}
        <a class="underline" href="/admin/settings?tab=email">{{ t('feishuKeyDelivery.mailSettings') }}</a>
      </div>
      <div>
        <label for="feishu-key-groups" class="input-label">{{ t('feishuKeyDelivery.groups') }}</label>
        <Select id="feishu-key-groups" v-model="selectedGroup" :options="groupOptions" searchable :disabled="saving" />
        <p class="mt-1 text-xs text-gray-500">{{ t('feishuKeyDelivery.groupsHint') }}</p>
      </div>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div v-for="field in numericFields" :key="field">
          <label :for="`feishu-key-${field}`" class="input-label">{{ t(`feishuKeyDelivery.fields.${field}`) }}</label>
          <input :id="`feishu-key-${field}`" v-model.number="draft[field]" class="input" type="number" min="0" :max="field === 'expires_in_days' ? 3650 : 1000000000" :step="field === 'expires_in_days' ? 1 : '0.00000001'" required :disabled="saving" />
        </div>
      </div>
      <p class="text-xs text-gray-500">{{ t('feishuKeyDelivery.limitsHint') }}</p>
      <div class="flex items-center justify-between gap-3 rounded-xl bg-gray-50 p-3 dark:bg-dark-800">
        <span class="text-sm">{{ t('feishuKeyDelivery.contactFallback') }}</span>
        <Toggle v-model="draft.allow_contact_email" :disabled="saving" />
      </div>
      <div>
        <label for="feishu-key-name" class="input-label">{{ t('feishuKeyDelivery.nameSuffix') }}</label>
        <input id="feishu-key-name" v-model="draft.name_suffix" class="input" maxlength="100" required :disabled="saving" />
      </div>
      <div>
        <label for="feishu-key-api" class="input-label">{{ t('feishuKeyDelivery.apiURL') }}</label>
        <input id="feishu-key-api" v-model="draft.api_url" class="input" type="url" placeholder="https://api.example.com/v1" required :disabled="saving" />
      </div>
      <div>
        <label for="feishu-key-subject" class="input-label">{{ t('feishuKeyDelivery.subject') }}</label>
        <input id="feishu-key-subject" v-model="draft.subject" class="input" maxlength="200" required :disabled="saving" />
      </div>
      <div>
        <label for="feishu-key-intro" class="input-label">{{ t('feishuKeyDelivery.introduction') }}</label>
        <textarea id="feishu-key-intro" v-model="draft.introduction" class="input min-h-24" maxlength="4000" :disabled="saving" />
      </div>
    </form>
    <p v-else class="text-sm text-gray-500">{{ t('common.loading') }}</p>
    <template #footer>
      <button class="btn btn-secondary" :disabled="saving" @click="close">{{ t('common.cancel') }}</button>
      <button class="btn btn-primary" form="feishu-key-settings" type="submit" :disabled="saving || !draft || !draft.group_ids.length">{{ t(saving ? 'common.saving' : 'common.save') }}</button>
    </template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import { getAll } from '@/api/admin/groups'
import { getFeishuKeyConfig, saveFeishuKeyConfig, type FeishuKeyConfig, type FeishuKeySettings } from '@/api/admin/feishuKeyDelivery'
import type { AdminGroup } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; saved: [settings: FeishuKeySettings] }>()
const { t } = useI18n()
const app = useAppStore()
const draft = ref<FeishuKeyConfig | null>(null)
const groups = ref<AdminGroup[]>([])
const saving = ref(false)
const smtpConfigured = ref(false)
const numericFields = ['initial_balance', 'quota', 'expires_in_days', 'rate_limit_5h', 'rate_limit_1d', 'rate_limit_7d'] as const
const selectedGroup = computed({ get: () => draft.value?.group_ids[0] ?? '', set: (value: string | number | null) => { if (draft.value) draft.value.group_ids = Number(value) > 0 ? [Number(value)] : [] } })
const groupOptions = computed(() => groups.value.map(g => ({ value: g.id, label: `${g.name}${g.subscription_type === 'subscription' ? ' · ' + t('feishuKeyDelivery.subscription') : ''}` })))
watch(() => props.show, async show => {
  if (!show) return
  draft.value = null
  try {
    const [settings, active] = await Promise.all([getFeishuKeyConfig(), getAll()])
    if (!props.show) return
    draft.value = { ...settings.config, group_ids: [...settings.config.group_ids] }
    smtpConfigured.value = settings.smtp_configured
    groups.value = active
  } catch (error) { app.showError(extractApiErrorMessage(error, t('feishuKeyDelivery.failed'))) }
})
function close() { if (!saving.value) emit('close') }
async function save() {
  if (!draft.value || saving.value) return
  saving.value = true
  try {
    const result = await saveFeishuKeyConfig(draft.value)
    app.showSuccess(t('feishuKeyDelivery.saved'))
    emit('saved', result)
    emit('close')
  } catch (error) { app.showError(extractApiErrorMessage(error, t('feishuKeyDelivery.failed'))) }
  finally { saving.value = false }
}
</script>
