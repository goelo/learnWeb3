package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// 程序 keypair 文件的相对路径（相对于 passengersys 目录）
const ProgramKeypairPath = "../order_program/target/deploy/order_program-keypair.json"

// OrderAccount 是链上存储的数据格式 (59 bytes)
type OrderAccount struct {
	Status    uint8
	Passenger [32]byte
	OrderID   uint8
	Price     uint64
	Memo      [16]byte
	Bump      uint8
}

// CreateOrderInstruction 是发往链上的"请求体" (Payload)
// 包含 1 字节指令 ID + OrderInstructionData (orderId + price + memo)
type CreateOrderInstruction struct {
	InstructionID uint8    // 指令 discriminator (1 = CreateOrder)
	OrderID       uint8    // 订单 ID
	Price         uint64   // 价格
	Memo          [16]byte // 备注
}

func (c CreateOrderInstruction) Serialize() ([]byte, error) {
	data := make([]byte, 26) // 1 + 1 + 8 + 16 = 26 bytes
	// 1. 写入 InstructionID (discriminator)
	data[0] = c.InstructionID
	// 2. 写入 OrderID
	data[1] = c.OrderID
	// 3. 写入 Price (使用 LittleEndian)
	binary.LittleEndian.PutUint64(data[2:10], c.Price)
	// 4. 写入 Memo
	copy(data[10:26], c.Memo[:])
	return data, nil
}

// LoadProgramIDFromKeypair 从 Solana keypair JSON 文件读取程序 ID (公钥)
// Solana keypair 文件格式: JSON 数组，包含 64 个字节 [私钥(32) + 公钥(32)]
func LoadProgramIDFromKeypair(keypairPath string) (solana.PublicKey, error) {
	// 读取文件
	data, err := os.ReadFile(keypairPath)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("无法读取 keypair 文件: %w", err)
	}

	// 解析 JSON 数组
	var keypairBytes []byte
	if err := json.Unmarshal(data, &keypairBytes); err != nil {
		return solana.PublicKey{}, fmt.Errorf("无法解析 keypair JSON: %w", err)
	}

	// keypair 应该是 64 字节: 前 32 字节私钥 + 后 32 字节公钥
	if len(keypairBytes) != 64 {
		return solana.PublicKey{}, fmt.Errorf("keypair 格式错误: 期望 64 字节, 实际 %d 字节", len(keypairBytes))
	}

	// 提取公钥 (后 32 字节)
	var pubkey solana.PublicKey
	copy(pubkey[:], keypairBytes[32:64])

	return pubkey, nil
}

// DeriveOrderPDA 派生订单 PDA 地址
func DeriveOrderPDA(passenger solana.PublicKey, orderID uint8, programID solana.PublicKey) (solana.PublicKey, uint8) {
	// 派生 PDA: Seeds = ["order", passenger_pubkey, order_id_byte]
	pda, bump, _ := solana.FindProgramAddress(
		[][]byte{
			[]byte("order"),
			passenger.Bytes(),
			{orderID},
		},
		programID,
	)
	return pda, bump
}

func main() {
	// ========== 配置 ==========
	// 从 keypair 文件动态读取程序 ID
	// 获取当前可执行文件所在目录，构建 keypair 文件的绝对路径
	execDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("无法获取当前目录: %v", err)
	}
	keypairPath := filepath.Join(execDir, ProgramKeypairPath)

	programID, err := LoadProgramIDFromKeypair(keypairPath)
	if err != nil {
		log.Fatalf("无法加载程序 ID: %v", err)
	}
	fmt.Printf("📋 程序 ID (从 keypair 读取): %s\n", programID)

	// 本地验证器 RPC
	rpcClient := rpc.New("http://127.0.0.1:8899")
	ctx := context.Background()

	// 加载本地钱包 (乘客)
	// 使用 Solana CLI 默认的 keypair
	passenger, err := solana.PrivateKeyFromSolanaKeygenFile("/Users/liyixin/.config/solana/id.json")
	if err != nil {
		log.Fatalf("无法加载钱包: %v", err)
	}
	passengerPubkey := passenger.PublicKey()
	fmt.Printf("🚗 乘客钱包地址: %s\n", passengerPubkey)

	// 检查余额
	balance, err := rpcClient.GetBalance(ctx, passengerPubkey, rpc.CommitmentConfirmed)
	if err != nil {
		log.Fatalf("无法获取余额: %v", err)
	}
	fmt.Printf("💰 当前余额: %d lamports (%.4f SOL)\n", balance.Value, float64(balance.Value)/1e9)

	// ========== 构建订单 ==========
	orderID := uint8(3)
	price := uint64(100000000) // 0.1 SOL = 100_000_000 lamports

	// 准备 Memo (16 bytes)
	var memo [16]byte
	copy(memo[:], "Airport->Hotel")

	// 派生 PDA
	orderPDA, bump := DeriveOrderPDA(passengerPubkey, orderID, programID)
	fmt.Printf("📦 订单 PDA: %s (bump: %d)\n", orderPDA, bump)

	// ========== 构建指令 ==========
	instruction := CreateOrderInstruction{
		InstructionID: 1, // CreateOrder
		OrderID:       orderID,
		Price:         price,
		Memo:          memo,
	}

	instructionData, err := instruction.Serialize()
	if err != nil {
		log.Fatalf("序列化指令失败: %v", err)
	}
	fmt.Printf("📝 指令数据 (%d bytes): %x\n", len(instructionData), instructionData)

	// 构建 Solana 指令
	solanaInstruction := solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(passengerPubkey).SIGNER().WRITE(), // 乘客 (signer, writable)
			solana.Meta(orderPDA).WRITE(),                 // PDA (writable, NOT signer)
			solana.Meta(solana.SystemProgramID),           // System Program (read-only)
		},
		instructionData,
	)

	// ========== 构建并发送交易 ==========
	recentBlockhash, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("获取 blockhash 失败: %v", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{solanaInstruction},
		recentBlockhash.Value.Blockhash,
		solana.TransactionPayer(passengerPubkey),
	)
	if err != nil {
		log.Fatalf("构建交易失败: %v", err)
	}

	// 签名
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(passengerPubkey) {
			return &passenger
		}
		return nil
	})
	if err != nil {
		log.Fatalf("签名失败: %v", err)
	}

	// 发送交易
	fmt.Println("\n🚀 发送交易...")
	sig, err := rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		log.Fatalf("发送交易失败: %v", err)
	}

	fmt.Printf("✅ 交易已发送! 签名: %s\n", sig)
	fmt.Printf("🔗 Explorer: https://explorer.solana.com/tx/%s?cluster=custom&customUrl=http://127.0.0.1:8899\n", sig)

	// ========== 验证订单创建 ==========
	fmt.Println("\n⏳ 等待确认...")
	// 简单等待
	fmt.Println("💡 可以使用以下命令查看订单账户数据:")
	fmt.Printf("   solana account %s -u http://127.0.0.1:8899\n", orderPDA)
}
