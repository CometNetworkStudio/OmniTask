<script setup>
defineProps({
  fields: { type: Array, default: () => [] },
  values: { type: Object, required: true },
})

function optionsOf(field) {
  return (field.options || []).map((option) =>
    typeof option === 'string' ? { title: option, value: option } : { title: option.label, value: option.value },
  )
}
</script>

<template>
  <template v-for="field in fields" :key="field.key">
    <v-switch
      v-if="field.type === 'switch'"
      v-model="values[field.key]"
      :label="field.label"
      color="primary"
      hide-details
    />
    <v-select
      v-else-if="field.type === 'select'"
      v-model="values[field.key]"
      :items="optionsOf(field)"
      :label="field.label + (field.required ? ' *' : '')"
      :placeholder="field.placeholder"
      :hint="field.help"
      :persistent-hint="!!field.help"
    />
    <v-textarea
      v-else-if="field.type === 'textarea' || field.multiline"
      v-model="values[field.key]"
      :label="field.label + (field.required ? ' *' : '')"
      :placeholder="field.placeholder"
      :hint="field.help"
      :persistent-hint="!!field.help"
      rows="3"
      auto-grow
    />
    <v-text-field
      v-else
      v-model="values[field.key]"
      :label="field.label + (field.required ? ' *' : '')"
      :placeholder="field.placeholder"
      :hint="field.help"
      :persistent-hint="!!field.help"
      :type="field.type === 'number' ? 'number' : 'text'"
      :min="field.min"
      :max="field.max"
      :step="field.step"
    />
  </template>
</template>
