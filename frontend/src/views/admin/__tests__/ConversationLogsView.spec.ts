import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ConversationLogsView from '../ConversationLogsView.vue'

const { listConversationLogs, getConversationLog, searchUsers, showError } = vi.hoisted(() => ({
  listConversationLogs: vi.fn(),
  getConversationLog: vi.fn(),
  searchUsers: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin/conversationLogs', () => ({
  adminConversationLogsAPI: {
    list: listConversationLogs,
    get: getConversationLog
  },
  default: {
    list: listConversationLogs,
    get: getConversationLog
  }
}))

vi.mock('@/api/admin/usage', () => ({
  adminUsageAPI: { searchUsers },
  default: { searchUsers }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const summary = {
  id: 101,
  request_id: 'request-101',
  session_id: 'session-101',
  user_id: 11,
  user_email: 'user@example.com',
  api_key_id: 7,
  api_key_name: 'analysis-key',
  group_id: 9,
  protocol: 'openai_chat_completions',
  endpoint: '/v1/chat/completions',
  model: 'gpt-test',
  stream: false,
  status: 'completed' as const,
  status_code: 200,
  content_type: 'application/json',
  request_bytes: 18,
  response_bytes: 20,
  request_truncated: false,
  response_truncated: false,
  duration_ms: 42,
  created_at: '2026-09-03T00:00:00Z',
  completed_at: '2026-09-03T00:00:00Z'
}

const otherUserSummary = {
  ...summary,
  id: 102,
  request_id: 'request-102',
  session_id: 'session-102',
  user_id: 12,
  user_email: 'other@example.com',
  api_key_id: 8,
  api_key_name: 'other-key'
}

function mountView() {
  return mount(ConversationLogsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Pagination: true,
        BaseDialog: {
          props: ['show', 'title'],
          template: '<div v-if="show" data-testid="detail"><slot /></div>'
        }
      }
    }
  })
}

describe('admin ConversationLogsView', () => {
  beforeEach(() => {
    listConversationLogs.mockReset()
    getConversationLog.mockReset()
    searchUsers.mockReset()
    showError.mockReset()
    listConversationLogs.mockResolvedValue({
      items: [summary, otherUserSummary],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getConversationLog.mockResolvedValue({
      ...summary,
      request_body: '{"messages":[{"role":"user","content":"hello"}]}',
      response_body: '{"choices":[{"message":{"content":"world"}}]}'
    })
    searchUsers.mockResolvedValue([
      { id: 12, email: 'other@example.com', deleted: false }
    ])
  })

  it('loads records and opens the stored request and response', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listConversationLogs).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20
    }))
    expect(wrapper.text()).toContain('user@example.com')
    expect(wrapper.text()).toContain('other@example.com')
    expect(wrapper.text()).toContain('gpt-test')
    expect(wrapper.text()).toContain('/v1/chat/completions')

    const viewButton = wrapper.findAll('button').find((button) => button.text() === 'common.view')
    expect(viewButton).toBeDefined()
    await viewButton?.trigger('click')
    await flushPromises()

    expect(getConversationLog).toHaveBeenCalledWith(101)
    expect(wrapper.get('[data-testid="detail"]').text()).toContain('hello')
    expect(wrapper.get('[data-testid="detail"]').text()).toContain('world')
    expect(showError).not.toHaveBeenCalled()
  })

  it('loads every user by default and filters after selecting a user', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listConversationLogs).toHaveBeenCalledWith(expect.objectContaining({
      user_id: undefined
    }))

    await wrapper.get('[data-testid="conversation-user-search"]').setValue('other')
    await new Promise((resolve) => setTimeout(resolve, 350))
    await flushPromises()

    expect(searchUsers).toHaveBeenCalledWith('other')
    const otherUser = wrapper.findAll('button').find((button) => button.text().includes('other@example.com'))
    expect(otherUser).toBeDefined()
    await otherUser?.trigger('click')
    await flushPromises()

    expect(listConversationLogs).toHaveBeenLastCalledWith(expect.objectContaining({
      user_id: 12
    }))
  })
})
