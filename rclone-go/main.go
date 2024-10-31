package main

import (
	"log"
	"net/http"
	"rclone/handlers"
)

// 设置CORS中间件
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
	http.HandleFunc("/api/login", handlers.Login)
	http.HandleFunc("/api/create_backup_task", handlers.CreateBackupTask)
	// 将CORS中间件应用于所有路由
	log.Println("Server started on :628")
	log.Fatal(http.ListenAndServe(":628", enableCORS(http.DefaultServeMux)))
}
