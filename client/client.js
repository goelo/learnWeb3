const {
    Connection,
    PublicKey,
    Keypair,
    Transaction,
    TransactionInstruction,
    sendAndConfirmTransaction,
} = require("@solana/web3.js");
const fs = require("fs");
const os = require("os");

async function main() {
    console.log("正在启动客户端...");

    // 1. 连接到本地测试网络
    const connection = new Connection("http://localhost:8899", "confirmed");

    // 2. 获取你的钱包 (Payer)
    // 默认路径通常在 ~/.config/solana/id.json
    const walletPath = os.homedir() + "/.config/solana/id.json";
    const secretKey = Uint8Array.from(JSON.parse(fs.readFileSync(walletPath)));
    const payer = Keypair.fromSecretKey(secretKey);

    console.log("当前钱包地址:", payer.publicKey.toBase58());

    // 3. 填入你的程序 ID
    // 👇👇👇 请在这里填入你刚才获得的那个 ID 👇👇👇
    const programId = new PublicKey("CuRF5bMpCoatpfGTKy7H99JoAseKEUCrENzFv9yHTnG4");

    console.log("目标程序 ID:", programId.toBase58());
    // ... 前面的代码 ...

    // 4. 创建指令 (Instruction)
    // 这里我们构建一个简单的指令：
    // - keys: 涉及的账户列表。至少要把你自己(payer)放进去，因为你要付钱。
    // - programId: 我们的目标程序。
    // - data: 传递给程序的参数。因为我们的 Rust 程序里暂时没处理参数，发个空包就行。
    const instruction = new TransactionInstruction({
        keys: [
            { pubkey: payer.publicKey, isSigner: true, isWritable: true }
        ],
        programId: programId,
        data: Buffer.alloc(0), // 空的字节数组
    });

    // 5. 将指令添加到交易对象中
    const transaction = new Transaction().add(instruction);

    console.log("正在发送交易...");

    // 6. 发送并确认交易
    //这一步会自动：获取最新区块哈希 -> 用你的私钥签名 -> 发送到网络 -> 等待网络确认
    const signature = await sendAndConfirmTransaction(
        connection,
        transaction,
        [payer] // 签名者列表
    );

    console.log("✅ 交易成功！");
    console.log("交易哈希 (Signature):", signature);
    console.log(`查看交易详情: solana confirm -v ${signature} --url localhost`);
}

// 这里的 main().catch... 已经在你之前的代码里有了，不用重复复制

main().catch(err => {
    console.error(err);
});