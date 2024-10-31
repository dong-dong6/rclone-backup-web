import { createRouter, createWebHistory } from 'vue-router';
import Login from '../views/Login.vue';
import CreateBackup from '../views/CreateBackup.vue';

const routes = [
    { path: '/', redirect: '/login' },
    { path: '/login', component: Login },
    { path: '/create-backup', component: CreateBackup }
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

export default router;
