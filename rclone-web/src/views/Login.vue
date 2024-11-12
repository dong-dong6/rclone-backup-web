<template>
  <div class="login-container">
    <div class="login-box">
      <h2>登录</h2>
      <form @submit.prevent="login" class="login-form">
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
        <button type="submit" class="login-button">登录</button>
        <!-- 新增：没有账号？前往注册 -->
        <div class="additional-link">
          没有账号？<router-link to="/register">前往注册</router-link>
        </div>
      </form>

      <!-- 错误信息弹窗 -->
      <transition name="fade">
        <div v-if="errorMessage" class="error-popup">
          <p>{{ errorMessage }}</p>
          <button @click="errorMessage = ''" class="close-button">关闭</button>
        </div>
      </transition>
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
      errorMessage: '', // 错误消息
      errorTimeout: null // 定时器存储
    };
  },
  methods: {
    async login() {
      try {
        const response = await api.post('/login', {
          username: this.username,
          password: this.password
        });
        const token = response.data.token;
        localStorage.setItem('token', token);
        this.$router.push('/create-backup'); // 登录成功后跳转
      } catch (error) {
        // 显示错误消息
        this.showErrorMessage('登录失败，请检查用户名和密码。');
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
/* 背景：全屏线性渐变 */
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #74b9ff, #0984e3);
  padding: 20px;
  box-sizing: border-box;
}

/* 登录框：响应式设计 */
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

/* 标题样式 */
.login-box h2 {
  margin-bottom: 20px;
  color: #333;
  font-size: 1.5rem; /* 响应式字体 */
  font-weight: bold;
}

/* 表单组 */
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

/* 登录按钮 */
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

/* 错误弹窗 */
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

/* 关闭按钮 */
.close-button {
  background: none;
  border: none;
  color: white;
  font-weight: bold;
  font-size: 16px;
  cursor: pointer;
}

/* 响应式设计：小屏幕设备优化 */
@media (max-width: 768px) {
  .login-box {
    padding: 30px;  /* 缩小内边距 */
  }

  .login-box h2 {
    font-size: 1.25rem;  /* 缩小标题字体 */
  }

  .form-group input {
    padding: 10px;  /* 缩小输入框内边距 */
    font-size: 0.9rem;  /* 缩小输入框字体 */
  }

  .login-button {
    padding: 10px;  /* 缩小按钮内边距 */
    font-size: 0.9rem;  /* 缩小按钮字体 */
  }
}

@media (max-width: 480px) {
  .login-box {
    width: 90%;  /* 更加紧凑 */
  }
}
</style>

