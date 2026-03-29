<script>
export default {
    data: function() {
        return {
            errormsg: null,
            loading: false,
            messages: [],
            newMessage: "",
            identifier: this.$route.params.id,
            refreshInterval: null,
            
            // Reply & Media
            replyingTo: null,
            photoFile: null,
            photoPreviewData: null,
            myId: localStorage.getItem("wasa_token"),
            availableEmojis: ['👍', '❤️', '😂', '😮', '😢'],
            activeReactionMsgId: null,
            
            // Conversation Details
            partnerName: "Loading...",
            partnerPhotoUrl: "",
            isGroup: false, 
            
            // Group Management Data
            showGroupSettings: false,
            newGroupName: "",
            newGroupPhoto: "",
            newGroupMemberName: "",
            groupMembers: [],
            myConversations: [],
            forwardingMsg: null,
            forwardTarget: "",
        }
    },
    methods: {
        async loadDetails() {
            try {
                let convs = await this.$axios.get("/conversations");
                let dataArray = convs.data || [];
                this.myConversations = dataArray; 

                let activeConv = dataArray.find(c => (c.id || c.ID) === this.identifier);
                
                if (activeConv) {
                    this.partnerName = activeConv.name || activeConv.Name;
                    this.partnerPhotoUrl = activeConv.photoUrl || activeConv.PhotoUrl;
                    this.isGroup = activeConv.isGroup || activeConv.IsGroup || false;
                } else {
                    // Safe Fallback assumption for externally targeted links
                    this.isGroup = true;
                    this.partnerName = "Group Chat";
                    try {
                        let res = await this.$axios.get("/conversations/" + this.identifier);
                        if (res.status === 200) {
                            this.isGroup = false; 
                            this.partnerName = "Direct Chat";
                        }
                    } catch(e) { } // Ignore fallback validation failures safely
                }
                
                if (this.isGroup) {
                    await this.loadGroupMembers();
                }
            } catch (e) {
                console.error("loadDetails Exception:", e);
                this.partnerName = "Err: " + (e.message || "Unknown");
            }
        },
        
        async refresh(silent = false) {
            if (!this.identifier) return;
            if (!silent) this.loading = true;
            try {
                let response = await this.$axios.get("/conversations/" + this.identifier);
                this.messages = response.data;
                
                // Real-time metadata propagation (Step 6 constraints)
                let convs = await this.$axios.get("/conversations");
                this.myConversations = convs.data || [];
                let activeConv = this.myConversations.find(c => (c.id || c.ID) === this.identifier);
                if (activeConv) {
                    this.partnerName = activeConv.name || activeConv.Name;
                    this.partnerPhotoUrl = activeConv.photoUrl || activeConv.PhotoUrl;
                }
                
                if (this.isGroup) {
                    let membersRes = await this.$axios.get(`/groups/${this.identifier}/members`);
                    this.groupMembers = membersRes.data || [];
                }

                if (!silent) this.errormsg = "";
            } catch (e) {
                if (!silent) {
                    this.errormsg = e.toString();
                }
            }
            if (!silent) this.loading = false;
        },
        
        async sendMessage() {
            if (!this.newMessage.trim() && !this.photoFile) return;
            
            let finalContent = this.newMessage.trim();
            if (this.photoPreviewData) {
                finalContent = finalContent + "|PHOTO|" + this.photoPreviewData;
            }

            let payload = {
                recipientId: this.identifier,
                content: finalContent
            };
            
            if (this.replyingTo) {
                payload.replyTo = this.replyingTo.id || this.replyingTo.ID;
            }

            try {
                await this.$axios.post("/messages", payload);
                this.newMessage = "";
                this.removePhoto();
                this.replyingTo = null;
                this.refresh(true);
            } catch (e) {
                this.errormsg = "Error sending message: " + e.toString();
            }
        },
        onPhotoSelected(event) {
            const file = event.target.files[0];
            if (!file) return;
            this.photoFile = file;
            
            const reader = new FileReader();
            reader.onload = (e) => {
                const b64 = e.target.result;
                if (b64.startsWith("data:image/heic") || b64.startsWith("data:application/octet-stream")) {
                    alert("🚨 Browsers cannot display Apple .HEIC photos! Please choose a .JPG or .PNG image instead!");
                    return;
                }
                this.photoPreviewData = b64;
            };
            reader.readAsDataURL(file);
            event.target.value = ''; // Reset input to allow catching same file
        },
        removePhoto() {
            this.photoFile = null;
            this.photoPreviewData = null;
        },
        getQuoteSnippet(messageId) {
            const original = this.messages.find(m => (m.id || m.ID) === messageId);
            if (!original) return "Deleted message";
            const c = original.content || original.Content;
            if (c.includes('|PHOTO|')) {
                const text = c.split('|PHOTO|')[0];
                return text ? `📷 ${text.substring(0, 15)}...` : '📷 Photo';
            }
            return c.substring(0, 20) + (c.length > 20 ? '...' : '');
        },
        getReactorName(reactorId) {
            if (reactorId === this.myId) return "Me";
            if (reactorId === this.identifier) return this.partnerName;
            return "User"; // Fallback for groups
        },
        async toggleReaction(messageId, emoji) {
            const msg = this.messages.find(m => (m.id || m.ID) === messageId);
            if (!msg) return;
            const reactions = msg.reactions || msg.Reactions || [];
            
            // Check if I already reacted with THIS emoji
            const existingMyReaction = reactions.find(r => (r.reactorId || r.ReactorId || r.ReactorID) === this.myId && (r.emoticon || r.Emoticon) === emoji);
            
            try {
                if (existingMyReaction) {
                    await this.$axios.delete(`/messages/${messageId}/r/reactions?emoticon=${encodeURIComponent(emoji)}`);
                } else {
                    await this.$axios.post(`/messages/${messageId}/r/reactions`, { emoticon: emoji });
                }
                this.activeReactionMsgId = null;
                this.refresh(true); // silent refresh
            } catch (e) {
                console.error("Reaction toggle failed", e);
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
        
        async loadGroupMembers() {
            if (!this.isGroup) return;
            try {
                let res = await this.$axios.get(`/groups/${this.identifier}/members`);
                this.groupMembers = res.data || [];
            } catch (e) {
                console.error("Failed to load members", e);
                this.groupMembers = [];
            }
        },

        // --- Content Modifiers ---
        cleanContent(text) {
            if (!text) return "";
            return text.startsWith('|FORWARDED|') ? text.substring(11) : text;
        },

        // --- Forwarding Handlers ---
        async submitForward() {
            if (!this.forwardTarget || !this.forwardingMsg) return;
            try {
                await this.$axios.post('/forward-message', {
                    messageId: this.forwardingMsg.id || this.forwardingMsg.ID,
                    recipientId: this.forwardTarget
                });
                
                alert("Message forwarded successfully!");
                this.forwardingMsg = null;
                this.forwardTarget = "";
            } catch (e) {
                console.error("Failed to forward", e);
                alert("Could not forward message.");
            }
        },

        async addGroupMember() {
            if (this.newGroupMemberName.trim().length < 3) return;
            try {
                let res = await this.$axios.get("/users", { params: { username: this.newGroupMemberName } });
                if (res.data && res.data.length > 0) {
                    const targetUser = res.data[0];
                    await this.$axios.post(`/groups/${this.identifier}/members`, { userId: targetUser.id || targetUser.ID });
                    alert(`Added ${targetUser.name || targetUser.Name} to the group!`);
                    this.newGroupMemberName = "";
                } else {
                    alert("User not found by that name!");
                }
            } catch (e) {
                this.errormsg = "Failed to add member: " + (e.response ? e.response.statusText : e.toString());
            }
        },

        async leaveGroup() {
            if (!confirm("Are you sure you want to leave this group forever?")) return;
            try {
                await this.$axios.delete(`/groups/${this.identifier}/members/${this.myId}`);
                this.$router.push("/");
            } catch (e) {
                this.errormsg = "Failed to leave group: " + (e.response ? e.response.statusText : e.toString());
            }
        },
        
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

        onGroupPhotoSelected(event) {
            const file = event.target.files[0];
            if (!file) return;

            if (file.type === "image/heic" || file.name.toLowerCase().endsWith(".heic")) {
                alert("🚨 Browsers cannot display Apple .HEIC photos! Please choose a .JPG or .PNG image instead!");
                event.target.value = ""; // reset
                return;
            }

            const reader = new FileReader();
            reader.onload = (e) => {
                const b64 = e.target.result;
                if (b64.startsWith("data:image/heic") || b64.startsWith("data:application/octet-stream")) {
                    alert("🚨 Browsers cannot display Apple .HEIC photos! Please choose a .JPG or .PNG image instead!");
                    return;
                }
                this.newGroupPhoto = b64;
            };
            reader.readAsDataURL(file);
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
    async mounted() {
        await this.loadDetails(); 
        
        // Prevent concurrent API executions causing SQLite deadlocks
        await this.refresh();     
        
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
            <h1 class="h2 d-flex align-items-center">
                <span v-if="isGroup" class="badge bg-secondary me-2">Group</span>
                <img v-if="partnerPhotoUrl" :src="partnerPhotoUrl" class="rounded-circle me-3 shadow-sm border border-secondary border-opacity-25" style="width: 45px; height: 45px; object-fit: cover;" alt="Global Avatar">
                <img v-if="!partnerPhotoUrl" src="data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%23aaa'><path d='M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z'/></svg>" class="rounded-circle me-3 shadow-sm border border-secondary border-opacity-25 bg-light" style="width: 45px; height: 45px; object-fit: cover; padding: 5px;" alt="Placeholder">
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

        <div v-if="showGroupSettings && isGroup" class="card mb-3 bg-light border border-secondary border-opacity-25 shadow-sm">
            <div class="card-body">
                <h5 class="card-title text-primary">
                    <svg class="feather me-1"><use href="/feather-sprite-v4.29.0.svg#settings"/></svg> 
                    Group Settings
                </h5>
                <div class="row g-3">
                    <div class="col-md-6">
                        <label class="form-label small text-muted mb-1">Rename Group</label>
                        <div class="input-group input-group-sm">
                            <input type="text" class="form-control" placeholder="New Group Name" v-model="newGroupName">
                            <button class="btn btn-outline-primary fw-bold" @click="updateGroupName">Update Name</button>
                        </div>
                    </div>
                    <div class="col-md-6">
                        <label class="form-label small text-muted mb-1">Upload Group Photo</label>
                        <div class="input-group input-group-sm">
                            <input type="file" class="form-control" id="groupPhotoInput" accept="image/*" @change="onGroupPhotoSelected">
                            <button class="btn btn-outline-primary fw-bold" @click="updateGroupPhoto">Upload Photo</button>
                        </div>
                    </div>
                    
                    <hr class="my-3 opacity-25">

                    <div class="col-md-8">
                        <label class="form-label small text-muted mb-1 text-success fw-bold">Add New Member</label>
                        <div class="input-group input-group-sm">
                            <input type="text" class="form-control border-success" placeholder="Search exact username (e.g. John)" v-model="newGroupMemberName" @keyup.enter="addGroupMember">
                            <button class="btn btn-success fw-bold" @click="addGroupMember">Add User</button>
                        </div>
                    </div>
                    <div class="col-md-4 d-flex align-items-end justify-content-end">
                        <button class="btn btn-sm btn-outline-danger w-100 fw-bold" @click="leaveGroup">
                           Exit / Leave Group
                        </button>
                    </div>

                    <div class="col-12 mt-3 p-3 bg-white rounded border border-light shadow-sm">
                        <label class="form-label small text-muted mb-2 fw-bold text-uppercase" style="letter-spacing: 0.5px;">Active Members ({{ groupMembers.length }})</label>
                        <div class="d-flex flex-wrap gap-2">
                            <span v-for="member in groupMembers" :key="member.id || member.ID" class="badge bg-light text-dark border border-secondary border-opacity-25 px-3 py-2 rounded-pill d-flex align-items-center shadow-sm">
                                <img :src="member.photoUrl || member.PhotoUrl || `data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%23aaa'><path d='M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z'/></svg>`" class="rounded-circle me-2 bg-light p-1" style="width: 22px; height: 22px; object-fit: cover;">
                                <span class="fw-bold">{{ member.name || member.Name }}</span>
                                <span v-if="(member.id || member.ID) === myId" class="text-primary small ms-1 fst-italic">(Me)</span>
                            </span>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div style="position: relative; height: 60vh;" class="mb-4">
            
            <div class="chat-container h-100" style="overflow-y: scroll; border: 1px solid #ddd; padding: 15px; border-radius: 5px; display: flex; flex-direction: column-reverse; background-color: transparent;">
                <div v-if="!messages || messages.length === 0" class="text-center text-muted mt-5">
                    No messages yet. Say hello!
                </div>

            <div v-for="msg in messages" :key="msg.ID || msg.id" class="mb-2 d-flex flex-column">
                <div class="d-flex w-100" :class="(msg.senderId || msg.SenderID) === myId ? 'justify-content-end' : 'justify-content-start align-items-end'">
                    
                    <!-- Sender Avatar (Groups Only) -->
                    <img v-if="isGroup && (msg.senderId || msg.SenderID) !== myId" 
                         :src="(msg.senderPhoto || msg.SenderPhoto) || `data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%23aaa'><path d='M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z'/></svg>`" 
                         class="rounded-circle me-2 mb-1 shadow-sm border border-secondary border-opacity-25 bg-light p-1" 
                         style="width: 32px; height: 32px; object-fit: cover;" alt="Avatar">

                    <div :class="{'align-self-end text-end': (msg.senderId || msg.SenderID) === myId, 'align-self-start': (msg.senderId || msg.SenderID) !== myId}" style="max-width: 85%;">
                        <div class="d-inline-block p-2 rounded position-relative text-start w-100" :class="(msg.senderId || msg.SenderID) === myId ? 'bg-primary text-white' : 'bg-light border text-dark'">
                        <!-- Forwarded Marker -->
                        <div v-if="((msg.content || msg.Content) || '').startsWith('|FORWARDED|')" class="fst-italic small text-white opacity-75 mb-1 pb-1 border-bottom" :class="(msg.senderId || msg.SenderID) === myId ? 'border-light' : 'text-dark border-secondary'">
                            <span class="fs-6 fw-bold">➦</span> Forwarded message
                        </div>
                        
                        <!-- Quoted message snippet -->
                        <div v-if="msg.replyTo || msg.ReplyTo" class="bg-dark bg-opacity-25 rounded p-1 mb-2 text-white small border-start border-3 border-light opacity-75">
                            <strong>Replying to:</strong>
                            <span class="fst-italic ms-1">{{ getQuoteSnippet(msg.replyTo || msg.ReplyTo) }}</span>
                        </div>

                        <strong>{{ (msg.senderId || msg.SenderID) === myId ? 'Me' : (msg.senderName || msg.SenderName || 'User') }}:</strong> 
                        
                        <!-- Content Rendering -->
                        <div v-if="cleanContent(msg.content || msg.Content).includes('|PHOTO|')" class="mt-1" style="white-space: pre-wrap; word-break: break-word;">
                            <span v-if="cleanContent(msg.content || msg.Content).split('|PHOTO|')[0]">{{ cleanContent(msg.content || msg.Content).split('|PHOTO|')[0] }}</span>
                            <img :src="cleanContent(msg.content || msg.Content).split('|PHOTO|')[1]" class="img-fluid rounded mt-2 d-block mx-auto" style="max-height: 250px; object-fit: contain; max-width: 100%; width: auto;" alt="Image" />
                        </div>
                        <span v-else style="white-space: pre-wrap; word-break: break-word; display: inline-block;">{{ cleanContent(msg.content || msg.Content) }}</span>

                        <!-- Reactions Display -->
                        <div v-if="(msg.reactions || msg.Reactions) && (msg.reactions || msg.Reactions).length > 0" class="d-flex flex-wrap gap-1 mt-2">
                            <span v-for="react in (msg.reactions || msg.Reactions)" :key="(react.reactorId || react.ReactorId || react.ReactorID) + (react.emoticon || react.Emoticon)" 
                                  class="badge bg-light text-dark border border-secondary border-opacity-25 rounded-pill px-2 py-1 d-flex align-items-center gap-1 shadow-sm"
                                  style="font-size: 0.85em; cursor: pointer;"
                                  :title="'Reacted by ' + ((react.reactorId || react.ReactorId || react.ReactorID) === myId ? 'Me' : (react.reactorName || react.ReactorName || getReactorName(react.reactorId || react.ReactorId || react.ReactorID)))"
                                  @click.stop="toggleReaction(msg.id || msg.ID, react.emoticon || react.Emoticon)">
                                <span>{{ react.emoticon || react.Emoticon }}</span> 
                                <small class="opacity-75 ms-1 fw-normal" style="font-size: 0.8em;">{{ (react.reactorId || react.ReactorId || react.ReactorID) === myId ? 'Me' : (react.reactorName || react.ReactorName || getReactorName(react.reactorId || react.ReactorId || react.ReactorID)) }}</small>
                            </span>
                        </div>

                        <!-- Action buttons panel -->
                        <div class="d-flex align-items-center gap-3 mt-2 justify-content-end border-top pt-1 border-opacity-25" :class="(msg.senderId || msg.SenderID) !== identifier ? 'border-light' : 'border-secondary'">
                            
                            <!-- Reaction Picker -->
                            <div class="position-relative ms-0 me-auto">
                                <button class="btn btn-link btn-sm p-0 m-0 fw-bold" 
                                        :class="(msg.senderId || msg.SenderID) === myId ? 'text-white opacity-75' : 'text-secondary'"
                                        style="text-decoration: none; font-size: 12px; letter-spacing: 0.5px;"
                                        @click="activeReactionMsgId = activeReactionMsgId === (msg.id || msg.ID) ? null : (msg.id || msg.ID)">
                                    🙂 React
                                </button>
                                <div v-if="activeReactionMsgId === (msg.id || msg.ID)" 
                                     class="p-1 bg-white border rounded shadow-sm position-absolute bottom-100 z-3" 
                                     :class="(msg.senderId || msg.SenderID) === myId ? 'end-0' : 'start-0'"
                                     style="white-space: nowrap; margin-bottom: 2px;">
                                    <button v-for="emoji in availableEmojis" :key="emoji" 
                                            class="btn btn-sm btn-light border-0 p-1 mx-1" style="font-size: 1.2rem; transition: transform 0.1s;"
                                            @click.stop="toggleReaction(msg.id || msg.ID, emoji)">
                                        {{ emoji }}
                                    </button>
                                </div>
                            </div>
                            
                            
                            <button class="btn btn-link btn-sm p-0 m-0 fw-bold" 
                                    :class="(msg.senderId || msg.SenderID) === myId ? 'text-white opacity-75' : 'text-secondary'"
                                    style="text-decoration: none; font-size: 12px; letter-spacing: 0.5px;"
                                    @click="forwardingMsg = msg">
                                ➦ Fwd
                            </button>
                            <button class="btn btn-link btn-sm p-0 m-0 fw-bold" 
                                    :class="(msg.senderId || msg.SenderID) === myId ? 'text-white opacity-75' : 'text-secondary'"
                                    style="text-decoration: none; font-size: 12px; letter-spacing: 0.5px;"
                                    @click="replyingTo = msg">
                                ⤿ Reply
                            </button>
                            <button v-if="(msg.senderId || msg.SenderID) === myId" 
                                    class="btn btn-link btn-sm p-0 m-0 text-white opacity-75" 
                                    style="text-decoration: none; font-size: 16px; line-height: 1;"
                                    title="Delete Message"
                                    @click="deleteMessage(msg.id || msg.ID)">
                                &times;
                            </button>
                        </div>
                    </div>
                    <div class="text-muted small mt-1">
                        {{ formatTime(msg.timestamp || msg.Timestamp) }}
                        <span v-if="(msg.senderId || msg.SenderID) === myId" class="ms-1 fw-bold fs-6">
                            <span v-if="(msg.status !== undefined ? msg.status : msg.Status) === 0 || (msg.status !== undefined ? msg.status : msg.Status) === 1" class="text-secondary">✓</span>
                            <span v-if="(msg.status !== undefined ? msg.status : msg.Status) === 2" class="text-primary">✓✓</span>
                        </span>
                    </div>
                </div>
            </div>
            </div>
            </div>
            
            <!-- Popout Forwarding Overlay -->
            <div v-if="forwardingMsg" class="position-absolute bottom-0 start-0 w-100 p-3 bg-light text-dark d-flex flex-column z-3 shadow-lg border-top border-secondary border-opacity-50 rounded-bottom" style="box-shadow: 0 -4px 12px rgba(0,0,0,0.15) !important;">
                <div class="d-flex justify-content-between align-items-center mb-2">
                    <span class="fw-bold"><span class="fs-5 me-2 text-primary">➦</span> Forward Message to:</span>
                    <button class="btn-close btn-sm" @click="forwardingMsg = null" title="Cancel"></button>
                </div>
                <div class="input-group">
                    <select v-model="forwardTarget" class="form-select fw-bold shadow-none text-dark bg-white border-secondary">
                        <option value="" disabled>Select destination from your active chats...</option>
                        <option v-for="c in myConversations" :key="c.id || c.ID" :value="c.id || c.ID">
                            {{ c.name || c.Name }} {{ (c.isGroup || c.IsGroup) ? '(Group)' : '' }}
                        </option>
                    </select>
                    <button class="btn btn-primary fw-bold px-3 py-2" @click="submitForward" :disabled="!forwardTarget">SEND 🚀</button>
                </div>
            </div>
            
        </div>

        <!-- Chat Input Footer / Replying / Photo Preview Banners -->
        <div class="d-flex flex-column gap-2 mb-2 w-100 mt-2">
            <div v-if="replyingTo" class="alert alert-secondary py-1 px-3 m-0 d-flex justify-content-between align-items-center rounded shadow-sm">
                <div class="text-truncate small" style="max-width: 85%;">
                    <strong>Replying to:</strong> 
                    <span class="ms-1 fst-italic">{{ getQuoteSnippet(replyingTo.id || replyingTo.ID) }}</span>
                </div>
                <button class="btn-close btn-close-sm" type="button" aria-label="Close" @click="replyingTo = null" style="font-size: 0.6em;"></button>
            </div>

            <div v-if="photoPreviewData" class="position-relative d-inline-block rounded shadow-sm border p-1 bg-white" style="width: fit-content;">
                <img :src="photoPreviewData" alt="Preview" class="rounded" style="max-height: 80px; max-width: 150px; object-fit: contain;" />
                <button type="button" class="btn btn-danger position-absolute top-0 start-100 translate-middle rounded-circle shadow p-0" 
                        style="width: 20px; height: 20px; line-height: 1; font-size: 10px;" 
                        @click="removePhoto">✖</button>
            </div>
        </div>

        <form @submit.prevent="sendMessage" class="d-flex align-items-center gap-2 w-100 mb-2">
            <label class="btn btn-outline-secondary mb-0 p-2 d-flex align-items-center justify-content-center" title="Attach Photo" style="cursor: pointer; width: 40px; height: 40px;">
                <span style="font-size: 1.2rem;">📎</span>
                <input type="file" class="d-none" accept="image/*" @change="onPhotoSelected" />
            </label>

            <input type="text" v-model="newMessage" class="form-control" placeholder="Type a message..." :required="!photoFile">
            <button class="btn btn-primary fw-bold" type="submit" :disabled="!newMessage.trim() && !photoFile">Send</button>
        </form>
    </div>
</template>

<style scoped>
.chat-container {
    display: flex;
    flex-direction: column-reverse; 
}
</style>