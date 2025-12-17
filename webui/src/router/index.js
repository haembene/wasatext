import {createRouter, createWebHashHistory} from 'vue-router'
import HomeView from '../views/HomeView.vue'
import LoginView from '../views/LoginView.vue'
import ChatView from '../views/ChatView.vue'

const router = createRouter({
	history: createWebHashHistory(import.meta.env.BASE_URL),
	routes: [
        {
            path: '/',
            name: 'home',
            component: HomeView
        },
        {
            path: '/login',
            name: 'login',
            component: LoginView
        },
        {
            path: '/chat/:id',  
            name: 'chat',
            component: ChatView
        }
	]
})

// Navigation Guard: Check if user is authenticated
router.beforeEach((to, from, next) => {
	const publicPages = ['/login'];
	const authRequired = !publicPages.includes(to.path);
	const loggedIn = localStorage.getItem('wasa_token');

	if (authRequired && !loggedIn) {
		next('/login');
	} else {
		next();
	}
});

export default router