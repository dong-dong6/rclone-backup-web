<template>
  <div class="container">
    <el-card class="box-card">
      <div slot="header" class="header">
        <span>创建备份任务</span>
      </div>
      <el-form @submit.prevent="createBackupTask" label-width="150px">
        <!-- 任务名称 -->
        <el-form-item label="任务名称" class="centered-label">
          <el-input v-model="taskName" placeholder="请输入任务名称"/>
        </el-form-item>

        <!-- 选择文件夹 -->
        <el-form-item label="选择文件夹" class="centered-label">
          <el-popover
              placement="bottom-start"
              width="400"
              trigger="click"
              v-model:visible="treePopoverVisible"
          >
            <el-tree
                :data="treeData"
                :props="defaultProps"
                lazy
                :load="loadNode"
                node-key="path"
                highlight-current
                @node-click="handleNodeClick"
                style="max-height: 300px; overflow-y: auto;"
            ></el-tree>
            <template #reference>
              <el-input
                  v-model="sourceDir"
                  placeholder="点击选择文件夹"
                  readonly
                  @focus="treePopoverVisible = true"
              >
                <template #suffix>
                  <i
                      class="el-icon-folder-opened"
                      style="cursor: pointer;"
                      @click="treePopoverVisible = true"
                  ></i>
                </template>
              </el-input>
            </template>
          </el-popover>
        </el-form-item>
        <div
            v-for="(remote, index) in rcloneRemote"
            :key="index"
            class="rclone-item"
        >
          <!-- 配置名称选择框 -->
          <el-form-item :label="`Rclone配置名称 ${index + 1}`" class="inline-label">
            <el-select
                v-model="remote.remoteName"
                placeholder="请选择 Rclone 配置名称"
                @change="configSelect"
            >
              <el-option
                  v-for="item in configOptions"
                  :key="item"
                  :label="item"
                  :value="item"
              >
              </el-option>
            </el-select>
          </el-form-item>

          <!-- 路径输入框 -->
          <el-form-item :label="`Rclone路径 ${index + 1}`" class="inline-label">
            <el-input
                v-model="remote.remotePath"
                placeholder="请输入路径"
            />
          </el-form-item>

          <!-- 删除按钮 -->
          <el-button
              v-if="rcloneRemote.length > 1"
              type="text"
              class="delete-button"
              @click="removeRemote(index)"
              icon="el-icon-delete"
          >
            删除
          </el-button>
        </div>


        <el-form-item class="centered-label">
          <el-button type="primary" icon="el-icon-plus" @click="addRemote">
            添加新的 Rclone 配置名称
          </el-button>
        </el-form-item>

        <!-- 最大留存备份数量 -->
        <el-form-item label="最大留存备份数量" class="centered-label">
          <el-input-number v-model="maxBackups" :min="1"/>
        </el-form-item>

        <!-- 是否分卷压缩 -->
        <el-form-item label="是否分卷压缩(暂不支持)" class="centered-label">
          <el-checkbox v-model="isSplit"></el-checkbox>
        </el-form-item>

        <!-- 是否加密压缩 -->
        <el-form-item label="是否加密压缩" class="centered-label">
          <el-checkbox v-model="isEncrypted"></el-checkbox>
        </el-form-item>

        <!-- 加密密码 -->
        <el-form-item label="加密密码" v-if="isEncrypted" class="centered-label">
          <el-input
              v-model="encryptionPassword"
              placeholder="输入加密密码"
              show-password
          />
        </el-form-item>

        <!-- 定时规则 -->
        <el-form-item label="定时规则" class="centered-label">
          <el-select
              v-model="selectedCronOption"
              placeholder="选择定时规则"
              @change="onCronOptionChange"
              style="width: 100%;"
          >
            <el-option
                v-for="option in cronOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
            ></el-option>
          </el-select>
          <el-input
              v-model="cronSchedule"
              :readonly="selectedCronOption !== 'custom'"
              placeholder="Cron表达式"
              style="margin-top: 10px;"
          ></el-input>
        </el-form-item>

        <!-- 按钮组 -->
        <el-form-item class="button-group">
          <el-button type="primary" native-type="submit" icon="el-icon-check">
            提交
          </el-button>
          <el-button
              @click="resetForm"
              type="default"
              icon="el-icon-refresh-right"
          >
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import api from '../api'; // 请根据您的项目结构调整路径

