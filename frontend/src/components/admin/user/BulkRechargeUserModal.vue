<template>
  <BaseDialog :show="show" :title="t('admin.users.bulkRecharge.title')" @close="close">
    <form id="bulk-recharge" class="space-y-4" @submit.prevent="submit">
      <p>{{ t('admin.users.bulkRecharge.count', { count: ids.length }) }}</p>
      <label class="input-label" for="recharge-amount">{{ t('admin.users.bulkRecharge.amount') }}</label>
      <input id="recharge-amount" v-model="amount" class="input" type="number" min="0.00000001" max="1000000000" step="0.00000001" required :disabled="!!pending || submitting" />
      <label class="input-label" for="recharge-notes">{{ t('admin.users.notes') }}</label>
      <input id="recharge-notes" v-model="notes" class="input" maxlength="1000" :disabled="!!pending || submitting" />
      <p class="font-medium">{{ t('admin.users.bulkRecharge.total', { total: total.toFixed(8).replace(/\.?0+$/, '') }) }}</p>
      <p class="text-sm text-gray-500">{{ t('admin.users.bulkRecharge.hint') }}</p>
      <p v-if="pending" class="text-sm text-amber-600" role="status">{{ t('admin.users.bulkRecharge.pending') }}</p>
    </form>
    <template #footer>
      <button class="btn btn-secondary" :disabled="submitting" @click="close">{{ t('common.cancel') }}</button>
      <button class="btn btn-primary" form="bulk-recharge" type="submit" data-test="recharge-submit" :disabled="!valid || submitting">{{ t(submitting ? 'common.saving' : 'admin.users.bulkRecharge.action') }}</button>
    </template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { BatchRechargeRequest } from '@/api/admin/users'
import { getAuthItem, AUTH_USER_KEY } from '@/utils/authStorage'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
const props = defineProps<{ show: boolean; selectedIds: number[] }>()
const emit = defineEmits<{ close: []; success: [affected: number] }>()
const { t } = useI18n()
const app = useAppStore()
const amount = ref('')
const notes = ref('')
const submitting = ref(false)
type Pending = { key: string; body: BatchRechargeRequest }
const pending = ref<Pending | null>(null)
function storageKey() {
  const user = JSON.parse(getAuthItem(AUTH_USER_KEY) || '{}') as { id?: number }
  if (!user.id) throw new Error(t('admin.users.bulkRecharge.sessionRequired'))
  return `batch-recharge-pending:${user.id}`
}
const ids = computed(() => pending.value?.body.user_ids || [...new Set(props.selectedIds)].sort((a,b) => a-b))
const total = computed(() => Number(amount.value) * ids.value.length || 0)
const valid = computed(() => ids.value.length > 0 && ids.value.length <= 500 && Number.isFinite(Number(amount.value)) && Number(amount.value) >= 1e-8 && Number(amount.value) <= 1e9)
watch(() => props.show, (show) => {
  if (!show) return
  try {
    const raw = localStorage.getItem(storageKey())
    pending.value = raw ? JSON.parse(raw) as Pending : null
    amount.value = pending.value ? String(pending.value.body.amount) : ''
    notes.value = pending.value?.body.notes || ''
  } catch (error) { app.showError(extractApiErrorMessage(error, t('admin.users.bulkRecharge.failed'))) }
}, { immediate: true })
function close() { if (!submitting.value) emit('close') }
async function submit() {
  if (!valid.value || submitting.value) return
  if (!pending.value && !window.confirm(t('admin.users.bulkRecharge.confirm', { count: ids.value.length, amount: amount.value, total: total.value }))) return
  submitting.value = true
  try {
    if (!pending.value) {
      const operation: Pending = { key: crypto.randomUUID(), body: { user_ids: [...ids.value], amount: Number(amount.value), notes: notes.value } }
      // Persist before sending. Closing the page or losing the response must
      // reuse the same operation rather than accidentally crediting twice.
      localStorage.setItem(storageKey(), JSON.stringify(operation))
      pending.value = operation
    }
    const result = await adminAPI.users.batchRecharge(pending.value.body, pending.value.key)
    localStorage.removeItem(storageKey())
    pending.value = null
    app.showSuccess(t('admin.users.bulkRecharge.success', { count: result.affected }))
    emit('success', result.affected)
    emit('close')
  } catch (error) {
    app.showError(extractApiErrorMessage(error, t('admin.users.bulkRecharge.failed')))
  } finally { submitting.value = false }
}
</script>
