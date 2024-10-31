<template>
  <div class="container">
    <el-card class="box-card">
      <div slot="header" class="header">
        <span>创建备份任务</span>
      </div>
      <el-form @submit.prevent="createBackupTask" label-width="150px">
        <el-form-item label="任务名称" class="centered-label">
          <el-input v-model="taskName" placeholder="请输入任务名称" />
        </el-form-item>

        <el-form-item label="文件夹路径" class="centered-label">
          <el-input v-model="sourceDir" placeholder="请输入文件夹路径" />
        </el-form-item>

        <div v-for="(remote, index) in rcloneRemote" :key="index" class="inline-item">
          <el-form-item :label="`Rclone配置名称 ${index + 1}`" class="centered-label input-with-button">
            <el-input v-model="rcloneRemote[index]" placeholder="请输入 Rclone 配置名称" />
            <el-button
                v-if="rcloneRemote.length > 1"
                type="text"
                class="delete-button"
                @click="removeRemote(index)"
                icon="el-icon-delete"
            >X</el-button>
          </el-form-item>
        </div>

        <el-form-item class="centered-label">
          <el-button type="primary" icon="el-icon-plus" @click="addRemote">
            添加新的 Rclone 配置名称
          </el-button>
        </el-form-item>

        <el-form-item label="最大留存备份数量" class="centered-label">
          <el-input-number v-model="maxBackups" :min="1" />
        </el-form-item>

        <el-form-item label="是否分卷压缩" class="centered-label">
          <el-checkbox v-model="isSplit" />
        </el-form-item>

        <el-form-item label="是否加密压缩" class="centered-label">
          <el-checkbox v-model="isEncrypted" />
        </el-form-item>

        <el-form-item label="加密密码" v-if="isEncrypted" class="centered-label">
          <el-input v-model="encryptionPassword" placeholder="输入加密密码" show-password />
        </el-form-item>

        <el-form-item label="定时规则" class="centered-label">
          <div class="cron">
            <el-popover v-model:visible="cronPopover" width="700px" trigger="click">
              <vue3Cron
                  @change="changeCron"
                  @close="togglePopover(false)"
                  max-height="400px"
                  i18n="cn"
              ></vue3Cron>
              <template #reference>
                <el-input
                    @focus="togglePopover(true)"
                    v-model="cronSchedule"
                    placeholder="* * * * * ? *"
                ></el-input>
              </template>
            </el-popover>
          </div>
        </el-form-item>

        <el-form-item class="button-group">
          <el-button type="primary" native-type="submit" icon="el-icon-check">
            提交
          </el-button>
          <el-button @click="resetForm" type="default" icon="el-icon-refresh-right">
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.container {
  max-width: 600px;
  margin: auto;
  padding: 20px;
}

.header {
  font-size: 1.5em;
  font-weight: bold;
  color: #333;
  text-align: center;
}

.box-card {
  border-radius: 10px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.centered-label .el-form-item__label {
  text-align: center;
}

.inline-item {
  display: flex;
  align-items: center;
}

.input-with-button {
  display: flex;
  align-items: center;
}

.input-with-button .el-input {
  flex: 1;
}

.delete-button {
  color: #ff4d4f;
  margin-left: 8px;
}

.button-group {
  text-align: center;
}

.button-group .el-button {
  margin: 0 10px;
}
</style>

<script>
import api from '../api';

export default {
  data() {
    return {
      sourceDir: '',
      rcloneRemote: [''],
      taskName: '',
      isSplit: false,
      isEncrypted: false,
      maxBackups: '',
      encryptionPassword: '',
      successMessage: '',
      cronSchedule: '',
      errorMessage: '',
      cronPopover: false,
    };
  },
  methods: {
    async createBackupTask() {
      try {
        const response = await api.post('/create_backup_task', {
          sourceDir: this.sourceDir,
          rcloneRemote: this.rcloneRemote,
          isSplit: this.isSplit,
          taskName: this.taskName,
          maxBackups: this.maxBackups,
          isEncrypted: this.isEncrypted,
          cronSchedule: this.cronSchedule,
          encryptionPassword: this.isEncrypted ? this.encryptionPassword : null,
        });
        this.successMessage = '备份任务创建成功！';
        this.errorMessage = '';
      } catch (error) {
        this.errorMessage = '备份任务创建失败，请重试。';
        this.successMessage = '';
      }
    },
    changeCron(val) {
      if (typeof val === 'string') {
        this.cronSchedule = val;
      }
    },
    togglePopover(bol) {
      this.cronPopover = bol;
    },
    addRemote() {
      this.rcloneRemote.push('');
    },
    removeRemote(index) {
      if (this.rcloneRemote.length > 1) {
        this.rcloneRemote.splice(index, 1);
      }
    },
    resetForm() {
      this.taskName = '';
      this.sourceDir = '';
      this.rcloneRemote = [''];
      this.isSplit = false;
      this.isEncrypted = false;
      this.maxBackups = '';
      this.encryptionPassword = '';
      this.cronSchedule = '';
      this.successMessage = '';
      this.errorMessage = '';
    },
  },
};
</script>

