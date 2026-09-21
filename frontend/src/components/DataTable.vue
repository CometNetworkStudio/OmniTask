<script setup>
import { computed } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  headers: { type: Array, required: true },
  items: { type: Array, default: () => [] },
  loading: Boolean,
  total: { type: Number, default: 0 },
  page: { type: Number, default: 1 },
  size: { type: Number, default: 100 },
  sizeOptions: { type: Array, default: () => [10, 20, 50, 100] },
  itemValue: { type: String, default: 'id' },
  paginated: { type: Boolean, default: true },
})

const emit = defineEmits(['update:page', 'update:size'])

const pageCount = computed(() => Math.max(1, Math.ceil((props.total || 0) / (props.size || 1))))
</script>

<template>
  <v-data-table
    v-bind="$attrs"
    :headers="headers"
    :items="items"
    :loading="loading"
    :item-value="itemValue"
    :items-per-page="-1"
    hide-default-footer
    hover
  >
    <template v-for="(_, name) in $slots" #[name]="slotProps" :key="name">
      <slot :name="name" v-bind="slotProps || {}" />
    </template>
  </v-data-table>

  <v-row v-if="paginated && total > 0" align="center" justify="end" class="ma-0 pa-2">
    <v-col cols="auto" class="text-body-2 text-medium-emphasis">共 {{ total }} 条</v-col>
    <v-col cols="auto">
      <v-select
        :model-value="size"
        :items="sizeOptions"
        label="每页"
        density="compact"
        hide-details
        @update:model-value="emit('update:size', $event)"
      />
    </v-col>
    <v-col cols="auto">
      <v-pagination
        :model-value="page"
        :length="pageCount"
        :total-visible="5"
        density="compact"
        @update:model-value="emit('update:page', $event)"
      />
    </v-col>
  </v-row>
</template>
