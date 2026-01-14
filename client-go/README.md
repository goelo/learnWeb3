# Solana Go 客户端

这是一个使用 Golang 编写的 Solana 客户端，用于与本地 Solana 测试验证器交互。

## 功能

- 连接到本地 Solana 测试验证器 (http://127.0.0.1:8899)
- 加载本地钱包密钥
- 使用 Borsh 序列化数据
- 发送交易到 Solana 程序
- 显示交易签名和日志链接

## 依赖

- [gagliardetto/solana-go](https://github.com/gagliardetto/solana-go) - Solana Go SDK
- [near/borsh-go](https://github.com/near/borsh-go) - Borsh 序列化库

## 安装依赖

```bash
go mod download
```

## 运行

```bash
go run main.go
```

## 注意事项

1. 确保本地 Solana 测试验证器正在运行
2. 确保钱包文件存在于 `~/.config/solana/id.json`
3. 确保 `main.go` 中的程序 ID 与你部署的程序 ID 一致
