package handlers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorhill/cronexpr"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"rclone/models"
	"rclone/utils"
	"strings"
	"text/template"
)

// 登录请求结构体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 注册请求结构体
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Bash 脚本所需的变量结构体
type BashValue struct {
	TaskName           string   `json:"taskName"`
	SourceDir          string   `json:"sourceDir"`
	RcloneRemote       []string `json:"rcloneRemote"`
	MaxBackups         int      `json:"maxBackups"`
	IsSplit            bool     `json:"isSplit"`
	IsEncrypt          bool     `json:"isEncrypted"`
	EncryptionPassword string   `json:"encryptionPassword"`
	CronTime           string   `json:"cronSchedule"`
	RcloneRemoteStr    string
}

// 登录响应结构体
type LoginResponse struct {
	Token string `json:"token"`
}

// RegisterUserHandler 处理用户注册（只能运行一次）
func RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var registerReq RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}

	err := models.RegisterUser(registerReq.Username, registerReq.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "注册成功"})
}

// Login 处理用户登录
func Login(w http.ResponseWriter, r *http.Request) {
	var loginReq LoginRequest
	// 解析请求体中的 JSON 数据到 LoginRequest 结构体中
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}
	// 验证用户名和密码
	if models.AuthenticateUser(loginReq.Username, loginReq.Password) {
		// 生成 JWT 令牌
		token, err := utils.GenerateJWT(loginReq.Username)
		if err != nil {
			http.Error(w, "生成令牌时出错", http.StatusInternalServerError)
			return
		}
		// 将令牌编码成 JSON 格式并返回
		json.NewEncoder(w).Encode(LoginResponse{Token: token})
	} else {
		http.Error(w, "用户名或密码无效", http.StatusUnauthorized)
	}
}

// CreateBackupTask 创建备份任务（需验证 token）
func CreateBackupTask(w http.ResponseWriter, r *http.Request) {
	// 从请求头中获取 Token
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 7 || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "缺少或格式错误的令牌", http.StatusUnauthorized)
		return
	}

	// 解析 Token 字符串
	tokenStr := authHeader[7:]
	claims, err := utils.ParseJWT(tokenStr)
	if err != nil {
		http.Error(w, "无效的令牌", http.StatusUnauthorized)
		return
	}
	fmt.Println("Token Claims:", claims) // 打印 Token 的声明内容（用于调试）

	var bashVal BashValue
	// 解析请求体中的 JSON 数据到 bashVal 结构体中
	if err := json.NewDecoder(r.Body).Decode(&bashVal); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}

	// 验证 Cron 表达式
	if err := validateCronExpr(bashVal.CronTime); err != nil {
		http.Error(w, "无效的 Cron 表达式", http.StatusBadRequest)
		return
	}

	// 修改远程仓库数组，构建 RcloneRemoteStr
	var RcloneRemoteStr strings.Builder
	RcloneRemoteStr.WriteString("(")
	for _, remote := range bashVal.RcloneRemote {
		RcloneRemoteStr.WriteString("\"" + remote + "\" ")
	}
	RcloneRemoteStr.WriteString(")")
	bashVal.RcloneRemoteStr = RcloneRemoteStr.String()

	// 从文件中读取 Bash 脚本模板内容
	templateContent, err := os.ReadFile("bashTemplate/backup.sh")
	if err != nil {
		fmt.Println("读取模板文件时出错:", err)
		http.Error(w, "读取模板文件时出错", http.StatusInternalServerError)
		return
	}

	// 使用 Go 的模板系统解析模板内容
	tmpl, err := template.New("script").Parse(string(templateContent))
	if err != nil {
		fmt.Println("创建模板时出错:", err)
		http.Error(w, "创建模板时出错", http.StatusInternalServerError)
		return
	}

	// 设置脚本文件名
	bashVal.TaskName = bashVal.TaskName + ".sh"

	exeDir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取工作目录失败:", err)
		http.Error(w, "获取工作目录失败", http.StatusInternalServerError)
		return
	}

	// 构建脚本的完整路径
	scriptDir := filepath.Join(exeDir, "Bash")
	scriptPath := filepath.Join(scriptDir, bashVal.TaskName)

	// 创建脚本文件的目录（如果不存在）
	err = os.MkdirAll(scriptDir, os.ModePerm)
	if err != nil {
		fmt.Println("创建脚本目录时出错:", err)
		http.Error(w, "创建脚本目录时出错", http.StatusInternalServerError)
		return
	}

	// 创建一个文件用于保存生成的脚本
	outputFile, err := os.Create(scriptPath)
	if err != nil {
		fmt.Println("创建文件时出错:", err)
		http.Error(w, "创建脚本文件时出错", http.StatusInternalServerError)
		return
	}
	defer outputFile.Close()

	// 将模板应用于 bashVal 变量，并将结果写入文件
	err = tmpl.Execute(outputFile, bashVal)
	if err != nil {
		fmt.Println("执行模板时出错:", err)
		http.Error(w, "执行模板时出错", http.StatusInternalServerError)
		return
	}

	// 设置脚本文件的执行权限
	err = os.Chmod(scriptPath, 0755)
	if err != nil {
		fmt.Println("设置文件权限时出错:", err)
		http.Error(w, "设置脚本权限时出错", http.StatusInternalServerError)
		return
	}

	fmt.Println("Bash 脚本生成成功: " + scriptPath)

	// 添加定时任务到系统 crontab
	err = addCronJob(bashVal.CronTime, scriptPath, bashVal.TaskName)
	if err != nil {
		fmt.Println("添加定时任务失败:", err)
		http.Error(w, "添加定时任务失败", http.StatusInternalServerError)
		return
	}
	fmt.Println("定时任务已添加到系统 crontab")

	// 成功返回响应，通知客户端备份任务已创建
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "备份任务创建成功"})
}

