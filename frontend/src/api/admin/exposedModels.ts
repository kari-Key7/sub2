/**
 * Admin Exposed Models API
 * 按分组预览 GET /v1/models 实际对外返回的模型列表（口径与网关完全一致）。
 */

import { apiClient } from '../client'

/**
 * 列表来源：
 * - account_mapping：分组内可调度账号 model_mapping 的 key 并集；
 * - custom_list：分组「自定义 /v1/models 模型列表」过滤后的结果；
 * - platform_default：无任何账号映射，回落到平台内置默认列表。
 */
export type ExposedModelsSource = 'account_mapping' | 'custom_list' | 'platform_default'

export interface ExposedModelsGroup {
  id: number
  name: string
  platform: string
  rate_multiplier: number
  source: ExposedModelsSource
  /** 与 GET /v1/models 的 data[].id 顺序一致。 */
  models: string[]
}

export interface ExposedModelsResponse {
  groups: ExposedModelsGroup[]
}

/**
 * Get exposed models of every active group.
 */
export async function list(): Promise<ExposedModelsResponse> {
  const { data } = await apiClient.get<ExposedModelsResponse>('/admin/exposed-models')
  return data
}

export const exposedModelsAPI = {
  list
}

export default exposedModelsAPI
