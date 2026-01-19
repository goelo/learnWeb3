# PassengerSys - 出行订单系统客户端

基于 Solana 区块链的出行订单系统 Go 客户端。

## 系统架构

```
┌─────────────────┐         ┌─────────────────────────┐
│  passengersys   │   RPC   │   Solana Local Testnet  │
│   (Go Client)   │ ──────► │                         │
│                 │         │   order_program (Rust)  │
└─────────────────┘         └─────────────────────────┘
```

## 环境要求

- Go 1.21+
- Solana CLI 1.17+
- 本地 Solana 验证器

## 代理配置

> ⚠️ **重要**: 如果你的环境配置了代理，需要设置 `NO_PROXY` 环境变量来绕过本地连接。

```bash
export NO_PROXY=localhost,127.0.0.1
```

或者在每条命令前添加：

```bash
NO_PROXY=localhost,127.0.0.1 <command>
```

## 快速开始

### 1. 启动本地 Solana 验证器

```bash
# 启动验证器 (在后台运行)
solana-test-validator --reset &

# 或者指定 ledger 目录 (推荐)
solana-test-validator --ledger ./test-ledger --reset &

# 验证是否启动成功 (注意代理配置)
NO_PROXY=localhost,127.0.0.1 solana cluster-version -u http://127.0.0.1:8899
```

### 2. 配置 Solana CLI

```bash
# 设置使用本地验证器
solana config set --url http://127.0.0.1:8899

# 查看当前配置
solana config get

# 查看钱包余额 (本地测试网有大量测试 SOL)
NO_PROXY=localhost,127.0.0.1 solana balance -u http://127.0.0.1:8899
```

### 3. 部署 Solana 程序

```bash
cd ../order_program

# 编译程序
cargo build-sbf

# 部署程序
NO_PROXY=localhost,127.0.0.1 solana program deploy \
  target/deploy/order_program.so \
  --program-id target/deploy/order_program-keypair.json \
  -u http://127.0.0.1:8899
```

### 4. 运行客户端

```bash
cd ../passengersys

# 安装依赖
go mod tidy

# 运行客户端创建订单
NO_PROXY=localhost,127.0.0.1 go run main.go
```

### 5. 验证订单

```bash
# 查看订单 PDA 账户数据
NO_PROXY=localhost,127.0.0.1 solana account <PDA地址> -u http://127.0.0.1:8899
```

## 项目结构

```
passengersys/
├── main.go      # 客户端主程序
├── go.mod       # Go 模块定义
├── go.sum       # 依赖锁定
└── README.md    # 本文档

../order_program/
├── src/
│   └── lib.rs   # Solana 程序源码
├── Cargo.toml   # Rust 依赖配置
└── target/
    └── deploy/  # 编译输出
```

## 关键配置

| 配置项 | 值 |
|--------|-----|
| Program ID | `7GYA35cY5s7wCPiEm5ni4EAGBaXZn73fNehU4dsS1VyP` |
| RPC URL | `http://127.0.0.1:8899` |
| WebSocket | `ws://127.0.0.1:8900` |

## 常见问题

### Q: 遇到 503 Service Unavailable 错误？

检查代理配置，使用 `NO_PROXY=localhost,127.0.0.1` 或使用 `127.0.0.1` 而不是 `localhost`。

### Q: 验证器无法启动？

检查端口是否被占用：
```bash
lsof -i :8899
```

### Q: 如何停止验证器？

```bash
pkill -f solana-test-validator
```
