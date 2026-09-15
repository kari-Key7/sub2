import { describe, expect, it } from 'vitest'
import { filterExposedGroups } from '../exposedModels'
import type { ExposedModelsGroup } from '@/api/admin/exposedModels'

function group(overrides: Partial<ExposedModelsGroup> = {}): ExposedModelsGroup {
  return {
    id: 1,
    name: 'vip',
    platform: 'antigravity',
    rate_multiplier: 1,
    source: 'account_mapping',
    models: ['gemini-3.1-pro-high', 'claude-opus-4-8'],
    ...overrides
  }
}

const groups: ExposedModelsGroup[] = [
  group({ id: 1, name: 'vip', platform: 'antigravity', rate_multiplier: 1 }),
  group({ id: 2, name: 'gpt', platform: 'openai', rate_multiplier: 1.5, models: ['gpt-5.5', 'gpt-5.4'] }),
  group({ id: 3, name: 'anthropic', platform: 'anthropic', rate_multiplier: 2, models: ['claude-opus-4-6'] })
]

describe('filterExposedGroups', () => {
  it('returns every group untouched when no filter is active', () => {
    const result = filterExposedGroups(groups, { platform: 'all', groupId: 'all', rate: 'all', search: '' })
    expect(result.map((g) => g.id)).toEqual([1, 2, 3])
    expect(result[0].models).toEqual(['gemini-3.1-pro-high', 'claude-opus-4-8'])
  })

  it('filters by platform, group and rate', () => {
    expect(
      filterExposedGroups(groups, { platform: 'openai', groupId: 'all', rate: 'all', search: '' }).map((g) => g.id)
    ).toEqual([2])
    expect(
      filterExposedGroups(groups, { platform: 'all', groupId: 3, rate: 'all', search: '' }).map((g) => g.id)
    ).toEqual([3])
    expect(
      filterExposedGroups(groups, { platform: 'all', groupId: 'all', rate: 1.5, search: '' }).map((g) => g.id)
    ).toEqual([2])
  })

  it('keeps only matching models on search and drops groups with no hit', () => {
    const result = filterExposedGroups(groups, { platform: 'all', groupId: 'all', rate: 'all', search: 'OPUS' })
    expect(result.map((g) => g.id)).toEqual([1, 3])
    expect(result[0].models).toEqual(['claude-opus-4-8'])
    expect(result[1].models).toEqual(['claude-opus-4-6'])
  })

  it('does not mutate the input groups when searching', () => {
    filterExposedGroups(groups, { platform: 'all', groupId: 'all', rate: 'all', search: 'gpt-5.5' })
    expect(groups[1].models).toEqual(['gpt-5.5', 'gpt-5.4'])
  })
})
