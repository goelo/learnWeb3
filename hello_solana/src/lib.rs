use solana_program::{
    account_info::AccountInfo, entrypoint, entrypoint::ProgramResult, pubkey::Pubkey,
};

// 1. 声明入口点
entrypoint!(process_instrction);

// 2 define process logic
pub fn process_instrction(
    program_id: &Pubkey, // program id
    accounts: &[AccountInfo],
    instruction_data: &[u8],
) -> ProgramResult {
    solana_program::msg!("hello solana!");
    Ok(())
}