// validateCronExpr 验证 Cron 表达式是否有效
func validateCronExpr(expr string) error {
	// 确保 expr 有 5 个字段
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return errors.New("Cron 表达式必须有 5 个字段")
	}
	// 在最前面添加 0 作为秒字段
	exprWithSeconds := "0 " + expr
	_, err := cronexpr.Parse(exprWithSeconds)
	if err != nil {
		return errors.New("无效的 Cron 表达式")
	}
	return nil
}

// addCronJob 将任务添加到系统 crontab
func addCronJob(cronExpr string, scriptPath string, taskName string) error {
	// 验证 Cron 表达式
	if err := validateCronExpr(cronExpr); err != nil {
		return fmt.Errorf("无效的 Cron 表达式: %s", cronExpr)
	}

	// 获取当前用户的 crontab
	cmd := exec.Command("crontab", "-l")
	currentCrontabBytes, err := cmd.Output()
	var currentCrontab string
	if err != nil {
		// 如果 crontab 不存在，初始化为空字符串
		currentCrontab = ""
	} else {
		currentCrontab = string(currentCrontabBytes)
	}

	// 构建新的 cron 任务，添加任务标识注释
	cronJob := fmt.Sprintf("# Backup Task: %s\n%s %q\n", taskName, cronExpr, scriptPath)

	// 检查是否已经存在相同的任务，避免重复添加
	if strings.Contains(currentCrontab, cronJob) {
		fmt.Println("任务已存在于 crontab 中")
		return nil
	}

	// 将新的任务添加到现有的 crontab
	newCrontab := currentCrontab + "\n" + cronJob

	// 创建一个临时文件，写入新的 crontab 内容
	tmpFile, err := os.CreateTemp("", "crontab_*.txt")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(newCrontab)
	if err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	// 使用 crontab 命令将新的 crontab 安装
	cmd = exec.Command("crontab", tmpFile.Name())
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("安装新的 crontab 失败: %v", err)
	}

	return nil
}

// FileNode 结构体，用于表示文件系统节点
type FileNode struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsLeaf      bool   `json:"isLeaf"`
	IsDirectory bool   `json:"isDirectory"`
}

// 验证路径是否合法，防止目录遍历攻击
func isValidPath(p string) bool {
	baseDir := "/"
	resolvedPath, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	baseDirAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}
	return strings.HasPrefix(resolvedPath, baseDirAbs)
}

// 获取rclone配置文件路径
func getRcloneConfigFilePath() (string, error) {
	// 运行 'rclone config file' 命令获取配置文件路径
	cmd := exec.Command("rclone", "config", "file")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// 将输出转为字符串并打印
	outputStr := string(output)
	//fmt.Println(outputStr)

	// 使用 strings.TrimSpace 去掉首尾空白字符，并提取路径部分
	// 假设路径始终位于 "Configuration file is stored at:" 后面
	// 查找配置文件路径并返回
	const prefix = "Configuration file is stored at:"
	if strings.HasPrefix(outputStr, prefix) {
		// 去掉 "Configuration file is stored at:" 部分，只保留路径
		configFilePath := strings.TrimSpace(outputStr[len(prefix):])
		print(configFilePath)
		return configFilePath, nil
	}

	return "", fmt.Errorf("无法解析配置文件路径: %s", outputStr)
}

// 解析 rclone 配置文件，返回所有配置的名称
func getRcloneConfigNames(configFilePath string) ([]string, error) {
	var configNames []string
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentConfig string
	for scanner.Scan() {
		line := scanner.Text()

		// 检查每一行是否为配置名称（以 [ 配对的部分开始）
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// 获取配置名称，去除前后的方括号
			currentConfig = strings.Trim(line, "[]")
			configNames = append(configNames, currentConfig)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return configNames, nil
}

func RcloneConfig(w http.ResponseWriter, r *http.Request) {
	// 从请求头中获取 Token
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 7 || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "缺少或格式错误的令牌", http.StatusUnauthorized)
		return
	}

	// 获取 rclone 配置文件路径
	configFilePath, err := getRcloneConfigFilePath()
	//print(configFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取 rclone 配置文件路径失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 获取配置名称列表
	configNames, err := getRcloneConfigNames(configFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("读取配置文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回配置名称列表
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(configNames)

}

// FilesystemHandler 处理文件系统请求
func FilesystemHandler(w http.ResponseWriter, r *http.Request) {
	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		requestedPath = "/"
	}

	if !isValidPath(requestedPath) {
		http.Error(w, "无效的路径", http.StatusBadRequest)
		return
	}

	entries, err := os.ReadDir(requestedPath)
	if err != nil {
		http.Error(w, "读取目录失败", http.StatusInternalServerError)
		return
	}

	var data []FileNode
	for _, entry := range entries {
		isDir := entry.IsDir()
		node := FileNode{
			Name:        entry.Name(),
			Path:        filepath.Join(requestedPath, entry.Name()),
			IsLeaf:      !isDir,
			IsDirectory: isDir,
		}
		data = append(data, node)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
