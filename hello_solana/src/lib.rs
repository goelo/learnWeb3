use solana_program::{
    account_info::AccountInfo, entrypoint, entrypoint::ProgramResult, pubkey::Pubkey,
};

use borsh::{BorshDeserialize, BorshSerialize};

// 定义结构体
// #[derive(...)]是rust的魔法，自动帮我们写好代码
#[derive(BorshDeserialize, BorshSerialize, Debug)]
pub struct HelloInstruction {
    pub message: String,
}
// 1. 声明入口点
entrypoint!(process_instrction);

// 2 define process logic
pub fn process_instrction(
    program_id: &Pubkey, // program id
    accounts: &[AccountInfo],
    instruction_data: &[u8],
) -> ProgramResult {
    let instruction = HelloInstruction::try_from_slice(instruction_data).unwrap();
    solana_program::msg!("接受来自客户端的消息: {}", instruction.message);
    Ok(())
}
