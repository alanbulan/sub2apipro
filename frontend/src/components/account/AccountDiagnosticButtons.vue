<template>
  <div class="grid grid-cols-2 gap-2" data-test="account-diagnostic-actions">
    <button type="button" class="btn btn-secondary min-w-0 gap-1.5 !px-2 !text-xs" data-test="test-connection" @click="emit('test')">
      <Icon name="play" size="sm" class="shrink-0" aria-hidden="true" />
      {{ t('admin.accounts.testConnection') }}
    </button>
    <button type="button" class="btn btn-secondary min-w-0 gap-1.5 !px-2 !text-xs" data-test="test-intelligence" @click="emit('intelligence')">
      <Icon name="brain" size="sm" class="shrink-0" aria-hidden="true" />
      {{ t('admin.accounts.pelicanTest.title') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const emit = defineEmits<{ test: []; intelligence: [] }>()
</script>
