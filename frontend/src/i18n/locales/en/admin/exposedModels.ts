export default {
  exposedModels: {
    title: 'Exposed Models',
    description: 'Preview, per group, the model list that GET /v1/models actually returns — what clients see',
    refresh: 'Refresh',
    loadFailed: 'Failed to load exposed models',
    empty: 'No active groups',
    noSearchResult: 'No matching models',
    modelCount: '{count} models',
    copyAll: 'Copy all',
    copyOne: 'Click to copy model ID',
    copiedAll: 'Copied {count} model IDs',
    noModels: 'This group returns no models',
    endpointHint: 'Computed by the same gateway logic, so it matches the id list returned by {endpoint} for a key in this group',
    source: {
      account_mapping: 'Account mapping',
      custom_list: 'Custom list',
      platform_default: 'Platform default'
    },
    sourceHint: {
      account_mapping: 'Union of model-mapping keys across the group\'s schedulable accounts',
      custom_list: 'The group\'s custom /v1/models list, filtered to what is actually available',
      platform_default: 'No account mapping in this group, so the platform\'s built-in default list is returned'
    }
  }
}
