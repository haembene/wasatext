<script>
export default {
    data: function() {
        return {
            errormsg: null,
            loading: false,
            conversations: [],
            
            // Search Data
            searchQuery: "",
            searchResults: [],

            // Create Group Data
            showCreateGroup: false,
            newGroupName: "",
            
            refreshInterval: null,
        }
    },
    methods: {
        async refresh(silent = false) {
            if (!silent) this.loading = true;
            this.errormsg = null;
            try {
                let response = await this.$axios.get("/conversations");
                this.conversations = response.data;
            } catch (e) {
                if (!silent) this.errormsg = e.toString();
            }
            if (!silent) this.loading = false;
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

        async startNewChat(targetUserID) {
            try {
                // Ensure conversation exists for unfamiliar users
                let response = await this.$axios.post("/conversations/" + targetUserID, {
                    name: "New Chat" 
                });
                let conversationId = response.data.ID || response.data.id || response.data.conversationId;
                this.$router.push("/chat/" + conversationId);
            } catch (e) {
                this.errormsg = "Failed to start conversation: " + e.toString();
            }
        },

        openExistingChat(convoId) {
            this.$router.push("/chat/" + convoId);
        },

        async createGroup() {
            if (this.newGroupName.length < 3) {
                alert("Group name must be at least 3 characters.");
                return;
            }
            this.loading = true;
            try {
                // Call POST /groups
                let response = await this.$axios.post("/groups", {
                    name: this.newGroupName
                });
                
                // Get the new Group ID
                let groupId = response.data.ID || response.data.id || response.data.groupId;
                
                // Redirect to the chat view
                this.$router.push("/chat/" + groupId);
            } catch (e) {
                this.errormsg = "Failed to create group: " + (e.response ? e.response.data.message : e.toString());
            }
            this.loading = false;
        },

        formatTime(timestamp) {
            if (!timestamp) return "";
            return new Date(timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
        }
    },
    mounted() {
        this.refresh();
        this.refreshInterval = setInterval(() => {
            this.refresh(true);
        }, 1500);
    },
    beforeUnmount() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
    }
}
</script>

<template>
    <div>
        <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
            <h1 class="h2">Conversations</h1>
            <div class="btn-toolbar mb-2 mb-md-0">
                <button type="button" class="btn btn-sm btn-outline-primary me-2" @click="showCreateGroup = !showCreateGroup">
                    <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#plus-circle"/></svg> New Group
                </button>
                <button type="button" class="btn btn-sm btn-outline-secondary" @click="refresh">
                    <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#refresh-cw"/></svg> Refresh
                </button>
            </div>
        </div>

        <ErrorMsg v-if="errormsg" :msg="errormsg"></ErrorMsg>

        <LoadingSpinner :loading="loading">
            
            <div v-if="showCreateGroup" class="card mb-4 bg-light">
                <div class="card-body">
                    <h5 class="card-title">Create a New Group</h5>
                    <div class="input-group">
                        <input type="text" class="form-control" placeholder="Group Name (e.g. Study Buddies)" v-model="newGroupName" @keyup.enter="createGroup">
                        <button class="btn btn-success" type="button" @click="createGroup">Create</button>
                    </div>
                </div>
            </div>

            <div class="mb-4">
                <div class="input-group">
                    <input type="text" class="form-control" placeholder="Search user to start chat..." v-model="searchQuery" @keyup.enter="searchUsers">
                    <button class="btn btn-primary" type="button" @click="searchUsers">Search</button>
                </div>
                <div v-if="searchResults && searchResults.length > 0" class="list-group mt-2">
                    <a href="#" class="list-group-item list-group-item-action" 
                       v-for="user in searchResults" :key="user.id || user.ID" 
                       @click.prevent="startNewChat(user.id || user.ID)">
                        Start chat with: <strong>{{ user.name || user.Name }}</strong>
                    </a>
                </div>
            </div>

            <div class="list-group">
                <div v-if="conversations.length === 0" class="text-center text-muted">
                    No conversations yet. Search for a user or create a group!
                </div>

                <a href="#" 
                   class="list-group-item list-group-item-action d-flex gap-3 py-3" 
                   v-for="c in conversations" 
                   :key="c.id || c.ID"
                   @click.prevent="openExistingChat(c.id || c.ID)"
                >
                    <img :src="c.photoUrl || c.PhotoUrl || 'https://via.placeholder.com/40'" alt="pic" width="40" height="40" class="rounded-circle flex-shrink-0">
                    <div class="d-flex gap-2 w-100 justify-content-between">
                        <div>
                            <h6 class="mb-0">
                                {{ c.name || c.Name }} 
                                <span v-if="c.isGroup || c.IsGroup" class="badge bg-secondary ms-1">Group</span>
                            </h6>
                            <p class="mb-0 opacity-75 text-truncate" style="max-width: 300px;">
                                {{ c.latestMessage ? (c.latestMessage.content || c.latestMessage.Content) : 'No messages' }}
                            </p>
                        </div>
                        <small class="opacity-50 text-nowrap">
                            {{ c.latestMessage ? formatTime(c.latestMessage.timestamp || c.latestMessage.Timestamp) : '' }}
                            <span v-if="(c.unreadCount || c.UnreadCount) > 0" class="badge bg-danger ms-2">{{ c.unreadCount || c.UnreadCount }}</span>
                        </small>
                    </div>
                </a>
            </div>
        </LoadingSpinner>
    </div>
</template>

<style>
</style>