<script>
export default {
    data: function() {
        return {
            loading: false,
            errormsg: null,
            successmsg: null,
            username: localStorage.getItem("wasa_username") || "",
            photoUrl: "", // We will try to fetch the current one if possible
        }
    },
    methods: {
        async updateUsername() {
            this.loading = true;
            this.errormsg = null;
            this.successmsg = null;
            try {
                // Assuming the endpoint is PUT /settings/username
                // defined in your backend router for setMyUserName
                await this.$axios.put("/settings/username", {
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
                // Assuming the endpoint is PUT /settings/photo
                // defined in your backend router for setMyPhoto
                await this.$axios.put("/settings/photo", {
                    photoUrl: this.photoUrl
                });
                
                this.successmsg = "Profile photo updated successfully!";
                
            } catch (e) {
                this.errormsg = e.response ? e.response.data.message : e.toString();
            }
            this.loading = false;
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
                <div class="card-header">Change Profile Photo</div>
                <div class="card-body">
                    <form @submit.prevent="updatePhoto">
                        <div class="mb-3">
                            <label for="photoInput" class="form-label">Photo URL</label>
                            <input type="url" class="form-control" id="photoInput" v-model="photoUrl" placeholder="https://example.com/my-photo.jpg">
                            <div class="form-text">Paste a direct link to an image (JPG/PNG).</div>
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