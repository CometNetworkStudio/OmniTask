import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  { path: '/register', name: 'register', component: () => import('@/views/RegisterView.vue'), meta: { public: true } },
  { path: '/', name: 'tasks', component: () => import('@/views/TasksView.vue') },
  { path: '/tasks/:id', name: 'task-detail', component: () => import('@/views/TaskDetailView.vue') },
  { path: '/credentials', name: 'credentials', component: () => import('@/views/CredentialsView.vue') },
  { path: '/questions', name: 'questions', component: () => import('@/views/QuestionsView.vue') },
  { path: '/scripts', name: 'scripts', component: () => import('@/views/ScriptsView.vue'), meta: { requiresAdmin: true } },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

router.beforeEach((to) => {
  const token = localStorage.getItem('omnitask_token')
  const user = JSON.parse(localStorage.getItem('omnitask_user') || 'null')
  if (!to.meta.public && !token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.public && token) {
    return { name: 'tasks' }
  }
  if (to.meta.requiresAdmin && user?.role !== 'admin') {
    return { name: 'tasks' }
  }
  return true
})

export default router
