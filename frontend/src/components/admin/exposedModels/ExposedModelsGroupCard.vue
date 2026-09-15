<template>
  <section
    class="overflow-hidden rounded-2xl border bg-white shadow-card dark:bg-dark-800/50"
    :class="[platformBorderStrongClass(group.platform)]"
  >
    <!-- 分组头部：名称/平台/倍率徽章 + 来源徽章 + 模型数 + 复制全部 -->
    <header
      class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700/60"
    >
      <div class="flex flex-wrap items-center gap-2">
        <GroupBadge
          :name="group.name"
          :platform="group.platform as GroupPlatform"
          :rate-multiplier="group.rate_multiplier"
          always-show-rate
        />
        <span
          class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium"
          :class="sourceBadgeClass"
          :title="t(`admin.exposedModels.sourceHint.${group.source}`)"
        >
          <Icon name="infoCircle" size="xs" class="h-3 w-3" />
          {{ t(`admin.exposedModels.source.${group.source}`) }}
        </span>
        <span class="text-xs text-gray-400 dark:text-dark-500">
          {{ t('admin.exposedModels.modelCount', { count: group.models.length }) }}
        </span>
      </div>
      <button
        v-if="group.models.length > 0"
        type="button"
        class="btn btn-secondary btn-sm"
        data-testid="copy-all"
        @click="copyAll"
      >
        <Icon name="copy" size="xs" class="h-3.5 w-3.5" />
        {{ t('admin.exposedModels.copyAll') }}
      </button>
    </header>

    <!-- 模型 ID 列表：顺序与 /v1/models 返回一致，点击即复制 -->
    <div class="px-5 py-4">
      <div v-if="group.models.length > 0" class="flex flex-wrap gap-2">
        <button
          v-for="model in group.models"
          :key="model"
          type="button"
          class="group inline-flex items-center gap-1.5 rounded-lg bg-gray-50 px-2.5 py-1.5 font-mono text-xs text-gray-700 ring-1 ring-inset ring-gray-200 transition hover:bg-primary-50 hover:text-primary-700 hover:ring-primary-200 dark:bg-dark-900/40 dark:text-dark-200 dark:ring-dark-700 dark:hover:bg-primary-500/10 dark:hover:text-primary-300 dark:hover:ring-primary-500/40"
          :data-testid="`model-chip-${model}`"
          :title="t('admin.exposedModels.copyOne')"
          @click="copyOne(model)"
        >
          {{ model }}
          <Icon name="copy" size="xs" class="h-3 w-3 opacity-0 transition group-hover:opacity-100" />
        </button>
      </div>
      <p v-else class="text-center text-sm text-gray-400 dark:text-dark-500">
        {{ t('admin.exposedModels.noModels') }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import { useClipboard } from '@/composables/useClipboard'
import { platformBorderStrongClass } from '@/utils/platformColors'
import type { ExposedModelsGroup, ExposedModelsSource } from '@/api/admin/exposedModels'
import type { GroupPlatform } from '@/types'

const props = defineProps<{
  group: ExposedModelsGroup
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const sourceBadgeClasses: Record<ExposedModelsSource, string> = {
  account_mapping: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400',
  custom_list: 'bg-sky-50 text-sky-600 dark:bg-sky-900/20 dark:text-sky-400',
  // 兜底列表通常意味着分组下没有可调度账号，用警示色提醒管理员
  platform_default: 'bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400'
}

const sourceBadgeClass = computed(
  () => sourceBadgeClasses[props.group.source] ?? sourceBadgeClasses.platform_default
)

function copyOne(model: string) {
  void copyToClipboard(model)
}

function copyAll() {
  void copyToClipboard(
    props.group.models.join('\n'),
    t('admin.exposedModels.copiedAll', { count: props.group.models.length })
  )
}
</script>
