import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ExposedModelsView from '../ExposedModelsView.vue'
import type { ExposedModelsResponse } from '@/api/admin/exposedModels'

const { list } = vi.hoisted(() => ({
  list: vi.fn<[], Promise<ExposedModelsResponse>>()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { exposedModels: { list } }
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ cachedPublicSettings: null })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copied: { value: false }, copyToClipboard: vi.fn() })
}))

const response: ExposedModelsResponse = {
  groups: [
    { id: 1, name: 'vip', platform: 'antigravity', rate_multiplier: 1, source: 'account_mapping', models: ['gemini-3.1-pro-high', 'claude-opus-4-8'] },
    { id: 2, name: 'gpt', platform: 'openai', rate_multiplier: 1.5, source: 'platform_default', models: ['gpt-5.5'] }
  ]
}

function mountView() {
  return mount(ExposedModelsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: { template: '<span />' },
        PlatformIcon: { template: '<span />' }
      }
    }
  })
}

describe('ExposedModelsView', () => {
  beforeEach(() => {
    list.mockReset()
  })

  it('loads on mount and renders one card per group with its models', async () => {
    list.mockResolvedValue(response)
    const wrapper = mountView()
    await flushPromises()

    expect(list).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('vip')
    expect(wrapper.text()).toContain('gemini-3.1-pro-high')
    expect(wrapper.text()).toContain('gpt-5.5')
    expect(wrapper.findAll('[data-testid^="model-chip-"]')).toHaveLength(3)
  })

  it('shows the load-failed state when the request rejects', async () => {
    list.mockRejectedValue(new Error('boom'))
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('admin.exposedModels.loadFailed')
    expect(wrapper.findAll('[data-testid^="model-chip-"]')).toHaveLength(0)
  })

  it('re-fetches when refresh is clicked', async () => {
    list.mockResolvedValue(response)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="refresh"]').trigger('click')
    await flushPromises()

    expect(list).toHaveBeenCalledTimes(2)
  })
})
