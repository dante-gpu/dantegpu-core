import { Connection, PublicKey, Transaction } from '@solana/web3.js';
import { PhantomWalletAdapter } from '@solana/wallet-adapter-phantom';
import { SolflareWalletAdapter } from '@solana/wallet-adapter-solflare';

const SOLANA_RPC_URL = import.meta.env.VITE_SOLANA_RPC_URL || 'https://api.mainnet-beta.solana.com';
const DGPU_TOKEN_MINT = '7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump';

export type WalletType = 'phantom' | 'solflare';

class WalletService {
  private connection: Connection;
  private adapter: PhantomWalletAdapter | SolflareWalletAdapter | null = null;
  private walletType: WalletType | null = null;

  constructor() {
    this.connection = new Connection(SOLANA_RPC_URL, 'confirmed');
  }

  async connect(walletType: WalletType): Promise<string> {
    try {
      if (walletType === 'phantom') {
        this.adapter = new PhantomWalletAdapter();
      } else if (walletType === 'solflare') {
        this.adapter = new SolflareWalletAdapter();
      } else {
        throw new Error('Unsupported wallet type');
      }

      await this.adapter.connect();
      this.walletType = walletType;

      if (!this.adapter.publicKey) {
        throw new Error('Failed to get wallet address');
      }

      return this.adapter.publicKey.toString();
    } catch (error) {
      console.error('Wallet connection error:', error);
      throw error;
    }
  }

  async disconnect(): Promise<void> {
    if (this.adapter) {
      await this.adapter.disconnect();
      this.adapter = null;
      this.walletType = null;
    }
  }

  getAddress(): string | null {
    return this.adapter?.publicKey?.toString() || null;
  }

  isConnected(): boolean {
    return this.adapter?.connected || false;
  }

  getWalletType(): WalletType | null {
    return this.walletType;
  }

  async getBalance(address: string): Promise<number> {
    try {
      const publicKey = new PublicKey(address);
      const balance = await this.connection.getBalance(publicKey);
      return balance / 1e9; // Convert lamports to SOL
    } catch (error) {
      console.error('Failed to get balance:', error);
      throw error;
    }
  }

  async getTokenBalance(address: string): Promise<number> {
    try {
      const publicKey = new PublicKey(address);
      const tokenMint = new PublicKey(DGPU_TOKEN_MINT);

      // Get associated token account
      const tokenAccounts = await this.connection.getParsedTokenAccountsByOwner(publicKey, {
        mint: tokenMint,
      });

      if (tokenAccounts.value.length === 0) {
        return 0;
      }

      const balance = tokenAccounts.value[0].account.data.parsed.info.tokenAmount.uiAmount;
      return balance || 0;
    } catch (error) {
      console.error('Failed to get token balance:', error);
      return 0;
    }
  }

  async signTransaction(transaction: Transaction): Promise<Transaction> {
    if (!this.adapter || !this.adapter.publicKey) {
      throw new Error('Wallet not connected');
    }

    try {
      const signedTransaction = await this.adapter.signTransaction(transaction);
      return signedTransaction;
    } catch (error) {
      console.error('Failed to sign transaction:', error);
      throw error;
    }
  }

  async sendTransaction(transaction: Transaction): Promise<string> {
    if (!this.adapter || !this.adapter.publicKey) {
      throw new Error('Wallet not connected');
    }

    try {
      const signedTransaction = await this.signTransaction(transaction);
      const signature = await this.connection.sendRawTransaction(signedTransaction.serialize());
      
      // Wait for confirmation
      await this.connection.confirmTransaction(signature, 'confirmed');
      
      return signature;
    } catch (error) {
      console.error('Failed to send transaction:', error);
      throw error;
    }
  }

  async requestAirdrop(address: string, amount: number): Promise<string> {
    try {
      const publicKey = new PublicKey(address);
      const signature = await this.connection.requestAirdrop(
        publicKey,
        amount * 1e9 // Convert SOL to lamports
      );
      
      await this.connection.confirmTransaction(signature, 'confirmed');
      return signature;
    } catch (error) {
      console.error('Failed to request airdrop:', error);
      throw error;
    }
  }

  async getTransactionHistory(address: string, limit: number = 10): Promise<any[]> {
    try {
      const publicKey = new PublicKey(address);
      const signatures = await this.connection.getSignaturesForAddress(publicKey, { limit });
      
      const transactions = await Promise.all(
        signatures.map(async (sig) => {
          const tx = await this.connection.getParsedTransaction(sig.signature, {
            maxSupportedTransactionVersion: 0,
          });
          return {
            signature: sig.signature,
            blockTime: sig.blockTime,
            status: sig.confirmationStatus,
            transaction: tx,
          };
        })
      );

      return transactions;
    } catch (error) {
      console.error('Failed to get transaction history:', error);
      throw error;
    }
  }

  getExplorerUrl(signature: string): string {
    return `https://solscan.io/tx/${signature}`;
  }

  isPhantomInstalled(): boolean {
    return typeof window !== 'undefined' && 'solana' in window && (window as any).solana?.isPhantom;
  }

  isSolflareInstalled(): boolean {
    return typeof window !== 'undefined' && 'solflare' in window;
  }

  getInstalledWallets(): WalletType[] {
    const wallets: WalletType[] = [];
    if (this.isPhantomInstalled()) wallets.push('phantom');
    if (this.isSolflareInstalled()) wallets.push('solflare');
    return wallets;
  }
}

export const walletService = new WalletService();
export default walletService;

