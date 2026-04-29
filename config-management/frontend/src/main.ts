import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { ArrowDown, Refresh, ArrowLeft, Plus } from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'

const app = createApp(App)

// 按需注册实际使用的图标，避免全量图标进入主包
app.component('ArrowDown', ArrowDown)
app.component('Refresh', Refresh)
app.component('ArrowLeft', ArrowLeft)
app.component('Plus', Plus)

app.use(createPinia())
app.use(router)

app.mount('#app')
