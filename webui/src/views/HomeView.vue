<script>
export default {
    data: function() {
        return {
            errormsg: null,
            loading: false,
            conversations: [],
            searchQuery: "",
            searchResults: [],
        }
    },
    methods: {
        async refresh() {
            this.loading = true;
            this.errormsg = null;
            try {
                let response = await this.$axios.get("/conversations");
                this.conversations = response.data;
            } catch (e) {
                this.errormsg = e.toString();
            }
            this.loading = false;
        },
        async searchUsers() {
            if (this.searchQuery.length < 3) return;
            try {
                let response = await this.$axios.get("/users", {
                    params: { username: this.searchQuery }
                });
                this.searchResults = response.data;
            } catch (e) {
                this.errormsg = "Error searching users: " + e.toString();
            }
        },
        async openChat(targetUserID) {
            try {
                let response = await this.$axios.put("/conversations/" + targetUserID, {
                    name: "New Chat" 
                });
                let conversationId = response.data.ID || response.data.id;
                this.$router.push("/chat/" + conversationId);
            } catch (e) {
                this.errormsg = "Failed to start conversation: " + e.toString();
            }
        },
        formatTime(timestamp) {
            if (!timestamp) return "";
            return new Date(timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
        }
    },
    mounted() {
        this.refresh()
    }
}
</script>

<template>
    <div>
        <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
            <h1 class="h2">Conversations</h1>
            <div class="btn-toolbar mb-2 mb-md-0">
                <button type="button" class="btn btn-sm btn-outline-secondary" @click="refresh">
                    <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#refresh-cw"/></svg> Refresh
                </button>
            </div>
        </div>

        <ErrorMsg v-if="errormsg" :msg="errormsg"></ErrorMsg>

        <LoadingSpinner :loading="loading">
            <div class="mb-4">
                <div class="input-group">
                    <input type="text" class="form-control" placeholder="Search user to start chat..." v-model="searchQuery" @keyup.enter="searchUsers">
                    <button class="btn btn-primary" type="button" @click="searchUsers">Search</button>
                </div>
                <div v-if="searchResults && searchResults.length > 0" class="list-group mt-2">
                    <a href="#" class="list-group-item list-group-item-action" 
                       v-for="user in searchResults" :key="user.ID" 
                       @click.prevent="openChat(user.ID)">
                        Start chat with: <strong>{{ user.Name }}</strong>
                    </a>
                </div>
            </div>

            <div class="list-group">
                <div v-if="conversations.length === 0" class="text-center text-muted">
                    No conversations yet. Search for a user above!
                </div>

                <a href="#" 
                   class="list-group-item list-group-item-action d-flex gap-3 py-3" 
                   v-for="c in conversations" 
                   :key="c.ID"
                   @click.prevent="openChat(c.ID)"
                >
                    <img :src="c.PhotoUrl || 'https://via.placeholder.com/40'" alt="twbs" width="40" height="40" class="rounded-circle flex-shrink-0">
                    <div class="d-flex gap-2 w-100 justify-content-between">
                        <div>
                            <h6 class="mb-0">{{ c.Name }} <span v-if="c.isGroup" class="badge bg-secondary">Group</span></h6>
                            <p class="mb-0 opacity-75 text-truncate" style="max-width: 300px;">
                                {{ c.LatestMessage ? c.LatestMessage.Content : 'No messages' }}
                            </p>
                        </div>
                        <small class="opacity-50 text-nowrap">
                            {{ c.LatestMessage ? formatTime(c.LatestMessage.Timestamp) : '' }}
                            <span v-if="c.UnreadCount > 0" class="badge bg-danger ms-2">{{ c.UnreadCount }}</span>
                        </small>
                    </div>
                </a>
            </div>
        </LoadingSpinner>
    </div>
</template>

<style>
</style>