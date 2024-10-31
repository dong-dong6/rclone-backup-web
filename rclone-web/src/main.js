import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
// 引入 Element Plus 和样式
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import vue3Cron from './util/crontab/index.js';
const app = createApp(App);
app.use(router)
    .use(ElementPlus)
    .use(vue3Cron)
    .mount('#app');
