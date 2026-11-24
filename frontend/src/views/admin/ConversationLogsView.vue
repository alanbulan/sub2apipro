<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card p-4">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
          <label class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.conversationLogs.filters.sessionId') }}
            <input v-model.trim="filters.session_id" class="input mt-1 w-full" :placeholder="t('admin.conversationLogs.filters.sessionPlaceholder')" @keyup.enter="applyFilters" />
          </label>
          <div ref="userSearchRef" class="relative text-sm text-gray-600 dark:text-gray-300">
            <label for="conversation-log-user">{{ t('admin.conversationLogs.filters.user') }}</label>
            <div class="relative mt-1">
              <input
                id="conversation-log-user"
                v-model="userKeyword"
                data-testid="conversation-user-search"
                type="text"
                class="input w-full pr-8"
                :placeholder="t('admin.conversationLogs.filters.userPlaceholder')"
                @input="onUserInput"
                @focus="showUserDropdown = true"
                @keyup.esc="showUserDropdown = false"
              />
              <button
                v-if="filters.user_id"
                type="button"
                class="absolute inset-y-0 right-2 text-gray-400 hover:text-gray-700 dark:hover:text-gray-200"
                :aria-label="t('admin.conversationLogs.filters.allUsers')"
                @click="selectAllUsers"
              >
                &times;
              </button>
            </div>
            <div
              v-if="showUserDropdown"
              class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-700 dark:bg-dark-800"
            >
              <button
                type="button"
                class="w-full px-4 py-2 text-left hover:bg-gray-100 dark:hover:bg-dark-700"
                @click="selectAllUsers"
              >
                {{ t('admin.conversationLogs.filters.allUsers') }}
              </button>
              <button
                v-for="user in userResults"
                :key="user.id"
                type="button"
                class="w-full px-4 py-2 text-left hover:bg-gray-100 dark:hover:bg-dark-700"
                @click="selectUser(user)"
              >
                <span>{{ user.email }}</span>
                <span v-if="user.deleted" class="ml-1 text-xs text-gray-400">({{ t('admin.conversationLogs.filters.deletedUser') }})</span>
                <span class="ml-2 text-xs text-gray-400">#{{ user.id }}</span>
              </button>
            </div>
          </div>
          <label class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.conversationLogs.filters.apiKeyId') }}
            <input v-model.number="filters.api_key_id" type="number" min="1" class="input mt-1 w-full" />
          </label>
          <label class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.conversationLogs.filters.requestId') }}
            <input v-model.trim="filters.request_id" class="input mt-1 w-full" @keyup.enter="applyFilters" />
          </label>
          <label class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.conversationLogs.filters.model') }}
            <input v-model.trim="filters.model" class="input mt-1 w-full" @keyup.enter="applyFilters" />
          </label>
          <label class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.conversationLogs.filters.protocol') }}
            <select v-model="filters.protocol" class="input mt-1 w-full">
              <option value="">{{ t('common.all') }}</option>
              <option value="openai_responses">OpenAI Responses</option>
              <option value="openai_responses_ws">OpenAI Responses WebSocket</option>
              <option value="openai_chat_completions">OpenAI Chat Completions</option>
              <option value="anthropic_messages">Anthropic Messages</option>
              <option value="gemini_generate_content">Gemini GenerateContent</option>
            </select>
          </label>
          <label class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.conversationLogs.filters.status') }}
            <select v-model="filters.status" class="input mt-1 w-full">
              <option value="">{{ t('common.all') }}</option>
              <option value="completed">{{ t('admin.conversationLogs.status.completed') }}</option>
              <option value="failed">{{ t('admin.conversationLogs.status.failed') }}</option>
              <option value="incomplete">{{ t('admin.conversationLogs.status.incomplete') }}</option>
              <option value="pending">{{ t('admin.conversationLogs.status.pending') }}</option>
            </select>
          </label>
          <label class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.conversationLogs.filters.startTime') }}
            <input v-model="filters.start_time" type="datetime-local" class="input mt-1 w-full" />
          </label>
          <label class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.conversationLogs.filters.endTime') }}
            <input v-model="filters.end_time" type="datetime-local" class="input mt-1 w-full" />
          </label>
          <div class="flex items-end gap-2">
            <button type="button" class="btn btn-primary" @click="applyFilters">{{ t('common.search') }}</button>
            <button type="button" class="btn btn-secondary" @click="resetFilters">{{ t('common.reset') }}</button>
          </div>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full min-w-[1100px] divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th v-for="column in columns" :key="column" class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-300">{{ column }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
              <tr v-if="loading"><td colspan="10" class="px-4 py-12 text-center text-gray-500">{{ t('common.loading') }}</td></tr>
              <tr v-else-if="!items.length"><td colspan="10" class="px-4 py-12 text-center text-gray-500">{{ t('admin.conversationLogs.empty') }}</td></tr>
              <tr v-for="item in items" v-else :key="item.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
                <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">{{ formatTime(item.created_at) }}</td>
                <td class="max-w-52 truncate px-4 py-3 font-mono text-xs" :title="item.session_id">{{ item.session_id }}</td>
                <td class="px-4 py-3 text-sm"><div>{{ item.user_email || `#${item.user_id}` }}</div><div class="text-xs text-gray-400">{{ item.api_key_name || `Key #${item.api_key_id}` }}</div></td>
                <td class="px-4 py-3 text-sm">{{ item.model || '-' }}</td>
                <td class="px-4 py-3 text-xs">{{ item.protocol }}</td>
                <td class="max-w-48 truncate px-4 py-3 font-mono text-xs" :title="item.endpoint">{{ item.endpoint }}</td>
                <td class="px-4 py-3 text-sm"><span :class="statusClass(item.status)" class="rounded px-2 py-1 text-xs font-medium">{{ t(`admin.conversationLogs.status.${item.status}`) }}</span></td>
                <td class="px-4 py-3 text-sm">{{ item.status_code || '-' }}</td>
                <td class="px-4 py-3 text-sm">{{ item.duration_ms }} ms</td>
                <td class="px-4 py-3"><button type="button" class="btn btn-ghost btn-sm" @click="openDetail(item.id)">{{ t('common.view') }}</button></td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination v-if="total > 0" :total="total" :page="page" :page-size="pageSize" @update:page="changePage" @update:pageSize="changePageSize" />
      </div>
    </div>

    <BaseDialog :show="detailVisible" :title="t('admin.conversationLogs.detailTitle')" width="full" @close="detailVisible = false">
      <div v-if="detailLoading" class="py-12 text-center text-gray-500">{{ t('common.loading') }}</div>
      <div v-else-if="detail" class="space-y-5">
        <dl class="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
          <div><dt class="text-gray-500">{{ t('admin.conversationLogs.filters.sessionId') }}</dt><dd class="break-all font-mono">{{ detail.session_id }}</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.usage.requestId') }}</dt><dd class="break-all font-mono">{{ detail.request_id }}</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.conversationLogs.filters.model') }}</dt><dd>{{ detail.model || '-' }}</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.conversationLogs.payloadSize') }}</dt><dd>{{ formatBytes(detail.request_bytes) }} / {{ formatBytes(detail.response_bytes) }}</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.conversationLogs.filters.userId') }}</dt><dd>{{ detail.user_email || `#${detail.user_id}` }}</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.conversationLogs.filters.apiKeyId') }}</dt><dd>{{ detail.api_key_name || `#${detail.api_key_id}` }}</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.conversationLogs.filters.protocol') }}</dt><dd>{{ detail.protocol }}</dd></div>
          <div><dt class="text-gray-500">{{ t('admin.conversationLogs.columns.endpoint') }}</dt><dd class="break-all font-mono">{{ detail.endpoint }}</dd></div>
        </dl>
        <div v-if="detail.request_truncated || detail.response_truncated" class="rounded border border-amber-300 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700 dark:bg-amber-950/30 dark:text-amber-300">{{ t('admin.conversationLogs.truncated') }}</div>
        <section><h4 class="mb-2 font-medium">{{ t('admin.conversationLogs.request') }}</h4><pre class="max-h-[40vh] overflow-auto whitespace-pre-wrap break-words rounded bg-gray-950 p-4 text-xs text-gray-100">{{ prettyPayload(detail.request_body) }}</pre></section>
        <section><h4 class="mb-2 font-medium">{{ t('admin.conversationLogs.response') }}</h4><pre class="max-h-[40vh] overflow-auto whitespace-pre-wrap break-words rounded bg-gray-950 p-4 text-xs text-gray-100">{{ prettyPayload(detail.response_body) }}</pre></section>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import { adminConversationLogsAPI, type ConversationLog, type ConversationLogQuery } from '@/api/admin/conversationLogs'
