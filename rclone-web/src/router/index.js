import {createRouter, createWebHistory} from 'vue-router';
import Login from '../views/Login.vue';
import CreateBackup from '../views/CreateBackup.vue';
import Register from '../views/Register.vue';
const routes = [
    {path: '/', redirect: '/login'},
    {path: '/login', component: Login},
    {path: '/create-backup', component: CreateBackup},
    {path: '/register', component: Register},
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

export default router;
