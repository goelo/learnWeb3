package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// serializeBorshString 手动序列化 Borsh 格式的字符串
// Borsh 字符串格式: [4字节长度(小端)] + [UTF-8字节]
func serializeBorshString(s string) []byte {
	strBytes := []byte(s)
	length := uint32(len(strBytes))
	
	// 创建缓冲区: 4字节长度 + 字符串内容
	buf := make([]byte, 4+len(strBytes))
	binary.LittleEndian.PutUint32(buf[0:4], length)
	copy(buf[4:], strBytes)
	
	return buf
}

func main() {
	fmt.Println("正在启动 Go 客户端...")

	// 1. 连接到本地 Solana 测试验证器
	client := rpc.New("http://127.0.0.1:8899")

	// 2. 加载钱包密钥
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("无法获取用户目录: %w", err))
	}

	walletPath := filepath.Join(homeDir, ".config", "solana", "id.json")
	walletData, err := os.ReadFile(walletPath)
	if err != nil {
		panic(fmt.Errorf("无法读取钱包文件: %w", err))
	}

	var secretKey []byte
	if err := json.Unmarshal(walletData, &secretKey); err != nil {
		panic(fmt.Errorf("无法解析钱包数据: %w", err))
	}

	payer := solana.PrivateKey(secretKey)
	fmt.Printf("钱包地址: %s\n", payer.PublicKey())

	// 3. 程序 ID（确保与你部署的程序 ID 一致）
	programID := solana.MustPublicKeyFromBase58("CuRF5bMpCoatpfGTKy7H99JoAseKEUCrENzFv9yHTnG4")

	// 4. 准备要发送的数据 - 使用 Borsh 序列化
	message := "Hello from Golang! 🚀"
	instructionData := serializeBorshString(message)

	// 5. 创建交易指令
	instruction := solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			{
				PublicKey:  payer.PublicKey(),
				IsSigner:   true,
				IsWritable: true,
			},
		},
		instructionData,
	)

	// 6. 获取最新的区块哈希
	recent, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		panic(fmt.Errorf("获取区块哈希失败: %w", err))
	}

	// 7. 创建交易
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		panic(fmt.Errorf("创建交易失败: %w", err))
	}

	// 8. 签名交易
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payer.PublicKey()) {
			return &payer
		}
		return nil
	})
	if err != nil {
		panic(fmt.Errorf("签名失败: %w", err))
	}

	// 9. 发送交易
	fmt.Println("正在发送交易...")
	sig, err := client.SendTransactionWithOpts(
		context.Background(),
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: rpc.CommitmentConfirmed,
		},
	)
	if err != nil {
		panic(fmt.Errorf("发送交易失败: %w", err))
	}

	fmt.Printf("✅ 交易成功！Signature: %s\n", sig)
	fmt.Printf("查看日志: https://explorer.solana.com/tx/%s?cluster=custom&customUrl=http://127.0.0.1:8899\n", sig)
	fmt.Printf("或使用命令行: solana confirm -v %s --url http://127.0.0.1:8899\n", sig)
}
