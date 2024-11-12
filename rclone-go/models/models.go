package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var (
	users []User
	once  sync.Once
)

// User 定义用户结构体
type User struct {
	Username string `json:"username"`
	Password string `json:"password"` // 存储密码哈希
}

// LoadConfig 加载配置文件（只加载一次）
func LoadConfig() {
	once.Do(func() {
		file, err := os.Open("users.json")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		err = decoder.Decode(&users)
		if err != nil {
			log.Fatal(err)
		}
	})
}

// ConnectDB 保留函数签名，但返回 nil
func ConnectDB() (*sql.DB, error) {
	return nil, nil
}

// AuthenticateUser 验证用户名和密码
func AuthenticateUser(username, password string) bool {
	LoadConfig()

	for _, user := range users {
		if user.Username == username {
			err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
			return err == nil
		}
	}
	return false
}

// RegisterUser 注册新用户并生成 users.json 文件
func RegisterUser(username, password string) error {
	// 检查 users.json 是否已存在，防止多次注册
	if _, err := os.Stat("users.json"); err == nil {
		return errors.New("注册已关闭：users.json 已存在")
	}

	// 生成密码哈希
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 创建用户列表
	newUser := User{
		Username: username,
		Password: string(passwordHash),
	}
	users = append(users, newUser)

	// 将用户列表写入 users.json 文件
	file, err := os.Create("users.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(users)
	if err != nil {
		return err
	}

	return nil
}
