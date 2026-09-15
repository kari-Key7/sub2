import type { ExposedModelsGroup } from '@/api/admin/exposedModels'

export interface ExposedModelsFilter {
  platform: string
  groupId: number | 'all'
  rate: number | 'all'
  /** 模型 ID 搜索词（大小写不敏感的子串匹配）。 */
  search: string
}

/**
 * 对外模型页的纯前端筛选：平台 / 分组 / 倍率按分组过滤，搜索词按模型 ID 过滤，
 * 分组内只留命中的模型，整组无命中则隐藏该分组。不修改入参。
 */
export function filterExposedGroups(
  groups: ExposedModelsGroup[],
  filter: ExposedModelsFilter
): ExposedModelsGroup[] {
  let result = groups
  if (filter.platform !== 'all') {
    result = result.filter((g) => g.platform === filter.platform)
  }
  if (filter.groupId !== 'all') {
    result = result.filter((g) => g.id === filter.groupId)
  }
  if (filter.rate !== 'all') {
    result = result.filter((g) => g.rate_multiplier === filter.rate)
  }
  const q = filter.search.trim().toLowerCase()
  if (q) {
    result = result
      .map((g) => ({ ...g, models: g.models.filter((m) => m.toLowerCase().includes(q)) }))
      .filter((g) => g.models.length > 0)
  }
  return result
}
