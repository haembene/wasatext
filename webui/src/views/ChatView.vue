<script>
export default {
    data: function() {
        return {
            errormsg: null,
            loading: false,
            messages: [],
            newMessage: "",
            identifier: this.$route.params.id,
            partnerName: "Loading...", // Will hold the real name (e.g., "Amina")
        }
    },
    methods: {
        async loadDetails() {
            // We use the PUT endpoint just to get the user's Name and Photo
            // (It won't create duplicates, it just returns the info)
            try {
                let response = await this.$axios.put("/conversations/" + this.identifier, { name: "chat" });
                this.partnerName = response.data.Name;
            } catch (e) {
                this.partnerName = "User " + this.identifier;
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
                let response = await this.$axios.post("/messages", {
                    recipientId: this.identifier, 
                    content: this.newMessage
                });
                this.newMessage = "";
                this.refresh();
            } catch (e) {
                this.errormsg = "Error sending message: " + e.toString();
            }
        },
        formatTime(timestamp) {
            if (!timestamp) return "";
            return new Date(timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
        }
    },
    mounted() {
        this.loadDetails(); // Get the name (Amina)
        this.refresh();     // Get the messages
    }
}
</script>

<template>
    <div>
        <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
            <h1 class="h2">Chat with {{ partnerName }}</h1>
            <div class="btn-toolbar mb-2 mb-md-0">
                <button type="button" class="btn btn-sm btn-outline-secondary me-2" @click="refresh">
                    Refresh
                </button>
                <button type="button" class="btn btn-sm btn-outline-secondary" @click="$router.push('/')">
                    Back
                </button>
            </div>
        </div>

        <div v-if="errormsg" class="alert alert-danger" role="alert">
            {{ errormsg }}
        </div>

        <div class="chat-container mb-4" style="height: 60vh; overflow-y: scroll; border: 1px solid #ddd; padding: 15px; border-radius: 5px; display: flex; flex-direction: column-reverse;">
            <div v-if="!messages || messages.length === 0" class="text-center text-muted mt-5">
                No messages yet. Say hello!
            </div>

            <div v-for="msg in messages" :key="msg.ID" class="mb-2">
                <strong>{{ msg.SenderID === identifier ? partnerName : 'Me' }}:</strong> {{ msg.Content }}
                <span class="text-muted small ms-2">{{ formatTime(msg.Timestamp) }}</span>
            </div>
        </div>

        <div class="input-group">
            <input type="text" class="form-control" placeholder="Type a message..." v-model="newMessage" @keyup.enter="sendMessage">
            <button class="btn btn-primary" type="button" @click="sendMessage">Send</button>
        </div>
    </div>
</template>

<style>
/* Optional: Makes the chat scroll from bottom */
.chat-container {
    display: flex;
    flex-direction: column-reverse; 
}
</style>