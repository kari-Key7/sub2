<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-5 pb-8">
      <!-- 说明 + 刷新 -->
      <div class="flex flex-wrap items-start justify-between gap-3">
        <p class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
          <Icon name="infoCircle" size="xs" class="h-3.5 w-3.5 shrink-0" />
          <span>
            {{ t('admin.exposedModels.endpointHint', { endpoint: 'GET /v1/models' }) }}
          </span>
        </p>
        <button
          type="button"
          class="btn btn-secondary btn-sm shrink-0"
          :disabled="loading"
          data-testid="refresh"
          @click="load"
        >
          <Icon name="refresh" size="xs" class="h-3.5 w-3.5" :class="{ 'animate-spin': loading }" />
          {{ t('admin.exposedModels.refresh') }}
        </button>
      </div>

      <!-- 加载 / 错误 / 空 -->
      <div v-if="loading && !response" class="flex min-h-[240px] items-center justify-center">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-600/25 border-t-primary-600 dark:border-primary-400/25 dark:border-t-primary-400"
        ></div>
      </div>
      <div
        v-else-if="error"
        class="rounded-2xl border border-red-200 bg-red-50 px-5 py-8 text-center text-sm text-red-600 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
      >
        {{ t('admin.exposedModels.loadFailed') }}
      </div>
      <template v-else>
        <!-- 筛选区：平台 → 分组 → 倍率 → 模型搜索（复用模型广场的筛选栏） -->
        <PlazaFilterBar
          :platforms="platforms"
          :groups="groupOptions"
          :rates="rates"
          :platform="selectedPlatform"
          :group-id="selectedGroupId"
          :rate="selectedRate"
          :search="searchQuery"
          @update:platform="selectedPlatform = $event"
          @update:group-id="selectedGroupId = $event"
          @update:rate="selectedRate = $event"
          @update:search="searchQuery = $event"
        />

        <div v-if="filteredGroups.length > 0" class="space-y-5">
          <ExposedModelsGroupCard v-for="g in filteredGroups" :key="g.id" :group="g" />
        </div>
        <div
          v-else
          class="rounded-2xl border border-dashed border-gray-300 px-5 py-12 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
        >
          {{ searchActive ? t('admin.exposedModels.noSearchResult') : t('admin.exposedModels.empty') }}
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlazaFilterBar from '@/components/modelPlaza/PlazaFilterBar.vue'
import ExposedModelsGroupCard from '@/components/admin/exposedModels/ExposedModelsGroupCard.vue'
import { adminAPI } from '@/api/admin'
import type { ExposedModelsResponse } from '@/api/admin/exposedModels'
import { filterExposedGroups } from '@/utils/exposedModels'

const { t } = useI18n()

const response = ref<ExposedModelsResponse | null>(null)
const loading = ref(false)
const error = ref(false)

const selectedPlatform = ref<string>('all')
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const searchQuery = ref('')

const searchActive = computed(() => searchQuery.value.trim() !== '')

const groups = computed(() => response.value?.groups ?? [])

const platforms = computed(() =>
  [...new Set(groups.value.map((g) => g.platform).filter(Boolean))].sort()
)

const groupOptions = computed(() =>
  groups.value.map((g) => ({ id: g.id, name: g.name, platform: g.platform, rate: g.rate_multiplier }))
)

/** 全量倍率；当前组合下不可用的项由 FilterBar 置灰而非隐藏。 */
const rates = computed(() =>
  [...new Set(groups.value.map((g) => g.rate_multiplier))].sort((a, b) => a - b)
)

/** 数据刷新后选中的倍率/分组可能不复存在，重置为全部。 */
watch(groups, (list) => {
  if (selectedRate.value !== 'all' && !list.some((g) => g.rate_multiplier === selectedRate.value)) {
    selectedRate.value = 'all'
  }
  if (selectedGroupId.value !== 'all' && !list.some((g) => g.id === selectedGroupId.value)) {
    selectedGroupId.value = 'all'
  }
})

const filteredGroups = computed(() =>
  filterExposedGroups(groups.value, {
    platform: selectedPlatform.value,
    groupId: selectedGroupId.value,
    rate: selectedRate.value,
    search: searchQuery.value
  })
)

async function load() {
  loading.value = true
  error.value = false
  try {
    response.value = await adminAPI.exposedModels.list()
  } catch (err) {
    console.error('Failed to load exposed models:', err)
    error.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
