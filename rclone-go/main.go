package main

import (
	"log"
	"net/http"
	"rclone/handlers"
)

// 设置 CORS 中间件
func enableCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// 如果是预检请求，直接返回
		if r.Method == "OPTIONS" {
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func main() {
	// 注册路由
	http.HandleFunc("/register", handlers.RegisterUserHandler)        // 注册接口，只能运行一次
	http.HandleFunc("/login", handlers.Login)                         // 登录接口
	http.HandleFunc("/create_backup_task", handlers.CreateBackupTask) // 创建备份任务接口
	http.HandleFunc("/filesystem", handlers.FilesystemHandler)        // 文件系统浏览接口
	http.HandleFunc("/rclone_config", handlers.RcloneConfig)          // 获取 rclone 配置文件路径接口
	// 将 CORS 中间件应用于所有路由
	log.Println("Server started on :628")
	log.Fatal(http.ListenAndServe(":628", enableCORS(http.DefaultServeMux)))
}
