package models

import (
	"database/sql"
	"log"
	_ "modernc.org/sqlite"
)

// ConnectDB 连接到SQLite数据库
func ConnectDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "./rclone.db")
	if err != nil {
		return nil, err
	}
	return db, nil
}

// AuthenticateUser 验证用户名和密码
func AuthenticateUser(username, password string) bool {
	db, err := ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var storedPassword string
	err = db.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&storedPassword)
	if err != nil {
		return false
	}
	return password == storedPassword // 实际应用中应使用密码哈希对比
}
