# Napcat 账号风控监控

监控 Napcat 账号在线状态，账号离线时自动发送邮件提醒。

## 功能

- ✅ 账号离线自动发送告警邮件
- ✅ 账号恢复上线发送恢复通知
- ✅ 跨平台支持（Windows / Linux / macOS）
- ✅ 配置文件自动生成
- ✅ 体积小巧（~5MB）

## 下载

前往 [Releases](https://github.com/UCKETX/napcat-monitor/releases) 下载对应平台的可执行文件：

| 平台 | 文件 |
|------|------|
| Windows | `napcat_monitor_windows.exe` |
| Linux | `napcat_monitor_linux` |
| macOS | `napcat_monitor_macos` |

## 配置

### 方式一：自动生成配置

1. 运行程序，会自动在当前目录创建 `.env` 文件
2. 编辑 `.env` 填入配置
3. 重新运行程序

### 方式二：手动创建配置

创建 `.env` 文件：

```bash
# Napcat HTTP 服务器配置
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
```

### 配置说明

| 配置项 | 说明 | 示例 |
|--------|------|------|
| `NAPCAT_URL` | Napcat HTTP 服务器地址 | `http://192.168.1.100:23236` |
| `ACCOUNT_API` | 账号状态检测 API | `/get_status` |
| `API_TOKEN` | API 认证 Token（可选） | - |
| `SMTP_SERVER` | SMTP 服务器 | `smtp.qq.com` |
| `SMTP_PORT` | SMTP 端口 | `587` |
| `SMTP_USER` | 发送邮箱 | `your_email@qq.com` |
| `SMTP_PASS` | SMTP 授权码 | - |
| `USE_TLS` | 是否使用 TLS | `true` |
| `TO_EMAIL` | 收件人邮箱 | `receive@example.com` |
| `CHECK_INTERVAL` | 检测间隔（秒） | `60` |
| `FAIL_THRESHOLD` | 失败次数阈值 | `3` |

## 运行

```bash
# Windows
napcat_monitor_windows.exe

# Linux
chmod +x napcat_monitor_linux
./napcat_monitor_linux

# macOS
chmod +x napcat_monitor_macos
./napcat_monitor_macos
```

### 后台运行

```bash
# Linux/macOS
nohup ./napcat_monitor_linux &
```

### 开机自启

**Linux (systemd)**
```ini
# /etc/systemd/system/napcat-monitor.service
[Unit]
Description=Napcat Monitor

[Service]
Type=simple
User=你的用户
WorkingDirectory=/path/to/napcat
ExecStart=/path/to/napcat/napcat_monitor_linux
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable napcat-monitor
sudo systemctl start napcat-monitor
```

## 编译

如需自行编译：

```bash
# 安装 Go
# https://go.dev/dl/

# 克隆项目
git clone https://github.com/UCKETX/napcat-monitor.git
cd napcat-monitor

# 编译
go build -ldflags="-s -w" -o napcat_monitor_linux .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o napcat_monitor_windows.exe .
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o napcat_monitor_macos .
```

## 行为说明

- 连续检测失败 3 次后发送离线告警
- 整个离线周期只发送一次告警
- 账号恢复上线后发送恢复通知，并重置状态

## GitHub

https://github.com/UCKETX/napcat-monitor
