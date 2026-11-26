import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useMetaStore } from '@/stores/meta'

const routes = [
  { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  { path: '/register', name: 'register', component: () => import('@/views/RegisterView.vue'), meta: { public: true } },
  { path: '/home', name: 'home', component: () => import('@/views/HomeView.vue') },
  { path: '/tasks', name: 'tasks', component: () => import('@/views/TasksView.vue') },
  { path: '/tasks/:id', name: 'task-detail', component: () => import('@/views/TaskDetailView.vue') },
  { path: '/platforms/:id', name: 'platform', component: () => import('@/views/PlatformView.vue') },
  { path: '/credentials', name: 'credentials', component: () => import('@/views/CredentialsView.vue') },
  { path: '/questions', name: 'questions', component: () => import('@/views/QuestionsView.vue') },
  { path: '/settings', name: 'settings', component: () => import('@/views/SettingsView.vue') },
  { path: '/scripts', name: 'scripts', component: () => import('@/views/ScriptsView.vue'), meta: { requiresAdmin: true } },
  { path: '/scripts/:id', name: 'script-detail', component: () => import('@/views/ScriptDetailView.vue'), meta: { requiresAdmin: true } },
  { path: '/users', name: 'users', component: () => import('@/views/UsersView.vue'), meta: { requiresAdmin: true } },
  { path: '/security', redirect: { path: '/settings', query: { tab: 'admin-security' } } },
  { path: '/', redirect: '/home' },
  { path: '/:pathMatch(.*)*', redirect: '/home' },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  const meta = useMetaStore()
  await Promise.all([auth.bootstrap(), meta.load()])

  if (to.meta.public && auth.isAuthenticated) {
    return { name: 'home' }
  }
  if (!to.meta.public && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'register' && meta.registration === 'closed') {
    return { name: 'login' }
  }
  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return { name: 'home' }
  }
  return true
})

export default router
