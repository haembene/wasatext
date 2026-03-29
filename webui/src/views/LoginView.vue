<script>
export default {
	data: function() {
		return {
			username: "",
			errormsg: null,
			loading: false
		}
	},
	methods: {
		async doLogin() {
			this.loading = true;
			this.errormsg = null;
			
			try {
				let response = await this.$axios.post("/session", {
					name: this.username
				});
				
				localStorage.setItem("wasa_token", response.data.identifier);
				localStorage.setItem("wasa_username", this.username);
				this.$router.push("/");
				
			} catch (e) {
				if (e.response) {
					this.errormsg = "Server Error: " + e.response.status + " " + e.response.statusText;
				} else if (e.request) {
					this.errormsg = "Network Error: Server is not reachable on port 3000.";
				} else {
					this.errormsg = e.toString();
				}
			}
			this.loading = false;
		}
	}
}
</script>

<template>
	<div class="d-flex align-items-center justify-content-center vh-100 bg-light">
		<div class="card p-4 shadow-sm" style="max-width: 400px; width: 100%">
			<h2 class="text-center mb-4">WASAText Login</h2>
			
			<form @submit.prevent="doLogin">
				<div class="mb-3">
					<label for="username" class="form-label">Username</label>
					<input 
						type="text" 
						class="form-control" 
						id="username" 
						v-model="username" 
						placeholder="e.g. Maria" 
						required 
					>
				</div>

				<ErrorMsg v-if="errormsg" :msg="errormsg" />

				<button type="submit" class="btn btn-primary w-100" :disabled="loading">
					Sign In
				</button>
			</form>
		</div>
	</div>
</template>

<style scoped>
</style>