export default {
  data() {
    return {
      sourceDir: '',
      rcloneRemote: [
        { remoteName: '', remotePath: '' }
      ],
      taskName: '',
      isSplit: false,
      isEncrypted: false,
      maxBackups: '',
      encryptionPassword: '',
      successMessage: '',
      cronSchedule: '',
      errorMessage: '',
      treeData: [],
      defaultProps: {
        children: 'children',
        label: 'name',
        isLeaf: 'isLeaf',
      },
      configOptions: [],
      treePopoverVisible: false, // 控制文件树弹出框的显示
      selectedCronOption: '',
      cronOptions: [
        {value: 'every6hours', label: '每隔6小时'},
        {value: 'daily3am', label: '每天凌晨3点'},
        {value: 'everyMonday', label: '每周一'},
        {value: 'custom', label: '自定义'},
      ],
    };
  },
  methods: {
    async createBackupTask() {
      try {
        const formattedRcloneRemote = this.rcloneRemote.map(
            (remote) => `${remote.remoteName}:${remote.remotePath}`
        );
        await api.post('/create_backup_task', {
          sourceDir: this.sourceDir,
          rcloneRemote: formattedRcloneRemote,
          isSplit: this.isSplit,
          taskName: this.taskName,
          maxBackups: this.maxBackups,
          isEncrypted: this.isEncrypted,
          cronSchedule: this.cronSchedule,
          encryptionPassword: this.isEncrypted
              ? this.encryptionPassword
              : null,
        });
        this.successMessage = '备份任务创建成功！';
        this.errorMessage = '';
      } catch (error) {
        this.errorMessage = '备份任务创建失败，请重试。';
        this.successMessage = '';
      }
    },
    async fetchConfigOptions() {
      try {
        const response = await api.get('rclone_config'); // 后端接口地址
        if (response.data) {
          this.configOptions = response.data; // 设置下拉框的选项
        }
      } catch (error) {
        console.error('获取配置项失败:', error);
      }
    },
    onCronOptionChange(value) {
      switch (value) {
        case 'every6hours':
          this.cronSchedule = '0 */6 * * *';
          break;
        case 'daily3am':
          this.cronSchedule = '0 3 * * *';
          break;
        case 'everyMonday':
          this.cronSchedule = '0 0 * * 1';
          break;
        case 'custom':
          this.cronSchedule = '';
          break;
        default:
          this.cronSchedule = '';
      }
    },
    addRemote() {
      this.rcloneRemote.push({ remoteName: '', remotePath: '' });
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
      this.selectedCronOption = '';
      this.successMessage = '';
      this.errorMessage = '';
    },
    loadNode(node, resolve) {
      const path = node.level === 0 ? '/' : node.data.path;
      api
          .get('/filesystem', {params: {path}})
          .then((response) => {
            resolve(response.data);
          })
          .catch((error) => {
            console.error(error);
            resolve([]);
          });
    },
    handleNodeClick(nodeData) {
      if (nodeData.isDirectory) {
        this.sourceDir = nodeData.path;
        this.treePopoverVisible = false; // 关闭弹出框
      }
    },
  },
  mounted() {
    this.fetchConfigOptions(); // 在组件挂载后获取配置选项
  }
};
</script>

<style scoped>
.container {
  max-width: 600px;
  margin: auto;
  padding: 20px;
}
.rclone-item {
  display: flex;
  align-items: center;
  gap: 10px; /* 控制各部分的间距 */
  margin-bottom: 20px; /* 设置行与行之间的间距 */
}

.rclone-item .inline-label {
  flex: 1; /* 配置名称和路径输入框等宽 */
  margin: 0; /* 去除多余的外边距 */
}

.delete-button {
  color: #ff4d4f;
  cursor: pointer;
  flex-shrink: 0; /* 删除按钮不缩小 */
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
