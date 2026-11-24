<template>
  <BaseDialog :show="show" :title="t('intelligence.title')" width="wide" @close="close">
    <div class="space-y-4">
      <div class="rounded-xl border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950/20">
        <p class="font-semibold text-gray-900 dark:text-gray-100">{{ currentAccount?.name }}</p>
        <p class="mt-1 text-xs text-gray-600 dark:text-gray-300">{{ t('intelligence.description') }}</p>
      </div>
      <p v-if="error" role="alert" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300">{{ error }}</p>
      <div class="grid gap-3 sm:grid-cols-2">
        <div><label class="input-label">{{ t('intelligence.question') }}</label><Select v-model="questionKind" :options="questionOptions" :disabled="running" /></div>
        <div><label class="input-label">{{ t('intelligence.model') }}</label><Select v-model="modelId" :options="modelOptions" :disabled="running || loadingModels" /></div>
        <div><label class="input-label">{{ t('intelligence.reasoning') }}</label><Select v-model="reasoningEffort" :options="reasoningOptions" :disabled="running || questionKind === 'state_probe'" /></div>
        <Input v-if="!poolContext && questionKind !== 'state_probe'" v-model="parallelCount" type="number" min="1" max="4" :label="t('intelligence.parallel')" :disabled="running" />
      </div>
      <TextArea v-if="questionKind !== 'state_probe'" v-model="prompt" :label="t('intelligence.prompt')" :rows="4" :disabled="!!poolContext || running" />
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-primary" :disabled="running || !modelId || loadingModels" @click="startTest">{{ running ? t('intelligence.running') : t('intelligence.start') }}</button>
        <button v-if="running" type="button" class="btn btn-secondary" @click="controller?.abort()">{{ t('common.cancel') }}</button>
        <button v-if="!poolContext" type="button" class="btn btn-secondary" @click="scheduleOpen = !scheduleOpen; loadPlans()">{{ t('intelligence.schedule') }}</button>
      </div>
      <div v-if="!poolContext && scheduleOpen" class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-600">
        <div class="grid gap-3 sm:grid-cols-2">
          <Input v-model="cron" :label="t('intelligence.cron')" placeholder="0 */6 * * *" />
          <Input v-model="keepCount" type="number" min="1" max="200" :label="t('intelligence.retention')" />
        </div>
        <button type="button" class="btn btn-primary" :disabled="running || saving || !modelId" @click="savePlan">{{ t('intelligence.createSchedule') }}</button>
        <div v-for="plan in plans" :key="plan.id" class="flex flex-wrap items-center gap-2 border-t border-gray-100 pt-3 dark:border-dark-600">
          <span class="min-w-0 flex-1 break-words text-sm">{{ plan.model_id }} · {{ plan.cron_expression }} · {{ plan.enabled ? t('common.enabled') : t('common.disabled') }}</span>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="togglePlan(plan)">{{ plan.enabled ? t('intelligence.pause') : t('intelligence.resume') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="loadHistory(plan)">{{ t('intelligence.history') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="removePlan(plan)">{{ t('common.delete') }}</button>
        </div>
        <div v-if="pendingDelete" class="rounded-lg border border-amber-200 p-3 text-sm" role="alert">
          <p>{{ t('intelligence.deleteConfirm') }}</p>
          <div class="mt-2 flex gap-2"><button type="button" class="btn btn-danger btn-sm" @click="confirmDelete">{{ t('common.confirm') }}</button><button type="button" class="btn btn-secondary btn-sm" @click="pendingDelete = null">{{ t('common.cancel') }}</button></div>
        </div>
        <p v-if="plans.length === 0" class="text-sm text-gray-500">{{ t('intelligence.noPlans') }}</p>
      </div>
      <div v-if="history.length" class="space-y-2">
        <p class="text-sm font-semibold">{{ t('intelligence.history') }}</p>
        <button v-for="result in history" :key="result.id" type="button" class="flex w-full items-center justify-between gap-2 rounded-lg border border-gray-200 p-3 text-left text-sm dark:border-dark-600" @click="previewResult(result)">
          <span>{{ new Date(result.started_at).toLocaleString() }} · {{ result.pelican_config?.model_id }}</span><span>{{ result.status }} · {{ result.latency_ms }} ms</span>
        </button>
      </div>
      <div class="grid gap-4 xl:grid-cols-2">
        <article v-for="(run, index) in runs" :key="index" class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-600">
          <div class="flex items-center justify-between border-b border-gray-200 p-3 text-sm dark:border-dark-600"><span>{{ t('intelligence.output') }} {{ index + 1 }} · {{ run.model }}</span><span>{{ run.status }}</span></div>
          <p v-if="run.error" class="break-words p-3 text-sm text-red-600">{{ run.error }}</p>
          <iframe v-if="run.status === 'success' && !isTextAnswerKind(run.kind) && htmlFor(run.output)" :title="t('intelligence.preview')" :srcdoc="htmlFor(run.output)" sandbox="allow-scripts" referrerpolicy="no-referrer" class="h-[440px] w-full bg-white" />
          <pre class="max-h-80 overflow-auto whitespace-pre-wrap break-words p-3 text-xs text-gray-700 dark:text-gray-200">{{ run.output }}</pre>
        </article>
      </div>
    </div>
  </BaseDialog>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Input from '@/components/common/Input.vue'
import TextArea from '@/components/common/TextArea.vue'
import { adminAPI } from '@/api/admin'
import { apiClient } from '@/api/client'
import { streamIntelligence } from '@/api/intelligence'
import { getPoolDiagnosticModels, getPoolDiagnosticReasoning, poolDiagnosticPath, type PoolDiagnosticContext, type PoolDiagnosticAccount } from '@/api/poolDashboard'
import { authorizePelicanInlineScripts, extractPelicanHtml, readPelicanPreviewNonce } from '@/utils/pelicanHtml'
import { isTextAnswerKind, questionPrompt } from '@/utils/intelligenceTest'
import type { Account, ScheduledTestPlan, ScheduledTestResult } from '@/types'
const props = defineProps<{ show: boolean; account?: Account | null; accounts?: Account[]; poolContext?: PoolDiagnosticContext }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const currentAccount = computed<Account | PoolDiagnosticAccount | null>(() => props.poolContext?.account || props.account || null)
const modelId = ref(''), questionKind = ref('candy'), reasoningEffort = ref(''), parallelCount = ref<number | string>(1), prompt = ref(questionPrompt('candy'))
const modelOptions = ref<{ value: string; label: string }[]>([]), reasoningOptions = ref<{ value: string; label: string }[]>([])
const loadingModels = ref(false), running = ref(false), error = ref(''), saving = ref(false), scheduleOpen = ref(false)
const cron = ref('0 */6 * * *'), keepCount = ref<number | string>(50), plans = ref<ScheduledTestPlan[]>([]), history = ref<ScheduledTestResult[]>([]), pendingDelete = ref<ScheduledTestPlan | null>(null)
interface Run { model: string; kind: string; output: string; error: string; status: string }
const runs = ref<Run[]>([])
let controller: AbortController | null = null
let generation = 0
const questionOptions = computed(() => [{value:'candy',label:t('intelligence.candy')},{value:'pelican',label:t('intelligence.pelican')},...(!props.poolContext && currentAccount.value?.platform === 'openai' && currentAccount.value.type === 'oauth' ? [{value:'state_probe',label:t('intelligence.probe')}] : [])])
const htmlFor = (text: string) => authorizePelicanInlineScripts(extractPelicanHtml(text), readPelicanPreviewNonce())
watch(questionKind, kind => { if (kind !== 'state_probe') prompt.value = questionPrompt(kind === 'candy' ? 'candy' : 'pelican') })
async function loadModels() {
  const account = currentAccount.value
  if (!account) return
  const current = ++generation; loadingModels.value = true; error.value = ''
  try {
    const rows = props.poolContext ? await getPoolDiagnosticModels(props.poolContext) : await adminAPI.accounts.getAvailableModels(account.id)
    if (current !== generation || !props.show) return
    modelOptions.value = rows.map(row => ({ value: row.id, label: row.display_name || row.id }))
    if (!modelOptions.value.some(item => item.value === modelId.value)) modelId.value = modelOptions.value[0]?.value || ''
    await loadReasoning()
  } catch (err) { if (current === generation) error.value = message(err) }
  finally { if (current === generation) loadingModels.value = false }
}
async function loadReasoning() {
  if (!modelId.value) return
  if (props.poolContext) {
    const result = await getPoolDiagnosticReasoning(props.poolContext, modelId.value)
    reasoningOptions.value = result.supported_reasoning_levels.map(value => ({ value, label:value }))
    reasoningEffort.value = result.default_reasoning_level
  } else {
    reasoningOptions.value = [{value:'',label:t('intelligence.defaultReasoning')},...['low','medium','high','xhigh','max','ultra'].map(value=>({value,label:value}))]
  }
}
function message(err: unknown) { return err instanceof Error ? err.message : String(err) }
async function startTest() {
  const account = currentAccount.value
  if (!account || !modelId.value || running.value) return
  error.value = ''; running.value = true; controller = new AbortController()
  const count = props.poolContext || questionKind.value === 'state_probe' ? 1 : Math.max(1, Math.min(4, Number(parallelCount.value) || 1))
  const signal = controller.signal
  runs.value = Array.from({length:count},()=>({model:modelId.value,kind:questionKind.value,output:'',error:'',status:'running'}))
  try {
    await Promise.all(runs.value.map(async run => {
      try {
        if (run.kind === 'state_probe') {
          const result = await apiClient.post(`/admin/accounts/${account.id}/probe-state`, {model_id:run.model},{signal,timeout:105000})
          run.output = JSON.stringify(result.data, null, 2)
        } else {
          const path = props.poolContext ? poolDiagnosticPath(props.poolContext,'pelican-test') : `/admin/accounts/${account.id}/test-pelican`
          const body = props.poolContext ? {model_id:run.model,question_kind:run.kind,reasoning_effort:reasoningEffort.value} : {model_id:run.model,prompt:prompt.value + '\n\n' + (run.kind === 'candy' ? '只输出最终整数，不要解释。' : '只输出独立 HTML，不使用外部依赖。'),reasoning_effort:reasoningEffort.value}
          await streamIntelligence(path,body,signal,event=>{if(event.type==='content')run.output += event.text || ''},!props.poolContext)
        }
        run.status='success'
      } catch (err) { run.status='failed';run.error=message(err) }
    }))
  } finally { running.value=false;controller=null }
}
async function loadPlans() {
  if (props.poolContext || !currentAccount.value) return
  try { plans.value=(await adminAPI.scheduledTests.listByAccount(currentAccount.value.id)).filter(plan=>!!plan.pelican_config) } catch(err){error.value=message(err)}
}
async function savePlan() {
  if (!currentAccount.value || props.poolContext) return
  saving.value=true
  try {
    await adminAPI.scheduledTests.create({account_id:currentAccount.value.id,model_id:modelId.value,cron_expression:cron.value,enabled:true,max_results:Number(keepCount.value),auto_recover:false,pelican_config:{question_kind:questionKind.value,prompt:questionKind.value==='state_probe'?'':prompt.value,reasoning_effort:reasoningEffort.value,parallel_count:questionKind.value==='state_probe'?1:Number(parallelCount.value)}})
    await loadPlans()
  } catch(err){error.value=message(err)} finally {saving.value=false}
}
async function togglePlan(plan:ScheduledTestPlan) {saving.value=true;try{await adminAPI.scheduledTests.update(plan.id,{enabled:!plan.enabled});await loadPlans()}catch(err){error.value=message(err)}finally{saving.value=false}}
function removePlan(plan:ScheduledTestPlan){pendingDelete.value=plan}
async function confirmDelete(){const plan=pendingDelete.value;if(!plan)return;saving.value=true;try{await adminAPI.scheduledTests.delete(plan.id);pendingDelete.value=null;history.value=[];await loadPlans()}catch(err){error.value=message(err)}finally{saving.value=false}}
async function loadHistory(plan:ScheduledTestPlan){try{history.value=await adminAPI.scheduledTests.listResults(plan.id,plan.max_results)}catch(err){error.value=message(err)}}
function previewResult(result:ScheduledTestResult){runs.value=[{model:result.pelican_config?.model_id||'',kind:result.pelican_config?.question_kind||'pelican',output:result.response_text,error:result.error_message,status:result.status}]}
function close(){controller?.abort();generation++;emit('close')}
watch(()=>[props.show,currentAccount.value?.id],()=>{if(props.show){history.value=[];plans.value=[];runs.value=[];scheduleOpen.value=false;void loadModels()}else{controller?.abort();generation++}},{immediate:true})
watch(modelId,()=>{if(props.show && !loadingModels.value)void loadReasoning().catch(err=>{error.value=message(err)})})
onBeforeUnmount(()=>{controller?.abort();generation++})
</script>