import { adminUsageAPI, type SimpleUser } from '@/api/admin/usage'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()
const items = ref<ConversationLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const detailLoading = ref(false)
const detailVisible = ref(false)
const detail = ref<ConversationLog | null>(null)
const userSearchRef = ref<HTMLElement | null>(null)
const userKeyword = ref('')
const userResults = ref<SimpleUser[]>([])
const showUserDropdown = ref(false)
let userSearchTimer: ReturnType<typeof setTimeout> | undefined
let userSearchRevision = 0
const filters = reactive({ session_id: '', request_id: '', user_id: undefined as number | undefined, api_key_id: undefined as number | undefined, model: '', protocol: '', status: '', start_time: '', end_time: '' })
const columns = computed(() => [t('admin.conversationLogs.columns.time'), t('admin.conversationLogs.columns.session'), t('admin.conversationLogs.columns.user'), t('admin.conversationLogs.columns.model'), t('admin.conversationLogs.columns.protocol'), t('admin.conversationLogs.columns.endpoint'), t('admin.conversationLogs.columns.status'), t('admin.conversationLogs.columns.httpStatus'), t('admin.conversationLogs.columns.duration'), t('common.actions')])

function query(): ConversationLogQuery {
  return { page: page.value, page_size: pageSize.value, session_id: filters.session_id || undefined, request_id: filters.request_id || undefined, user_id: filters.user_id || undefined, api_key_id: filters.api_key_id || undefined, model: filters.model || undefined, protocol: filters.protocol || undefined, status: filters.status || undefined, start_time: filters.start_time ? new Date(filters.start_time).toISOString() : undefined, end_time: filters.end_time ? new Date(filters.end_time).toISOString() : undefined }
}
async function load() { loading.value = true; try { const data = await adminConversationLogsAPI.list(query()); items.value = data.items || []; total.value = data.total || 0 } catch { appStore.showError(t('admin.conversationLogs.failedToLoad')) } finally { loading.value = false } }
function applyFilters() { showUserDropdown.value = false; page.value = 1; void load() }
function clearUserSearchTimer() {
  if (userSearchTimer) {
    clearTimeout(userSearchTimer)
    userSearchTimer = undefined
  }
  userSearchRevision += 1
}
function onUserInput() {
  filters.user_id = undefined
  clearUserSearchTimer()
  const keyword = userKeyword.value.trim()
  showUserDropdown.value = true
  if (!keyword) { userResults.value = []; return }
  const revision = userSearchRevision
  userSearchTimer = setTimeout(async () => {
    userSearchTimer = undefined
    try {
      const results = await adminUsageAPI.searchUsers(keyword)
      if (revision === userSearchRevision) userResults.value = results.sort((a, b) => Number(a.deleted) - Number(b.deleted))
    } catch {
      if (revision === userSearchRevision) userResults.value = []
    }
  }, 300)
}
function selectUser(user: SimpleUser) {
  clearUserSearchTimer()
  filters.user_id = user.id
  userKeyword.value = user.email
  userResults.value = []
  applyFilters()
}
function selectAllUsers() {
  clearUserSearchTimer()
  filters.user_id = undefined
  userKeyword.value = ''
  userResults.value = []
  applyFilters()
}
function resetFilters() {
  clearUserSearchTimer()
  userKeyword.value = ''
  userResults.value = []
  Object.assign(filters, { session_id: '', request_id: '', user_id: undefined, api_key_id: undefined, model: '', protocol: '', status: '', start_time: '', end_time: '' })
  applyFilters()
}
function changePage(value: number) { page.value = value; void load() }
function changePageSize(value: number) { pageSize.value = value; page.value = 1; void load() }
async function openDetail(id: number) { detailVisible.value = true; detailLoading.value = true; detail.value = null; try { detail.value = await adminConversationLogsAPI.get(id) } catch { appStore.showError(t('admin.conversationLogs.failedToLoad')) } finally { detailLoading.value = false } }
function formatTime(value: string) { return new Date(value).toLocaleString() }
function formatBytes(value: number) { if (value < 1024) return `${value} B`; if (value < 1048576) return `${(value / 1024).toFixed(1)} KB`; return `${(value / 1048576).toFixed(1)} MB` }
function prettyPayload(value = '') { if (!value) return '-'; try { return JSON.stringify(JSON.parse(value), null, 2) } catch { return value } }
function statusClass(status: ConversationLog['status']) { return status === 'completed' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' : status === 'pending' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300' : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300' }
function onDocumentClick(event: MouseEvent) { if (!userSearchRef.value?.contains(event.target as Node)) showUserDropdown.value = false }
onMounted(() => { document.addEventListener('click', onDocumentClick); void load() })
onUnmounted(() => { clearUserSearchTimer(); document.removeEventListener('click', onDocumentClick) })
</script>
