import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '@/layouts/AdminLayout.vue'
import UserLayout from '@/layouts/UserLayout.vue'

const routes = [
  // 管理端登录
  {
    path: '/admin/login',
    name: 'AdminLogin',
    component: () => import('@/views/admin/AdminLogin.vue'),
    meta: { title: '管理端登录', requiresAuth: false, layout: 'admin' }
  },
  // 用户端登录
  {
    path: '/user/login',
    name: 'UserLogin',
    component: () => import('@/views/user/UserLogin.vue'),
    meta: { title: '用户登录', requiresAuth: false, layout: 'user' }
  },
  // 注册页面
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/Register.vue'),
    meta: { title: '注册', requiresAuth: false, layout: 'user' }
  },
  // 管理端路由
  {
    path: '/admin',
    component: AdminLayout,
    redirect: '/admin/dashboard',
    meta: { role: 'admin' },
    children: [
      {
        path: 'dashboard',
        name: 'AdminDashboard',
        component: () => import('@/views/admin/Dashboard.vue'),
        meta: { title: '仪表盘', icon: 'DashboardOutlined' }
      },
      {
        path: 'nodes',
        name: 'Nodes',
        component: () => import('@/views/admin/Nodes.vue'),
        meta: { title: '节点管理', icon: 'CloudServerOutlined' }
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/admin/Users.vue'),
        meta: { title: '用户管理', icon: 'TeamOutlined' }
      },
      {
        path: 'packages',
        name: 'Packages',
        component: () => import('@/views/admin/Packages.vue'),
        meta: { title: '套餐管理', icon: 'AppstoreOutlined' }
      },
      {
        path: 'orders',
        name: 'Orders',
        component: () => import('@/views/admin/Orders.vue'),
        meta: { title: '订单管理', icon: 'ShoppingOutlined' }
      },
      {
        path: 'tickets',
        name: 'Tickets',
        component: () => import('@/views/admin/Tickets.vue'),
        meta: { title: '工单管理', icon: 'MessageOutlined' }
      },
      {
        path: 'templates',
        name: 'Templates',
        component: () => import('@/views/admin/Templates.vue'),
        meta: { title: '节点模板', icon: 'FileTextOutlined' }
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/admin/Settings.vue'),
        meta: { title: '系统设置', icon: 'SettingOutlined' }
      },
      {
        path: 'announcements',
        name: 'Announcements',
        component: () => import('@/views/admin/Announcements.vue'),
        meta: { title: '公告管理', icon: 'NotificationOutlined' }
      },
      {
        path: 'email-settings',
        name: 'EmailSettings',
        component: () => import('@/views/admin/EmailSettings.vue'),
        meta: { title: '邮件配置', icon: 'MailOutlined' }
      }
    ]
  },
  // 用户端路由
  {
    path: '/user',
    component: UserLayout,
    redirect: '/user/nodes',
    meta: { role: 'user' },
    children: [
      {
        path: 'nodes',
        name: 'MyNodes',
        component: () => import('@/views/user/MyNodes.vue'),
        meta: { title: '我的节点', icon: 'CloudServerOutlined' }
      },
      {
        path: 'buy',
        name: 'BuyPackage',
        component: () => import('@/views/user/BuyPackage.vue'),
        meta: { title: '购买套餐', icon: 'ShoppingCartOutlined' }
      },
      {
        path: 'orders',
        name: 'MyOrders',
        component: () => import('@/views/user/MyOrders.vue'),
        meta: { title: '我的订单', icon: 'ShoppingOutlined' }
      },
      {
        path: 'traffic',
        name: 'MyTraffic',
        component: () => import('@/views/user/MyTraffic.vue'),
        meta: { title: '流量统计', icon: 'LineChartOutlined' }
      },
      {
        path: 'tickets',
        name: 'MyTickets',
        component: () => import('@/views/user/MyTickets.vue'),
        meta: { title: '我的工单', icon: 'MessageOutlined' }
      },
      {
        path: 'settings',
        name: 'UserSettings',
        component: () => import('@/views/user/Settings.vue'),
        meta: { title: '账户设置', icon: 'SettingOutlined' }
      }
    ]
  },
  // 默认重定向
  {
    path: '/',
    redirect: '/user/login'
  },
  // 兼容旧登录页
  {
    path: '/login',
    redirect: '/admin/login'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  document.title = to.meta.title ? `${to.meta.title} - NexCore代理主机` : 'NexCore代理主机'
  
  // 根据路由判断使用哪个 token
  const isAdminRoute = to.path.startsWith('/admin')
  const isUserRoute = to.path.startsWith('/user')
  
  const adminToken = localStorage.getItem('admin_token')
  const userToken = localStorage.getItem('user_token')
  
  // 管理端路由需要管理端 token
  if (isAdminRoute && to.meta.requiresAuth !== false && !adminToken) {
    next('/admin/login')
    return
  }
  
  // 用户端路由需要用户端 token
  if (isUserRoute && to.meta.requiresAuth !== false && !userToken) {
    next('/user/login')
    return
  }
  
  next()
})

export default router