<script setup>
const headers = [
  { title: '脚本', key: 'name' },
  { title: '标识', key: 'id' },
  { title: '动作数', key: 'actions' },
  { title: '状态', key: 'status' },
]

const scripts = [
  { id: 'cidaren', name: '词达人', actions: 2, status: '内置' },
  { id: 'welearn', name: 'WeLearn', actions: 2, status: '内置' },
]
</script>

<template>
  <v-container>
    <v-row align="center" class="mb-2">
      <v-col>
        <div class="text-h5">脚本</div>
        <div class="text-medium-emphasis">管理员安装与管理平台脚本</div>
      </v-col>
      <v-col cols="auto">
        <v-btn color="primary" prepend-icon="mdi-github" disabled>从 GitHub 安装</v-btn>
      </v-col>
    </v-row>

    <v-alert type="info" class="mb-4">
      「从 GitHub 安装脚本」即将开放：填写符合开发规范的脚本仓库地址，审核后即可安装。
      规范见 <code>docs/integration-guide.md</code>。
    </v-alert>

    <v-card>
      <v-data-table :headers="headers" :items="scripts" item-value="id">
        <template #item.actions="{ item }">{{ item.actions }}</template>
        <template #item.status="{ item }">
          <v-chip size="small" color="success" variant="tonal">{{ item.status }}</v-chip>
        </template>
      </v-data-table>
    </v-card>
  </v-container>
</template>
