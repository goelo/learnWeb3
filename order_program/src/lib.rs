use borsh::{BorshDeserialize, BorshSerialize};

use solana_program::{
    account_info::{next_account_info, AccountInfo},
    entrypoint,
    entrypoint::ProgramResult,
    msg,
    program::invoke_signed,
    program_error::ProgramError,
    pubkey::Pubkey,
    rent::Rent,
    system_instruction,
    sysvar::Sysvar,
};

// 1. 定义指令数据的负载 (Payload)
// 这里需要跟 Go 里的 Price 和 Memo 对应
#[derive(BorshDeserialize, BorshSerialize, Debug)]
pub struct OrderInstructionData {
    pub orderId: u8, // orderId
    pub price: u64,     // 对应 Go 的 uint64 (8 bytes)
    pub memo: [u8; 16], // 对应 Go 的 [u8; 16]
}
// 2. 定义账户内存储的数据结构 (Account Data)
#[derive(BorshDeserialize, BorshSerialize, Debug)]
pub struct OrderAccount {
    pub status: u8,
    pub passenger: Pubkey,
    pub order_id: u8,
    pub price: u64,
    pub memo: [u8; 16],
    pub bump: u8,
}

// 声明程序入口点
entrypoint!(process_instruction);

pub fn process_instruction(
    program_id: &Pubkey,
    accounts: &[AccountInfo],
    instruction_data: &[u8],
) -> ProgramResult {
    // A. 解析指令 ID (Discriminator)
    let (tag, rest) = instruction_data
        .split_first()
        .ok_or(ProgramError::InvalidInstructionData)?;

    // 判断指令 ID，假设 1 代表 "CreateOrder"
    match tag {
        1 => process_create_order(program_id, accounts, rest),
        _ => {
            msg!("Error: Unknown Instruct ID");
            Err(ProgramError::InvalidInstructionData)
        }
    }
}

// ==========================================
// 3. 核心处理逻辑
// ==========================================
fn process_create_order(
    program_id: &Pubkey,
    accounts: &[AccountInfo],
    instruction_data: &[u8],
) -> ProgramResult {
    msg!("Instruction: Create Order");

    // --- 步骤 1: 解析账户列表 ---
    // 顺序必须与 Go 客户端传入的顺序完全一致
    let account_info_iter = &mut accounts.iter();
    let passenger_info = next_account_info(account_info_iter)?; // 0. 乘客 (Signer, Writable)
    let pda_info = next_account_info(account_info_iter)?; // 1. 订单PDA (Writable)
    let system_program_info = next_account_info(account_info_iter)?; // 2. 系统程序

    // 基本的安全检查
    if !passenger_info.is_signer {
        return Err(ProgramError::MissingRequiredSignature);
    }

    // --- 步骤 2: 反序列化指令数据 ---
    // 解析 Price 和 Memo
    let payload = OrderInstructionData::try_from_slice(instruction_data)
    .map_err(|_| ProgramError::InvalidInstructionData)?;

    msg!("OrderId: {}, Memo received", payload.orderId);

    // --- 步骤 3: 准备 PDA 种子和 Bump ---
    // 注意：这里我们需要找到 bump 才能签名
    // 在实际生产中，bump 通常也是由客户端传进来以节省计算资源
    // 但为了演示完整性，我们在链上再算一次，或者假设客户端把 bump 放在了数据里
    // 这里演示链上重新计算验证（更安全但费一点点 gas）
    let (pda_address, bump_seed) = Pubkey::find_program_address(
        &[
            b"order",
            passenger_info.key.as_ref(),
            &[payload.orderId] // 假设我们要用 orderId 做种子
            // 注意：如果你的 Go 代码里用的是 OrderID，这里必须通过指令传 OrderID 进来
        ],
        program_id,
    );

    // 校验传入的 PDA 账户地址是否正确
    if pda_info.key != &pda_address {
        msg!("Error: PDA address mismatch");
        return Err(ProgramError::InvalidSeeds);
    }

    // --- 步骤 4: 计算租金 (Rent) ---
    // status(1) + passenger(32) + price(8) + memo(16) + order_id(1) + bump(1) = 59 bytes
    let account_len = 59;  // 之前算过的长度
    let rent = Rent::get()?;
    let lamports_required = rent.minimum_balance(account_len);

    // --- 步骤 5: CPI 调用系统程序创建账户 ---
    // 构造签名种子
    // 注意：这里需要根据你实际的种子逻辑来。
    // 假设种子是 ["order", passenger_pubkey, orderId_bytes, bump]  
    let signers_seeds : &[&[u8]] = &[
        b"order",
        passenger_info.key.as_ref(),
        &[payload.orderId],
        &[bump_seed],
    ];
    invoke_signed(
        &system_instruction::create_account(
            passenger_info.key, // 谁付钱
            pda_info.key, // 创建的账户
            lamports_required,//多少钱
            account_len as u64, // 多大空间
            program_id, // 归谁管理，owner
        ),
        &[
            passenger_info.clone(),
            pda_info.clone(),
            system_program_info.clone(),
        ],
        &[signers_seeds], // 关键：带着 PDA 的“身份证”去签名
    )?;

    msg!("Order PDA Account Created!");

    // --- 步骤 6: 初始化账户数据 ---
    // 账户刚创建时是全 0 的，我们需要把数据填进去
    let mut order_data = OrderAccount::try_from_slice(&pda_info.data.borrow())?;

    order_data.status = 0; // 0 = created
    order_data.passenger = *passenger_info.key;
    order_data.order_id = payload.orderId;
    order_data.price = payload.price;
    order_data.memo = payload.memo;
    order_data.bump = bump_seed;

    // 将结构体写回账户内存
    // serialize 写入的是 &mut [u8]
    order_data.serialize(&mut &mut pda_info.data.borrow_mut()[..])?;

    msg!("Order Data Initialized successfully");

    Ok(())
}
