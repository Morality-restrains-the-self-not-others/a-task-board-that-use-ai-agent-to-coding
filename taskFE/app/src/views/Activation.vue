<template>
  <div class="activation-container">
    <div class="activation-card">
      <h1>账号激活</h1>
      
      <div v-if="!activationStatus" class="activation-form">
        <p>请点击下方按钮激活您的账号：</p>
        <button @click="activateAccount" class="activate-button" :disabled="isLoading">
          {{ isLoading ? '激活中...' : '激活账号' }}
        </button>
      </div>
      
      <div v-else-if="isLoading" class="loading-spinner">
        <div class="spinner"></div>
      </div>
      
      <div v-else-if="activationStatus === 'success'" class="success-message">
        <p>{{ activationMessage }}</p>
        <p>您现在可以使用您的账号登录我们的平台。</p>
        <router-link to="/auth/login/" class="login-button">前往登录</router-link>
      </div>
      
      <div v-else-if="activationStatus === 'error'" class="error-message">
        <p>{{ activationMessage }}</p>
        <p>请检查链接是否正确，或重新请求激活邮件。</p>
        <router-link to="/auth/login/" class="login-button">返回登录</router-link>
      </div>
    </div>
  </div>
</template>

<script>
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

// OPT-20260819-038: 激活账号是写操作，防连点/超时重试双发 POST
const activateAccountGuard = createClickGuard()

export default {
  name: 'Activation',
  data() {
    return {
      activationStatus: null,
      activationMessage: '',
      isLoading: false
    }
  },
  methods: {
    async activateAccount() {
      // OPT-20260819-038: 激活账号是写操作，防连点/超时重试双发 POST
      await activateAccountGuard.run(async ({ idempotencyKey }) => {
        try {
          this.isLoading = true
          // Get token from URL params
          const token = this.$route.params.token

          if (!token) {
            this.activationStatus = 'error'
            this.activationMessage = '激活链接无效'
            this.isLoading = false
            return
          }

          // Call activation API via POST method
          const response = await apiFetch(`/api/accounts/users/confirm_activation/${token}/`, {
            method: 'POST',
            headers: mergeIdempotencyHeaders(
              {
                'Content-Type': 'application/json'
              },
              idempotencyKey,
            ),
          })
          const data = await response.json()

          if (response.ok && data.status === 'success') {
            this.activationStatus = 'success'
            this.activationMessage = data.message
          } else {
            this.activationStatus = 'error'
            this.activationMessage = data.message || '激活失败'
          }
        } catch (error) {
          console.error('激活失败:', error)
          this.activationStatus = 'error'
          this.activationMessage = '激活失败，请稍后重试'
        } finally {
          this.isLoading = false
        }
      })
    }
  }
}
</script>

<style scoped>
.activation-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 80vh;
  padding: 20px;
  background-color: #f5f5f5;
}

.activation-card {
  background-color: white;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  padding: 40px;
  max-width: 500px;
  width: 100%;
  text-align: center;
}

h1 {
  margin-bottom: 20px;
  color: #333;
  font-size: 24px;
}

.loading-spinner {
  display: flex;
  justify-content: center;
  margin: 20px 0;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid #007bff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.success-message {
  background-color: #d4edda;
  color: #155724;
  padding: 20px;
  border-radius: 5px;
  margin: 20px 0;
}

.error-message {
  background-color: #f8d7da;
  color: #721c24;
  padding: 20px;
  border-radius: 5px;
  margin: 20px 0;
}

p {
  margin: 10px 0;
}

.activate-button {
  display: inline-block;
  background-color: #007bff;
  color: white;
  padding: 12px 24px;
  border: none;
  border-radius: 5px;
  font-size: 16px;
  cursor: pointer;
  transition: background-color 0.3s;
  margin-top: 20px;
}

.activate-button:hover {
  background-color: #0069d9;
}

.activate-button:disabled {
  background-color: #6c757d;
  cursor: not-allowed;
}

.login-button {
  display: inline-block;
  background-color: #007bff;
  color: white;
  padding: 10px 20px;
  text-decoration: none;
  border-radius: 5px;
  margin-top: 20px;
  transition: background-color 0.3s;
}

.login-button:hover {
  background-color: #0069d9;
}
</style>
