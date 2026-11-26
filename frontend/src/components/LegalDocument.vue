<script setup>
defineProps({
  doc: {
    type: Object,
    required: true,
  },
})
</script>

<template>
  <article>
    <div class="text-headline-small mb-1">{{ doc.title }}</div>
    <div class="text-body-small text-medium-emphasis mb-4">版本 {{ doc.updated }}</div>
    <p class="text-body-medium mb-6">{{ doc.intro }}</p>

    <section v-for="section in doc.sections" :key="section.title" class="mb-6">
      <h3 class="text-title-medium mb-2">{{ section.title }}</h3>
      <p v-for="paragraph in section.paragraphs || []" :key="paragraph" class="text-body-medium mb-2">
        {{ paragraph }}
      </p>
      <ul v-if="section.items?.length" class="ps-6 mb-2">
        <li v-for="item in section.items" :key="item" class="text-body-medium mb-1">
          {{ item }}
        </li>
      </ul>
    </section>
  </article>
</template>
