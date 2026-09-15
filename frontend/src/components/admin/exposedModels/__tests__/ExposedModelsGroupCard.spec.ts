import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ExposedModelsGroupCard from '../ExposedModelsGroupCard.vue'
import type { ExposedModelsGroup } from '@/api/admin/exposedModels'

const copyToClipboard = vi.fn().mockResolvedValue(true)

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ cachedPublicSettings: null })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copied: { value: false }, copyToClipboard })
}))

function group(overrides: Partial<ExposedModelsGroup> = {}): ExposedModelsGroup {
  return {
    id: 7,
    name: 'vip',
    platform: 'antigravity',
    rate_multiplier: 1,
    source: 'account_mapping',
    models: ['gemini-3.1-pro-high', 'claude-opus-4-8'],
    ...overrides
  }
}

describe('ExposedModelsGroupCard', () => {
  beforeEach(() => {
    copyToClipboard.mockClear()
  })

  it('renders every exposed model id, the group name and the source badge', () => {
    const wrapper = mount(ExposedModelsGroupCard, { props: { group: group() } })

    expect(wrapper.text()).toContain('vip')
    expect(wrapper.text()).toContain('gemini-3.1-pro-high')
    expect(wrapper.text()).toContain('claude-opus-4-8')
    expect(wrapper.text()).toContain('admin.exposedModels.source.account_mapping')
    expect(wrapper.text()).toContain('admin.exposedModels.modelCount:{"count":2}')
  })

  it('copies a single model id when its chip is clicked', async () => {
    const wrapper = mount(ExposedModelsGroupCard, { props: { group: group() } })

    await wrapper.get('[data-testid="model-chip-claude-opus-4-8"]').trigger('click')

    expect(copyToClipboard).toHaveBeenCalledTimes(1)
    expect(copyToClipboard.mock.calls[0][0]).toBe('claude-opus-4-8')
  })

  it('copies all model ids one per line from the copy-all button', async () => {
    const wrapper = mount(ExposedModelsGroupCard, { props: { group: group() } })

    await wrapper.get('[data-testid="copy-all"]').trigger('click')

    expect(copyToClipboard).toHaveBeenCalledTimes(1)
    expect(copyToClipboard.mock.calls[0][0]).toBe('gemini-3.1-pro-high\nclaude-opus-4-8')
  })

  it('shows an empty hint and no copy-all button when the group exposes nothing', () => {
    const wrapper = mount(ExposedModelsGroupCard, { props: { group: group({ models: [] }) } })

    expect(wrapper.text()).toContain('admin.exposedModels.noModels')
    expect(wrapper.find('[data-testid="copy-all"]').exists()).toBe(false)
  })
})
