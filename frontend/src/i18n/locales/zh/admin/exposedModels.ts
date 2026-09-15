export default {
  exposedModels: {
    title: '对外模型',
    description: '按分组预览 GET /v1/models 实际返回的模型列表，即客户端拉取到的对外模型',
    refresh: '刷新',
    loadFailed: '加载对外模型失败',
    empty: '暂无活跃分组',
    noSearchResult: '没有匹配的模型',
    modelCount: '{count} 个模型',
    copyAll: '复制全部',
    copyOne: '点击复制模型 ID',
    copiedAll: '已复制 {count} 个模型 ID',
    noModels: '该分组不会返回任何模型',
    endpointHint: '结果由网关的同一份逻辑计算，与用该分组 API Key 请求 {endpoint} 得到的 id 列表一致',
    source: {
      account_mapping: '账号映射',
      custom_list: '自定义列表',
      platform_default: '平台默认'
    },
    sourceHint: {
      account_mapping: '分组内可调度账号「模型映射」的 key 并集',
      custom_list: '分组「自定义 /v1/models 模型列表」按可用模型过滤后的结果',
      platform_default: '分组下没有任何账号映射，回落到平台内置默认列表'
    }
  }
}
