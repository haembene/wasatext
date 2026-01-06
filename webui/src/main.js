import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import axios from './services/axios.js';

// Import the components
import ErrorMsg from './components/ErrorMsg.vue'
import LoadingSpinner from './components/LoadingSpinner.vue'

import './assets/dashboard.css'
import './assets/main.css'

const app = createApp(App)

// Register axios globally
app.config.globalProperties.$axios = axios;

// Register components globally
app.component("ErrorMsg", ErrorMsg);
app.component("LoadingSpinner", LoadingSpinner);

app.use(router)
app.mount('#app')