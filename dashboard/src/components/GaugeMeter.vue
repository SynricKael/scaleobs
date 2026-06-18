<script setup lang="ts">
const props = defineProps<{
  value: number        // 0-100
  label: string
  unit?: string
  color?: string       // optional override
}>()

const unit = props.unit || '%'

// Arc sweep: 180° (half circle), start at bottom-left, sweep clockwise
const startAngle = 180  // degrees, 12 o'clock = -90
const sweepAngle = 180  // half circle
const cx = 50, cy = 50, r = 36

// Convert to SVG arc path
function polarToCartesian(cx: number, cy: number, r: number, deg: number) {
  const rad = ((deg - 90) * Math.PI) / 180
  return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad) }
}

function describeArc(cx: number, cy: number, r: number, start: number, end: number) {
  const s = polarToCartesian(cx, cy, r, end)
  const e = polarToCartesian(cx, cy, r, start)
  const large = end - start > 180 ? 1 : 0
  return `M ${s.x} ${s.y} A ${r} ${r} 0 ${large} 0 ${e.x} ${e.y}`
}

// Background arc: full 180°
const bgArc = describeArc(cx, cy, r, startAngle, startAngle + sweepAngle)
// Value arc
const valEnd = startAngle + (props.value / 100) * sweepAngle
const valArc = describeArc(cx, cy, r, startAngle, Math.min(valEnd, startAngle + sweepAngle))

// Needle angle
const needleAngle = startAngle + (props.value / 100) * sweepAngle
const needleLen = r * 0.75
const needleTip = polarToCartesian(cx, cy, needleLen, needleAngle)
const needleBase = polarToCartesian(cx, cy, -8, needleAngle + 180)

// Auto color
function autoColor(v: number): string {
  if (props.color) return props.color
  if (v > 85) return '#ef4444'   // red
  if (v > 65) return '#f59e0b'   // amber
  return '#22c55e'                // green
}
</script>

<template>
  <div class="flex flex-col items-center">
    <svg viewBox="0 0 100 65" class="w-full h-auto max-w-[100px]">
      <!-- Background arc -->
      <path :d="bgArc" fill="none" stroke="var(--border)" stroke-width="5" stroke-linecap="round" />

      <!-- Value arc -->
      <path
        :d="valArc"
        fill="none"
        :stroke="autoColor(value)"
        stroke-width="5"
        stroke-linecap="round"
        class="transition-all duration-700 ease-out"
      />

      <!-- Needle -->
      <line
        :x1="cx"
        :y1="cy"
        :x2="needleTip.x"
        :y2="needleTip.y"
        :stroke="autoColor(value)"
        stroke-width="1.8"
        stroke-linecap="round"
        class="transition-all duration-700 ease-out"
      />
      <!-- Center dot -->
      <circle :cx="cx" :cy="cy" r="2.5" fill="var(--text-secondary)" />

      <!-- Tick marks (every 20%) -->
      <line
        v-for="t in [0,20,40,60,80,100]"
        :key="t"
        :x1="polarToCartesian(cx, cy, r - 2, startAngle + (t/100)*sweepAngle).x"
        :y1="polarToCartesian(cx, cy, r - 2, startAngle + (t/100)*sweepAngle).y"
        :x2="polarToCartesian(cx, cy, r - 7, startAngle + (t/100)*sweepAngle).x"
        :y2="polarToCartesian(cx, cy, r - 7, startAngle + (t/100)*sweepAngle).y"
        stroke="var(--text-muted)"
        stroke-width="1"
      />
    </svg>

    <!-- Value text -->
    <span class="text-lg font-bold tabular-nums mt-0.5" :style="{ color: autoColor(value) }">
      {{ value.toFixed(1) }}<span class="text-xs font-normal opacity-60">{{ unit }}</span>
    </span>
    <span class="text-[10px] text-gray-400 dark:text-gray-500 uppercase tracking-wider">{{ label }}</span>
  </div>
</template>
