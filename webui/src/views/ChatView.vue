<script>
export default {
    data: function() {
        return {
            errormsg: null,
            loading: false,
            messages: [],
            newMessage: "",
            identifier: this.$route.params.id,
            
            // Conversation Details
            partnerName: "Loading...",
            isGroup: false, 
            
            // Group Management Data
            showGroupSettings: false,
            newGroupName: "",
            newGroupPhoto: "",
        }
    },
    methods: {
        async loadDetails() {
            try {
                // Fetch basic conversation info
                // We use PUT /conversations/... to ensure it exists or get details
                let response = await this.$axios.put("/conversations/" + this.identifier, { name: "chat" });
                this.partnerName = response.data.Name;
                
                // Check if it is a group (Backend should return this flag)
                // If your backend uses specific logic, adjust this check.
                // Assuming response.data.IsGroup (boolean)
                this.isGroup = response.data.IsGroup || false;
                
            } catch (e) {
                this.partnerName = "Chat " + this.identifier;
            }
        },
        
        async refresh() {
            this.loading = true;
            this.errormsg = null;
            try {
                let response = await this.$axios.get("/conversations/" + this.identifier);
                this.messages = response.data;
            } catch (e) {
                this.errormsg = e.toString();
            }
            this.loading = false;
        },
        
        async sendMessage() {
            if (this.newMessage.trim() === "") return;
            try {
                await this.$axios.post("/messages", {
                    recipientId: this.identifier, 
                    content: this.newMessage
                });
                this.newMessage = "";
                this.refresh();
            } catch (e) {
                this.errormsg = "Error sending message: " + e.toString();
            }
        },

        async deleteMessage(messageId) {
            if (!confirm("Are you sure you want to delete this message?")) return;
            try {
                await this.$axios.delete("/messages/" + messageId);
                this.refresh(); // Reload to see it gone
            } catch (e) {
                this.errormsg = "Error deleting message: " + e.toString();
            }
        },

        // --- Group Management Functions ---
        
        async updateGroupName() {
            if (!this.newGroupName) return;
            try {
                // Endpoint corresponding to set-group-name.go
                await this.$axios.put("/groups/" + this.identifier + "/name", {
                    newName: this.newGroupName
                });
                alert("Group name updated!");
                this.partnerName = this.newGroupName;
                this.newGroupName = "";
            } catch (e) {
                this.errormsg = "Failed to update group name: " + e.toString();
            }
        },

        async updateGroupPhoto() {
            if (!this.newGroupPhoto) return;
            try {
                // Endpoint corresponding to set-group-photo.go
                await this.$axios.put("/groups/" + this.identifier + "/photo", {
                    photoUrl: this.newGroupPhoto
                });
                alert("Group photo updated!");
                this.newGroupPhoto = "";
            } catch (e) {
                this.errormsg = "Failed to update group photo: " + e.toString();
            }
        },

        formatTime(timestamp) {
            if (!timestamp) return "";
            return new Date(timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
        }
    },
    mounted() {
        this.loadDetails(); 
        this.refresh();     
    }
}
</script>

<template>
    <div>
        <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
            <h1 class="h2">
                <span v-if="isGroup" class="badge bg-secondary me-2">Group</span>
                {{ partnerName }}
            </h1>
            <div class="btn-toolbar mb-2 mb-md-0">
                <button v-if="isGroup" type="button" class="btn btn-sm btn-outline-primary me-2" @click="showGroupSettings = !showGroupSettings">
                    {{ showGroupSettings ? 'Hide Settings' : 'Group Settings' }}
                </button>
                <button type="button" class="btn btn-sm btn-outline-secondary me-2" @click="refresh">
                    Refresh
                </button>
                <button type="button" class="btn btn-sm btn-outline-secondary" @click="$router.push('/')">
                    Back
                </button>
            </div>
        </div>

        <ErrorMsg v-if="errormsg" :msg="errormsg"></ErrorMsg>

        <div v-if="showGroupSettings && isGroup" class="card mb-3 bg-light">
            <div class="card-body">
                <h5 class="card-title">Group Settings</h5>
                <div class="row g-3">
                    <div class="col-md-6">
                        <div class="input-group">
                            <input type="text" class="form-control" placeholder="New Group Name" v-model="newGroupName">
                            <button class="btn btn-outline-primary" @click="updateGroupName">Update Name</button>
                        </div>
                    </div>
                    <div class="col-md-6">
                        <div class="input-group">
                            <input type="text" class="form-control" placeholder="New Photo URL" v-model="newGroupPhoto">
                            <button class="btn btn-outline-primary" @click="updateGroupPhoto">Update Photo</button>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="chat-container mb-4" style="height: 60vh; overflow-y: scroll; border: 1px solid #ddd; padding: 15px; border-radius: 5px; display: flex; flex-direction: column-reverse;">
            <div v-if="!messages || messages.length === 0" class="text-center text-muted mt-5">
                No messages yet. Say hello!
            </div>

            <div v-for="msg in messages" :key="msg.ID" class="mb-2 d-flex flex-column">
                <div :class="{'align-self-end text-end': msg.SenderID !== identifier, 'align-self-start': msg.SenderID === identifier}">
                    <div class="d-inline-block p-2 rounded" :class="msg.SenderID !== identifier ? 'bg-primary text-white' : 'bg-light border'">
                        <strong>{{ msg.SenderID === identifier ? partnerName : 'Me' }}:</strong> 
                        {{ msg.Content }}
                        
                        <button v-if="msg.SenderID !== identifier" 
                                class="btn btn-link btn-sm text-white p-0 ms-2" 
                                style="text-decoration: none;"
                                @click="deleteMessage(msg.ID)">
                            &times;
                        </button>
                    </div>
                    <div class="text-muted small mt-1">
                        {{ formatTime(msg.Timestamp) }}
                    </div>
                </div>
            </div>
        </div>

        <div class="input-group">
            <input type="text" class="form-control" placeholder="Type a message..." v-model="newMessage" @keyup.enter="sendMessage">
            <button class="btn btn-primary" type="button" @click="sendMessage">Send</button>
        </div>
    </div>
</template>

<style scoped>
.chat-container {
    display: flex;
    flex-direction: column-reverse; 
}
</style>