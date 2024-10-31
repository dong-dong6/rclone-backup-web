import axios from 'axios';

const api = axios.create({
    // baseURL: 'http://10.34.67.257:628/api',
    baseURL: 'http://127.0.0.1:628/api',

    timeout: 10000,
});

api.interceptors.request.use((config) => {
    const token = localStorage.getItem('token');
    if (token) {
        config.headers['Authorization'] = `Bearer ${token}`;
    }
    return config;
}, (error) => Promise.reject(error));

api.interceptors.response.use((response) => response, (error) => {
    if (error.response && error.response.status === 401) {
        window.location.href = '/login';
    }
    return Promise.reject(error);
});

export default api;
