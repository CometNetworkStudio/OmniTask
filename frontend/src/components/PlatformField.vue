<script setup>
import { computed } from 'vue'
import { parseChip, parseProgress, parseSparkline } from '@/utils/taskParams'

const props = defineProps({
  value: { type: [String, Number], default: '' },
  type: { type: String, default: 'text' },
})

const chip = computed(() => parseChip(props.value))
</script>

<template>
  <v-progress-linear
    v-if="type === 'progress'"
    :model-value="parseProgress(value)"
    height="6"
    rounded
    color="primary"
    class="my-2"
  />
  <v-chip v-else-if="type === 'chip'" size="small" variant="tonal" :color="chip.color">
    {{ chip.text }}
  </v-chip>
  <v-sparkline
    v-else-if="type === 'sparkline'"
    :model-value="parseSparkline(value)"
    color="primary"
    height="24"
    :padding="4"
    smooth
  />
  <span v-else>{{ value }}</span>
</template>
