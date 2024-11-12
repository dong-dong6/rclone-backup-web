<template>
  <div class="login-container">
    <div class="login-box">
      <h2>注册</h2>
      <form @submit.prevent="register" class="login-form">
        <div class="form-group">
          <label for="username">用户名：</label>
          <input
              id="username"
              v-model="username"
              type="text"
              required
              placeholder="请输入用户名"
          />
        </div>
        <div class="form-group">
          <label for="password">密码：</label>
          <input
              id="password"
              v-model="password"
              type="password"
              required
              placeholder="请输入密码"
          />
        </div>
        <div class="form-group">
          <label for="confirmPassword">确认密码：</label>
          <input
              id="confirmPassword"
              v-model="confirmPassword"
              type="password"
              required
              placeholder="请再次输入密码"
          />
        </div>
        <button type="submit" class="login-button">注册</button>
      </form>

      <!-- 错误信息弹窗 -->
      <transition name="fade">
        <div v-if="errorMessage" class="error-popup">
          <p>{{ errorMessage }}</p>
          <button @click="closeErrorPopup" class="close-button">关闭</button>
        </div>
      </transition>

      <!-- 已有账号，返回登录 -->
      <div class="additional-link">
        已有账号？<router-link to="/login">立即登录</router-link>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api';

export default {
  data() {
    return {
      username: '',
      password: '',
      confirmPassword: '',
      errorMessage: '',
      errorTimeout: null
    };
  },
  methods: {
    async register() {
      if (this.password !== this.confirmPassword) {
        this.showErrorMessage('两次输入的密码不一致。');
        return;
      }

      try {
        const response = await api.post('/register', {
          username: this.username,
          password: this.password
        });
        // 注册成功，跳转到登录页面
        this.$router.push('/login');
      } catch (error) {
        // 显示错误消息
        this.showErrorMessage(error.response.data || '注册失败，请重试。');
      }
    },
    showErrorMessage(message) {
      this.errorMessage = message;

      // 清除之前的定时器，避免冲突
      if (this.errorTimeout) {
        clearTimeout(this.errorTimeout);
      }

      // 设置新的定时器，在 3 秒后自动隐藏弹窗
      this.errorTimeout = setTimeout(() => {
        this.errorMessage = '';
      }, 3000);
    },
    closeErrorPopup() {
      // 手动关闭错误弹窗
      this.errorMessage = '';
      if (this.errorTimeout) {
        clearTimeout(this.errorTimeout); // 清除定时器
      }
    }
  },
  beforeDestroy() {
    // 在组件销毁前清除定时器，防止内存泄漏
    if (this.errorTimeout) {
      clearTimeout(this.errorTimeout);
    }
  }
};
</script>

<style scoped>
/* 样式与登录页面保持一致 */
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #74b9ff, #0984e3);
  padding: 20px;
  box-sizing: border-box;
}

.login-box {
  background-color: #fff;
  padding: 40px;
  border-radius: 12px;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  width: 100%;
  max-width: 400px;
  box-sizing: border-box;
  text-align: center;
}

.login-box h2 {
  margin-bottom: 20px;
  color: #333;
  font-size: 1.5rem;
  font-weight: bold;
}

.form-group {
  margin-bottom: 15px;
  text-align: left;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
  color: #555;
  font-size: 0.9rem;
}

.form-group input {
  width: 100%;
  padding: 12px;
  border: 1px solid #ddd;
  border-radius: 8px;
  font-size: 1rem;
  box-sizing: border-box;
  transition: border-color 0.3s;
}

.form-group input:focus {
  border-color: #0984e3;
  outline: none;
  box-shadow: 0 0 8px rgba(9, 132, 227, 0.3);
}

.login-button {
  width: 100%;
  padding: 12px;
  background-color: #0984e3;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: bold;
  cursor: pointer;
  transition: background-color 0.3s, transform 0.1s;
  margin-top: 10px;
}

.login-button:hover {
  background-color: #74b9ff;
}

.login-button:active {
  transform: scale(0.98);
}

.error-popup {
  position: fixed;
  top: 20px;
  right: 20px;
  background-color: #e74c3c;
  color: white;
  padding: 15px 20px;
  border-radius: 8px;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.close-button {
  background: none;
  border: none;
  color: white;
  font-weight: bold;
  font-size: 16px;
  cursor: pointer;
}

/* 额外的链接样式 */
.additional-link {
  margin-top: 15px;
  font-size: 0.9rem;
  color: #555;
}

.additional-link a {
  color: #0984e3;
  text-decoration: none;
}

.additional-link a:hover {
  text-decoration: underline;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .login-box {
    padding: 30px;
  }

  .login-box h2 {
    font-size: 1.25rem;
  }

  .form-group input {
    padding: 10px;
    font-size: 0.9rem;
  }

  .login-button {
    padding: 10px;
    font-size: 0.9rem;
  }
}

@media (max-width: 480px) {
  .login-box {
    width: 90%;
  }
}
</style>
