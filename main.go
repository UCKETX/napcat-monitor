package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Config 配置结构
type Config struct {
	NAPCAT_URL     string
	ACCOUNT_API    string
	API_TOKEN      string
	SMTP_SERVER    string
	SMTP_PORT      int
	SMTP_USER      string
	SMTP_PASS      string
	USE_TLS        bool
	TO_EMAIL       string
	CHECK_INTERVAL int
	FAIL_THRESHOLD int
}

// Response API 响应结构
type Response struct {
	Status  string `json:"status"`
	Retcode int    `json:"retcode"`
	Data    Data   `json:"data"`
	Message string `json:"message"`
	Wording string `json:"wording"`
}

type Data struct {
	Online bool `json:"online"`
	Good   bool `json:"good"`
}

var (
	config              Config
	consecutiveFailures int
	wasOffline          bool
	offlineAlertSent    bool // 标记是否已发送过掉线邮件，避免重复发送
)

func loadConfig() error {
	// 如果 .env 不存在，创建它
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		envContent := `# Napcat 监控配置
# 复制这份文件为 .env 并填入配置

# Napcat 配置
NAPCAT_URL=http://192.168.1.100:23236
ACCOUNT_API=/get_status
API_TOKEN=

# SMTP 配置
SMTP_SERVER=smtp.qq.com
SMTP_PORT=587
SMTP_USER=your_email@qq.com
SMTP_PASS=your_auth_code
USE_TLS=true
TO_EMAIL=receive@example.com

# 监控配置
CHECK_INTERVAL=60
FAIL_THRESHOLD=3
`
		if err := os.WriteFile(".env", []byte(envContent), 0644); err != nil {
			return fmt.Errorf("创建配置文件失败: %v", err)
		}
		fmt.Println("配置文件已创建: .env")
		fmt.Println("请编辑该文件填入配置后重新运行程序")
		os.Exit(0)
	}

	// 读取配置文件
	data, err := os.ReadFile(".env")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析配置
	lines := strings.Split(string(data), "\n")
	config = Config{
		ACCOUNT_API:    "/get_status",
		SMTP_PORT:      587,
		USE_TLS:        true,
		CHECK_INTERVAL: 60,
		FAIL_THRESHOLD: 3,
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "NAPCAT_URL":
			config.NAPCAT_URL = value
		case "ACCOUNT_API":
			config.ACCOUNT_API = value
		case "API_TOKEN":
			config.API_TOKEN = value
		case "SMTP_SERVER":
			config.SMTP_SERVER = value
		case "SMTP_PORT":
			config.SMTP_PORT, _ = strconv.Atoi(value)
		case "SMTP_USER":
			config.SMTP_USER = value
		case "SMTP_PASS":
			config.SMTP_PASS = value
		case "USE_TLS":
			config.USE_TLS = value == "true"
		case "TO_EMAIL":
			config.TO_EMAIL = value
		case "CHECK_INTERVAL":
			config.CHECK_INTERVAL, _ = strconv.Atoi(value)
		case "FAIL_THRESHOLD":
			config.FAIL_THRESHOLD, _ = strconv.Atoi(value)
		}
	}

	// 验证必填项
	if config.NAPCAT_URL == "" || config.SMTP_SERVER == "" || config.SMTP_USER == "" || config.SMTP_PASS == "" || config.TO_EMAIL == "" {
		return fmt.Errorf("缺少必填配置项")
	}

	return nil
}

func checkAccountStatus() (bool, string) {
	url := config.NAPCAT_URL + config.ACCOUNT_API

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return false, fmt.Sprintf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NapcatMonitor/1.0")
	if config.API_TOKEN != "" {
		req.Header.Set("Authorization", "Bearer "+config.API_TOKEN)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("连接失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Sprintf("读取响应失败: %v", err)
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Sprintf("解析响应失败: %v", err)
	}

	if result.Retcode == 0 && result.Status == "ok" {
		if result.Data.Online && result.Data.Good {
			return true, "账号在线"
		} else if !result.Data.Online {
			return false, "账号已离线"
		} else if !result.Data.Good {
			return false, "账号状态异常"
		}
	} else if result.Status == "failed" {
		msg := result.Message
		if msg == "" {
			msg = result.Wording
		}
		return false, fmt.Sprintf("请求失败: %s", msg)
	}

	return true, "API 可访问"
}

func sendEmail(subject, body string) error {
	from := config.SMTP_USER
	to := config.TO_EMAIL
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", from, to, subject, body)

	auth := smtp.PlainAuth("", config.SMTP_USER, config.SMTP_PASS, config.SMTP_SERVER)

	err := smtp.SendMail(
		config.SMTP_SERVER+":"+strconv.Itoa(config.SMTP_PORT),
		auth,
		from,
		[]string{to},
		[]byte(msg),
	)
	if err != nil {
		fmt.Printf("[%s] 邮件发送失败: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		return err
	}

	fmt.Printf("[%s] 邮件发送成功\n", time.Now().Format("2006-01-02 15:04:05"))
	return nil
}

func main() {
	// 加载配置
	if err := loadConfig(); err != nil {
		fmt.Printf("配置错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[%s] Napcat 账号监控启动\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("  API: %s%s\n", config.NAPCAT_URL, config.ACCOUNT_API)
	fmt.Printf("  检测间隔: %d秒\n", config.CHECK_INTERVAL)

	// 优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n收到退出信号，正在关闭...")
		os.Exit(0)
	}()

	for {
		isOnline, statusMsg := checkAccountStatus()
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] %s\n", currentTime, statusMsg)

		if isOnline {
			if wasOffline {
				fmt.Printf("[%s] 账号已恢复在线！\n", currentTime)
				go sendEmail("✅ Napcat 账号已恢复！", fmt.Sprintf("检测时间: %s\n账号已重新上线。", currentTime))
				wasOffline = false
				offlineAlertSent = false // 重置标志，允许下次掉线时再次发送告警
			}
			consecutiveFailures = 0
		} else {
			consecutiveFailures++
			wasOffline = true
			fmt.Printf("[%s] 账号离线 (%d/%d)\n", currentTime, consecutiveFailures, config.FAIL_THRESHOLD)

			// 只在达到阈值且尚未发送过掉线邮件时发送
			if consecutiveFailures >= config.FAIL_THRESHOLD && !offlineAlertSent {
				go sendEmail("⚠️ Napcat 账号已掉线！", fmt.Sprintf("检测时间: %s\n检测结果: %s", currentTime, statusMsg))
				offlineAlertSent = true // 标记已发送，避免重复
				fmt.Printf("[%s] 已发送掉线告警\n", currentTime)
			}
		}

		time.Sleep(time.Duration(config.CHECK_INTERVAL) * time.Second)
	}
}
