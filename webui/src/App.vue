<script setup>
import { RouterLink, RouterView } from 'vue-router'
</script>
<script>
export default {
    data() {
        return {
            username: localStorage.getItem("wasa_username") || "User"
        }
    },
    computed: {
        // Only show sidebar if not on login page
        showSidebar() {
            return this.$route.path !== '/login';
        }
    },
    watch: {
        $route() {
            this.username = localStorage.getItem("wasa_username") || "User";
        }
    },
    methods: {
        logout() {
            localStorage.removeItem("wasa_token");
            localStorage.removeItem("wasa_username");
            this.$router.push('/login');
        }
    }
}
</script>

<template>
    <header class="navbar navbar-dark sticky-top bg-dark flex-md-nowrap p-0 shadow" v-if="showSidebar">
        <a class="navbar-brand col-md-3 col-lg-2 me-0 px-3 fs-6" href="#/">WASAText</a>
        <div class="navbar-nav">
            <div class="nav-item text-nowrap">
                <a class="nav-link px-3" href="#" @click.prevent="logout">Sign out</a>
            </div>
        </div>
    </header>

    <div class="container-fluid">
        <div class="row">
            <nav id="sidebarMenu" class="col-md-3 col-lg-2 d-md-block bg-light sidebar collapse" v-if="showSidebar">
                <div class="position-sticky pt-3 sidebar-sticky">
                    <h6 class="sidebar-heading d-flex justify-content-between align-items-center px-3 mt-4 mb-1 text-muted text-uppercase">
                        <span>Profile: {{ username }}</span>
                    </h6>
                    <ul class="nav flex-column">
                        <li class="nav-item">
                            <RouterLink to="/" class="nav-link">
                                <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#message-square"/></svg>
                                Conversations
                            </RouterLink>
                        </li>

                        <li class="nav-item">
                            <RouterLink to="/profile" class="nav-link">
                                <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#user"/></svg>
                                My Profile
                            </RouterLink>
                        </li>
                    </ul>
                </div>
            </nav>

            <main :class="showSidebar ? 'col-md-9 ms-sm-auto col-lg-10 px-md-4' : 'col-12'">
                <RouterView />
            </main>
        </div>
    </div>
</template>

<style>
</style>