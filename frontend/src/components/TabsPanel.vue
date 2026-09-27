<script setup>
defineOptions({ inheritAttrs: false })

defineProps({
  // [{ value, title, icon? }]
  tabs: { type: Array, required: true },
  modelValue: { type: String, required: true },
})

const emit = defineEmits(['update:modelValue'])
</script>

<template>
  <div v-bind="$attrs">
    <v-btn-toggle
      :model-value="modelValue"
      mandatory
      variant="outlined"
      color="primary"
      rounded="xl"
      density="comfortable"
      class="mb-4 ga-2"
      @update:model-value="emit('update:modelValue', $event)"
    >
      <v-btn v-for="tab in tabs" :key="tab.value" :value="tab.value" :prepend-icon="tab.icon">
        {{ tab.title }}
      </v-btn>
    </v-btn-toggle>

    <v-window
      :model-value="modelValue"
      class="pa-2"
      @update:model-value="emit('update:modelValue', $event)"
    >
      <v-window-item v-for="tab in tabs" :key="tab.value" :value="tab.value">
        <slot :name="tab.value" />
      </v-window-item>
    </v-window>
  </div>
</template>
