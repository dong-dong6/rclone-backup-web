package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"rclone/models"
	"rclone/utils"
	"text/template"
)

// 登录请求结构体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Bash脚本所需的变量结构体
type bashValue struct {
	TaskName           string   `json:"taskName"`
	SourceDir          string   `json:"sourceDir"`
	RcloneRemote       []string `json:"rcloneRemote"`
	MaxBackups         int      `json:"maxBackups"`
	IsSplit            bool     `json:"isSplit"`
	IsEncrypt          bool     `json:"isEncrypted"`
	EncryptionPassword string   `json:"encryptionPassword"`
	RcloneRemoteStr    string
}

// 登录响应结构体
type LoginResponse struct {
	Token string `json:"token"`
}

// Login 处理用户登录
func Login(w http.ResponseWriter, r *http.Request) {
	var loginReq LoginRequest
	// 解析请求体中的JSON数据到LoginRequest结构体中
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}
	// 验证用户名和密码
	if models.AuthenticateUser(loginReq.Username, loginReq.Password) {
		// 生成JWT令牌
		token, err := utils.GenerateJWT(loginReq.Username)
		if err != nil {
			http.Error(w, "生成令牌时出错", http.StatusInternalServerError)
			return
		}
		// 将令牌编码成JSON格式并返回
		json.NewEncoder(w).Encode(LoginResponse{Token: token})
	} else {
		http.Error(w, "用户名或密码无效", http.StatusUnauthorized)
	}
}

// CreateBackupTask 创建备份任务 (需验证token)
func CreateBackupTask(w http.ResponseWriter, r *http.Request) {
	// 从请求头中获取Token
	authHeader := r.Header.Get("Authorization")
	// 检查Token是否为空或格式是否错误
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		http.Error(w, "缺少或格式错误的令牌", http.StatusUnauthorized)
		return
	}

	// 解析Token字符串
	tokenStr := authHeader[7:]
	claims, err := utils.ParseJWT(tokenStr)
	if err != nil {
		http.Error(w, "无效的令牌", http.StatusUnauthorized)
		return
	}
	print(claims) // 打印Token的声明内容（用于调试）

	var bashValue bashValue
	// 解析请求体中的JSON数据到bashValue结构体中
	if err := json.NewDecoder(r.Body).Decode(&bashValue); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}
	//修改远程仓库数组
	var RcloneRemoteStr string
	//("remote1:path1" "remote2:path2" "remote3:path3")
	RcloneRemoteStr = "("
	for i := 0; i < len(bashValue.RcloneRemote); i++ {
		RcloneRemoteStr += "\"" + bashValue.RcloneRemote[i] + "\" "
	}
	RcloneRemoteStr += ")"
	//bashValue.RcloneRemoteStr = "(\\\"" + strings.Join(bashValue.RcloneRemote, "\\\" \\\"") + "\\\")"
	bashValue.RcloneRemoteStr = RcloneRemoteStr
	// 从文件中读取Bash脚本模板内容
	templateContent, err := ioutil.ReadFile("bashTemplate/backup.sh")
	if err != nil {
		fmt.Println("读取模板文件时出错:", err)
		return
	}

	// 使用Go的模板系统解析模板内容
	tmpl, err := template.New("script").Parse(string(templateContent))
	if err != nil {
		fmt.Println("创建模板时出错:", err)
		return
	}
	bashValue.TaskName = "Bash/" + bashValue.TaskName
	// 创建一个文件用于保存生成的脚本
	outputFile, err := os.Create(bashValue.TaskName)
	if err != nil {
		fmt.Println("创建文件时出错:", err)
		return
	}
	defer outputFile.Close()

	// 将模板应用于bashValue变量，并将结果写入文件
	err = tmpl.Execute(outputFile, bashValue)
	if err != nil {
		fmt.Println("执行模板时出错:", err)
		return
	}

	// 设置脚本文件的执行权限
	err = os.Chmod(bashValue.TaskName, 0755)
	if err != nil {
		fmt.Println("设置文件权限时出错:", err)
		return
	}

	fmt.Println("Bash脚本生成成功: " + bashValue.TaskName)
	// 成功返回响应，通知客户端备份任务已创建
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "备份任务创建成功"})
}
