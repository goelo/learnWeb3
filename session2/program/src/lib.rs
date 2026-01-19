use borsh::{BorshDeserialize, BorshSerialize};
use solana_program::pubkey::Pubkey;

solana_program::entrypoint!(process_instruction);

#[derive(BorshSerialize, BorshDeserialize)]
pub enum StorageInstruction {
    InitializeConfig { admin_key: Pubkey },
}

#[derive(BorshSerialize, BorshDeserialize, Debug)]
pub struct AdminConfig {
    pub admin_key:Pubkey,
}
pub fn process_instruction(
    program_id: &solana_program::pubkey::Pubkey,
    accounts: &[solana_program::account_info::AccountInfo],
    data: &[u8], // 这里的 data 是前端传来的原始字节
) -> solana_program::entrypoint::ProgramResult {
    // 使用 try_from_slice 将字节解析成枚举
    // 注意末尾的 ?，如果解析失败（比如 data 长度不对），会直接报错返回
    let instruction = StorageInstruction::try_from_slice(data)?;

    let account_info = &mut accounts.iter();
    match instruction {
        StorageInstruction::InitializeConfig { admin_key } => {
            let payer_account = next_account_info(accounts_iter)?;
            let config_pda_account = next_account_info(accounts_iter)?;
            let system_program = next_account_info(accounts_iter)?;

            if !payer_account.is_signer {
                return Err(ProgramError::MissingRequiredSignature);
            }
// 1. 获取租金豁免所需的最小余额
            let rent = Rent::get()?;
            let space = 32// 目前只存一个 Pubkey
            let rent_lamports = rent.minimum_balance(space);

            let signers_seeds: &[&[u8]] = &[
                b"admin_config",
                &[bump],
            ];
            solana_program::program::invoke_signed(
                &solana_program::system_instruction::create_account(
                    payer_account.key,
                    config_pda_account.key,
                    rent_lamports,
                    space as u64,
                    program_id,
                ),
                &[
                    payer_account.clone(),
                    config_pda_account.clone(),
                    system_program.clone(),
                ],
                &[signers_seeds],
            )?;
        }
    }
    solana_program::msg!("hello solana!");
    Ok(())
}
