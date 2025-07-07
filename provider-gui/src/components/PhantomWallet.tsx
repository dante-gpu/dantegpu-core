import React, { useState, useEffect } from 'react';
import { invoke } from '@tauri-apps/api/tauri';

interface PhantomWalletProps {
  onWalletConnected: (walletAddress: string, balance: number) => void;
  onPaymentComplete: (transactionHash: string) => void;
  onError: (error: string) => void;
}

interface WalletState {
  isConnected: boolean;
  address: string | null;
  balance: number;
  isConnecting: boolean;
  publicKey: string | null;
}

interface PaymentRequest {
  amount: number;
  currency: 'SOL' | 'USDC' | 'DGPU';
  recipient: string;
  description: string;
  jobId?: string;
}

interface TransactionStatus {
  hash: string;
  status: 'pending' | 'confirmed' | 'failed';
  confirmations: number;
  timestamp: string;
}

export const PhantomWallet: React.FC<PhantomWalletProps> = ({ 
  onWalletConnected, 
  onPaymentComplete, 
  onError 
}) => {
  const [walletState, setWalletState] = useState<WalletState>({
    isConnected: false,
    address: null,
    balance: 0,
    isConnecting: false,
    publicKey: null
  });
  
  const [paymentRequest, setPaymentRequest] = useState<PaymentRequest | null>(null);
  const [isPaymentPending, setIsPaymentPending] = useState(false);
  const [transactionHistory, setTransactionHistory] = useState<TransactionStatus[]>([]);
  const [showPaymentModal, setShowPaymentModal] = useState(false);

  // Connect to Phantom wallet
  const connectWallet = async () => {
    setWalletState(prev => ({ ...prev, isConnecting: true }));
    
    try {
      const response = await invoke<string>('connect_phantom_wallet');
      const walletData = JSON.parse(response);
      
      if (walletData.connected) {
        setWalletState({
          isConnected: true,
          address: walletData.address,
          balance: walletData.balance_dgpu,
          isConnecting: false,
          publicKey: walletData.address
        });
        onWalletConnected(walletData.address, walletData.balance_dgpu);
      } else {
        onError('Failed to connect to Phantom wallet');
      }
    } catch (error) {
      console.error('Failed to connect wallet:', error);
      onError(`Failed to connect wallet: ${error}`);
      setWalletState(prev => ({ ...prev, isConnecting: false }));
    }
  };

  // Disconnect wallet
  const disconnectWallet = async () => {
    setWalletState({
      isConnected: false,
      address: null,
      balance: 0,
      isConnecting: false,
      publicKey: null
    });
  };

  // Get wallet balance
  const getWalletBalance = async (publicKey: string): Promise<number> => {
    try {
      const balance = await invoke<number>('get_dgpu_balance', { walletAddress: publicKey });
      return balance;
    } catch (error) {
      console.error('Failed to get wallet balance:', error);
      return 0;
    }
  };

  // Create payment request
  const createPaymentRequest = (request: PaymentRequest) => {
    setPaymentRequest(request);
    setShowPaymentModal(true);
  };

  // Process payment
  const processPayment = async () => {
    if (!paymentRequest || !walletState.isConnected) {
      onError('Wallet not connected or payment request missing');
      return;
    }

    setIsPaymentPending(true);
    
    try {
      const response = await invoke<string>('process_dgpu_payment', {
        amountDgpu: paymentRequest.amount,
        recipientAddress: paymentRequest.recipient
      });
      
      const transactionData = JSON.parse(response);
      
      // Create transaction status
      const status: TransactionStatus = {
        hash: transactionData.transaction_hash,
        status: 'confirmed',
        confirmations: 32,
        timestamp: transactionData.timestamp
      };

      // Add to transaction history
      setTransactionHistory(prev => [status, ...prev]);

      // Update balance
      const newBalance = walletState.balance - paymentRequest.amount;
      setWalletState(prev => ({ ...prev, balance: newBalance }));

      onPaymentComplete(transactionData.transaction_hash);
      setShowPaymentModal(false);
      setPaymentRequest(null);
    } catch (error) {
      console.error('Payment failed:', error);
      onError(`Payment failed: ${error}`);
    } finally {
      setIsPaymentPending(false);
    }
  };

  // Refresh wallet balance
  const refreshBalance = async () => {
    if (walletState.address) {
      const balance = await getWalletBalance(walletState.address);
      setWalletState(prev => ({ ...prev, balance }));
    }
  };

  // Auto-refresh balance every 30 seconds
  useEffect(() => {
    if (walletState.isConnected) {
      const interval = setInterval(refreshBalance, 30000);
      return () => clearInterval(interval);
    }
  }, [walletState.isConnected]);

  return (
    <div className="phantom-wallet-container">
      <div className="wallet-header">
        <h3>Phantom Wallet</h3>
        <button 
          onClick={refreshBalance} 
          disabled={!walletState.isConnected}
          className="refresh-button"
        >
          Refresh
        </button>
      </div>

      {!walletState.isConnected ? (
        <div className="wallet-disconnected">
          <p>Connect to your Phantom wallet for dGPU payments</p>
          <p className="wallet-info-text">
            This will connect to your existing Phantom wallet to process dGPU token payments
          </p>
          <button 
            onClick={connectWallet} 
            disabled={walletState.isConnecting}
            className="connect-button"
          >
            {walletState.isConnecting ? 'Connecting...' : 'Connect Phantom Wallet'}
          </button>
          <div className="wallet-features">
            <p>✅ dGPU Token Support</p>
            <p>✅ Secure Payment Processing</p>
            <p>✅ Real-time Balance Updates</p>
          </div>
        </div>
      ) : (
        <div className="wallet-connected">
          <div className="wallet-info">
            <div className="wallet-address">
              <strong>Address:</strong>
              <span className="address-text">
                {walletState.address?.slice(0, 8)}...{walletState.address?.slice(-8)}
              </span>
            </div>
            <div className="wallet-balance">
              <strong>dGPU Balance:</strong> {walletState.balance.toFixed(4)} dGPU
            </div>
          </div>

          <div className="wallet-actions">
            <button 
              onClick={() => createPaymentRequest({
                amount: 10.0,
                currency: 'DGPU',
                recipient: '7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump',
                description: 'Test dGPU payment for GPU rental'
              })}
              className="test-payment-button"
            >
              Test dGPU Payment
            </button>
            <button onClick={disconnectWallet} className="disconnect-button">
              Disconnect
            </button>
          </div>
        </div>
      )}

      {/* Transaction History */}
      {transactionHistory.length > 0 && (
        <div className="transaction-history">
          <h4>Recent Transactions</h4>
          <div className="transaction-list">
            {transactionHistory.slice(-5).map((tx, index) => (
              <div key={index} className="transaction-item">
                <div className="transaction-hash">
                  {tx.hash.slice(0, 8)}...{tx.hash.slice(-8)}
                </div>
                <div className={`transaction-status ${tx.status}`}>
                  {tx.status}
                </div>
                <div className="transaction-time">
                  {new Date(tx.timestamp).toLocaleTimeString()}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Payment Modal */}
      {showPaymentModal && paymentRequest && (
        <div className="payment-modal-backdrop">
          <div className="payment-modal">
            <h3>Confirm Payment</h3>
            <div className="payment-details">
              <p><strong>Amount:</strong> {paymentRequest.amount} {paymentRequest.currency}</p>
              <p><strong>To:</strong> {paymentRequest.recipient}</p>
              <p><strong>Description:</strong> {paymentRequest.description}</p>
              {paymentRequest.jobId && <p><strong>Job ID:</strong> {paymentRequest.jobId}</p>}
            </div>
            <div className="payment-actions">
              <button 
                onClick={processPayment} 
                disabled={isPaymentPending}
                className="confirm-payment-button"
              >
                {isPaymentPending ? 'Processing...' : 'Confirm Payment'}
              </button>
              <button 
                onClick={() => setShowPaymentModal(false)} 
                disabled={isPaymentPending}
                className="cancel-payment-button"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export type { PaymentRequest, TransactionStatus, WalletState }; 