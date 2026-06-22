import { useState } from "react";
import { useConnection, useWallet } from "@solana/wallet-adapter-react";
import { PublicKey, Transaction } from "@solana/web3.js";
import {
  getAssociatedTokenAddress,
  getAccount,
  createAssociatedTokenAccountInstruction,
  createTransferCheckedInstruction,
  TokenAccountNotFoundError,
} from "@solana/spl-token";
import { USDC_MINT, USDC_DECIMALS, PLATFORM_DEPOSIT_ADDRESS, toUsdcAtoms } from "@/lib/solana";

interface DepositResult {
  signature: string;
}

// Builds and sends an on-chain USDC transfer from the connected wallet to the
// platform deposit address, creating the destination token account if needed.
// Returns the confirmed signature so the caller can notify the billing service.
export function useDeposit() {
  const { connection } = useConnection();
  const { publicKey, sendTransaction } = useWallet();
  const [busy, setBusy] = useState(false);

  async function deposit(amountUsdc: number): Promise<DepositResult> {
    if (!publicKey) throw new Error("Connect your wallet first.");
    if (!PLATFORM_DEPOSIT_ADDRESS) throw new Error("Platform deposit address is not configured.");
    if (amountUsdc <= 0) throw new Error("Enter an amount greater than zero.");

    setBusy(true);
    try {
      const mint = new PublicKey(USDC_MINT);
      const platform = new PublicKey(PLATFORM_DEPOSIT_ADDRESS);

      const source = await getAssociatedTokenAddress(mint, publicKey);
      const dest = await getAssociatedTokenAddress(mint, platform);

      const tx = new Transaction();

      // If the platform's token account is missing, the renter funds its rent so
      // the transfer can land. (Normally it already exists.)
      try {
        await getAccount(connection, dest);
      } catch (err) {
        if (err instanceof TokenAccountNotFoundError) {
          tx.add(createAssociatedTokenAccountInstruction(publicKey, dest, platform, mint));
        } else {
          throw err;
        }
      }

      tx.add(
        createTransferCheckedInstruction(
          source,
          mint,
          dest,
          publicKey,
          toUsdcAtoms(amountUsdc),
          USDC_DECIMALS,
        ),
      );

      const signature = await sendTransaction(tx, connection);
      const latest = await connection.getLatestBlockhash();
      await connection.confirmTransaction(
        { signature, blockhash: latest.blockhash, lastValidBlockHeight: latest.lastValidBlockHeight },
        "confirmed",
      );

      return { signature };
    } finally {
      setBusy(false);
    }
  }

  return { deposit, busy };
}
