<script setup lang="ts">
defineProps<{
  status: string
}>()

const statusConfig: Record<string, { color: string; bg: string; label: string }> = {
  online: {
    color: 'bg-emerald-500',
    bg: 'bg-emerald-100 dark:bg-emerald-900/30',
    label: '在线',
  },
  degraded: {
    color: 'bg-amber-500',
    bg: 'bg-amber-100 dark:bg-amber-900/30',
    label: '告警',
  },
  offline: {
    color: 'bg-red-500',
    bg: 'bg-red-100 dark:bg-red-900/30',
    label: '离线',
  },
  unknown: {
    color: 'bg-gray-400',
    bg: 'bg-gray-100 dark:bg-gray-700',
    label: '未知',
  },
}
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium"
    :class="statusConfig[status]?.bg || statusConfig.unknown.bg"
  >
    <span
      class="w-2 h-2 rounded-full inline-block"
      :class="statusConfig[status]?.color || statusConfig.unknown.color"
    />
    {{ statusConfig[status]?.label || status }}
  </span>
</template>
