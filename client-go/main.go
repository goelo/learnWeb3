package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
)

type HelloInstruction struct {
	Message string
}

func (i *HelloInstruction) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	msgBytes := []byte(i.Message)
	// 写入长度，小端
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(msgBytes))); err != nil {
		return nil, err
	}
	// 写入内容
	if _, err := buf.Write(msgBytes); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func main() {
	fmt.Println("正在启动 Go 客户端...")

	// 1. 连接到本地 Solana 测试验证器
	client := rpc.New("http://127.0.0.1:8899")
	ctx := context.Background()

	// 2. 加载钱包密钥
	homeDir, _ := os.UserHomeDir()
	wallerPath := homeDir + "/.config/solana/id.json"

	payer, err := solana.PrivateKeyFromSolanaKeygenFile(wallerPath)
	if err != nil {
		panic(fmt.Errorf("无法加载钱包密钥: %w", err))
	}

	fmt.Printf("钱包地址: %s\n", payer.PublicKey())

	// 3. 程序 ID（确保与你部署的程序 ID 一致）
	programID := solana.MustPublicKeyFromBase58("CuRF5bMpCoatpfGTKy7H99JoAseKEUCrENzFv9yHTnG4")

	greetedAccount := solana.NewWallet()
	fmt.Printf("📝 新生成的记事本地址: %s\n", greetedAccount.PublicKey())

	message := "Hello from Golang! 🚀"
	instructionData := &HelloInstruction{
		Message: message,
	}

	serializeData, _ := instructionData.Serialize()
	// 计算所需空间 (4字节头部 + 字符串长度 + 额外一点冗余)
	// 4 (u32 len) + 21 (content) = 25. 给 50 字节足够了
	space := uint64(50)
	// 获取租金豁免所需的最小 lamports
	lamports, err := client.GetMinimumBalanceForRentExemption(ctx, space, rpc.CommitmentFinalized)
	if err != nil {
		panic(fmt.Errorf("获取租金失败: %w", err))
	}
	// 4. 构建指令 A: SystemProgram 创建账户
	// 这是一个原子操作的起点

	createAccoutIx := system.NewCreateAccountInstruction(
		lamports,
		space,
		programID,
		payer.PublicKey(),
		greetedAccount.PublicKey(),
	).Build()

	// 5. 构建指令 B: 调用我们的 Rust 程序写入数据
	helloIx := solana.NewInstruction(
		programID,
		[]*solana.AccountMeta{
			// 对应 Rust 里的 accounts (AccountInfo)
			// 注意：这个账户必须是可写的(Writeable)，但不需要是签名者(Signer)，
			// 因为上一条指令已经创建了它，且现在的 Owner 是程序自己。
			// 修正：在创建交易的同一个原子块内，如果通过 SystemProgram 创建，
			// 初始化时通常需要新账户的签名。
			{
				PublicKey:  greetedAccount.PublicKey(),
				IsWritable: true,
				IsSigner:   false,
			},
		},
		serializeData,
	)

	// 6. 构建并发送交易
	// 获取最新的区块哈希 (Recent Blockhash)
	recent, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		panic(fmt.Errorf("获取区块哈希失败: %w", err))
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			createAccoutIx,
			helloIx,
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		panic(fmt.Errorf("构建交易失败: %w", err))
	}

	// 7. 签名交易
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if payer.PublicKey().Equals(key) {
				return &payer
			}
			if greetedAccount.PublicKey().Equals(key) {
				return &greetedAccount.PrivateKey
			}
			return nil
		},
	)
	if err != nil {
		panic(fmt.Errorf("签名失败: %w", err))
	}
	// 8. 发送交易
	fmt.Println("正在发送交易...")
	sig, err := client.SendTransaction(ctx, tx)
	if err != nil {
		panic(fmt.Errorf("交易发送失败: %w", err))
	}
	fmt.Printf("✅ 交易成功！Signature: %s\n", sig)
	fmt.Printf("查看日志: https://explorer.solana.com/tx/%s?cluster=custom&customUrl=http://127.0.0.1:8899\n", sig)
	fmt.Printf("或使用命令行: solana confirm -v %s --url http://127.0.0.1:8899\n", sig)
}
