<script>
export default {
    data: function() {
        return {
            loading: false,
            errormsg: null,
            successmsg: null,
            username: localStorage.getItem("wasa_username") || "",
            photoUrl: "", // base64 string
            currentPhotoUrl: "", // fetched dynamically
        }
    },
    async mounted() {
        await this.fetchMyProfile();
    },
    methods: {
        async fetchMyProfile() {
            try {
                let res = await this.$axios.get("/users", {
                    params: { username: this.username }
                });
                if (res.data && res.data.length > 0) {
                    const me = res.data.find(u => (u.id || u.ID) === localStorage.getItem("wasa_token"));
                    if (me && (me.photoUrl || me.PhotoUrl)) {
                        this.currentPhotoUrl = me.photoUrl || me.PhotoUrl;
                    }
                }
            } catch (e) {
                console.error("Could not fetch profile", e);
            }
        },
        async updateUsername() {
            this.loading = true;
            this.errormsg = null;
            this.successmsg = null;
            try {
                // Match the explicit backend router mapping for setMyUserName
                await this.$axios.put("/user/name", {
                    newName: this.username
                });
                
                // Update local storage so the sidebar updates too
                localStorage.setItem("wasa_username", this.username);
                this.successmsg = "Username updated successfully!";
                
            } catch (e) {
                this.errormsg = e.response ? e.response.data.message : e.toString();
            }
            this.loading = false;
        },
        
        async updatePhoto() {
            this.loading = true;
            this.errormsg = null;
            this.successmsg = null;
            try {
                // Match the explicit backend router mapping for setMyPhoto
                await this.$axios.put("/user/photo", {
                    photoUrl: this.photoUrl
                });
                
                this.successmsg = "Profile photo updated successfully!";
                this.currentPhotoUrl = this.photoUrl; // sync locally automatically
                
            } catch (e) {
                this.errormsg = e.response ? e.response.data.message : e.toString();
            }
            this.loading = false;
        },
        
        onFileSelected(event) {
            const file = event.target.files[0];
            if (!file) return;

            if (file.type === "image/heic" || file.name.toLowerCase().endsWith(".heic")) {
                alert("🚨 Browsers cannot display Apple .HEIC photos! Please stretch choose a .JPG or .PNG image instead!");
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
                this.photoUrl = b64;
            };
            reader.readAsDataURL(file);
        }
    }
}
</script>

<template>
    <div>
        <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
            <h1 class="h2">My Profile</h1>
        </div>

        <div v-if="successmsg" class="alert alert-success">
            {{ successmsg }}
        </div>

        <ErrorMsg v-if="errormsg" :msg="errormsg"></ErrorMsg>

        <LoadingSpinner :loading="loading">
            <div class="card mb-4">
                <div class="card-header">Change Username</div>
                <div class="card-body">
                    <form @submit.prevent="updateUsername">
                        <div class="mb-3">
                            <label for="usernameInput" class="form-label">New Username</label>
                            <input type="text" class="form-control" id="usernameInput" v-model="username" minlength="3" maxlength="16" required>
                        </div>
                        <button type="submit" class="btn btn-primary">Update Username</button>
                    </form>
                </div>
            </div>

            <div class="card">
                <div class="card-header d-flex align-items-center">
                    <img v-if="currentPhotoUrl" :src="currentPhotoUrl" class="rounded-circle me-3 shadow-sm border border-secondary border-opacity-25" style="width: 45px; height: 45px; object-fit: cover;" alt="Global Avatar">
                    <img v-else src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%23aaa'%3E%3Cpath d='M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z'/%3E%3C/svg%3E" class="rounded-circle me-3 shadow-sm border border-secondary border-opacity-25 bg-light p-1" style="width: 45px; height: 45px; object-fit: cover;" alt="Placeholder">
                    <span class="fs-5 fw-bold">Change Profile Photo</span>
                </div>
                <div class="card-body">
                    <form @submit.prevent="updatePhoto">
                        <div class="mb-3">
                            <label for="photoInput" class="form-label">Upload Profile Photo</label>
                            <input type="file" class="form-control" id="photoInput" accept="image/*" @change="onFileSelected">
                            <div class="form-text">Choose an image from your computer. We'll secure it entirely natively.</div>
                        </div>
                        <button type="submit" class="btn btn-primary">Update Photo</button>
                    </form>
                </div>
            </div>
        </LoadingSpinner>
    </div>
</template>

<style scoped>
</style>