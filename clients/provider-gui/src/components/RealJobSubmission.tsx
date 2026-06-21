import React, { useState } from 'react';
import { invoke } from '@tauri-apps/api/tauri';
import { RealPhantomWallet } from './RealPhantomWallet';
import './JobSubmission.css';

interface RealJobSubmissionProps {
  onJobSubmitted: (jobId: string) => void;
  availableGpus: GpuInfo[];
}

interface GpuInfo {
  id: string;
  name: string;
  model: string;
  vram_total_mb: number;
  vram_free_mb: number;
  utilization_gpu_percent?: number;
  is_available_for_rent: boolean;
  current_hourly_rate_dgpu: number | null;
}

interface JobSubmissionForm {
  jobName: string;
  jobType: 'machine_learning' | 'cryptocurrency_mining' | 'scientific_computing' | 'rendering' | 'custom_compute';
  gpuId: string;
  duration: number;
  resources: {
    gpuMemoryMb: number;
    cpuCores: number;
    ramMb: number;
    storageMb: number;
  };
  jobConfig: {
    dockerImage: string;
    environmentVariables: Record<string, string>;
    command: string;
    args: string[];
    inputFiles: string[];
    expectedOutputs: string[];
  };
}

export const RealJobSubmission: React.FC<RealJobSubmissionProps> = ({ onJobSubmitted, availableGpus }) => {
  const [form, setForm] = useState<JobSubmissionForm>({
    jobName: '',
    jobType: 'machine_learning',
    gpuId: '',
    duration: 1,
    resources: {
      gpuMemoryMb: 8000,
      cpuCores: 4,
      ramMb: 16000,
      storageMb: 50000
    },
    jobConfig: {
      dockerImage: 'pytorch/pytorch:latest',
      environmentVariables: {},
      command: 'python',
      args: [],
      inputFiles: [],
      expectedOutputs: []
    }
  });
  
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);
  const [walletConnected, setWalletConnected] = useState(false);
  const [walletAddress, setWalletAddress] = useState<string>('');
  const [walletBalance, setWalletBalance] = useState<number>(0);
  const [paymentCompleted, setPaymentCompleted] = useState(false);
  const [paymentTransactionHash, setPaymentTransactionHash] = useState<string>('');
  const [showWallet, setShowWallet] = useState(false);

  const getFrameworkFromJobType = (jobType: JobSubmissionForm['jobType']): string => {
    switch (jobType) {
      case 'machine_learning':
        return 'pytorch';
      case 'scientific_computing':
        return 'tensorflow';
      case 'cryptocurrency_mining':
        return 'cuda';
      case 'rendering':
        return 'blender';
      default:
        return 'custom';
    }
  };

  const validateForm = (): boolean => {
    const errors: string[] = [];
    
    if (!form.jobName.trim()) {
      errors.push('Job name is required');
    }
    
    if (!form.gpuId) {
      errors.push('Please select a GPU');
    }
    
    if (form.duration <= 0) {
      errors.push('Duration must be greater than 0');
    }
    
    if (!form.jobConfig.dockerImage.trim()) {
      errors.push('Docker image is required');
    }
    
    if (!walletConnected) {
      errors.push('Wallet must be connected');
    }
    
    if (!paymentCompleted) {
      errors.push('Payment must be completed before job submission');
    }
    
    setValidationErrors(errors);
    return errors.length === 0;
  };

  const calculateJobCost = (): number => {
    const selectedGpu = availableGpus.find(gpu => gpu.id === form.gpuId);
    if (!selectedGpu || !selectedGpu.current_hourly_rate_dgpu) return 0;
    return selectedGpu.current_hourly_rate_dgpu * form.duration;
  };

  const handleWalletConnected = (address: string, balance: number) => {
    setWalletConnected(true);
    setWalletAddress(address);
    setWalletBalance(balance);
  };

  const handleWalletDisconnected = () => {
    setWalletConnected(false);
    setWalletAddress('');
    setWalletBalance(0);
    setPaymentCompleted(false);
    setPaymentTransactionHash('');
  };

  const handlePaymentComplete = (transactionHash: string) => {
    setPaymentCompleted(true);
    setPaymentTransactionHash(transactionHash);
    setShowWallet(false);
  };

  const handleWalletError = (error: string) => {
    setValidationErrors([error]);
  };

  const processPayment = async () => {
    if (!walletConnected) {
      setValidationErrors(['Please connect your wallet first']);
      return;
    }

    const cost = calculateJobCost();
    if (walletBalance < cost) {
      setValidationErrors([`Insufficient balance. Required: ${cost} dGPU, Available: ${walletBalance} dGPU`]);
      return;
    }

    try {
      // The actual payment processing will be handled by the RealPhantomWallet component
      setShowWallet(true);
    } catch (error) {
      setValidationErrors([`Payment initiation failed: ${error}`]);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) return;

    setIsSubmitting(true);
    
    try {
      const jobSpecification = {
        job_name: form.jobName,
        job_description: `${form.jobType} job - ${form.jobName}`,
        framework: getFrameworkFromJobType(form.jobType),
        python_version: "3.9",
        conda_environment: null,
        pip_requirements: [],
        environment_variables: form.jobConfig.environmentVariables,
        startup_script: null,
        data_sources: [],
        expected_outputs: form.jobConfig.expectedOutputs,
        estimated_completion_time: Math.floor(form.duration * 60),
        gpu_memory_requirements: form.resources.gpuMemoryMb,
        cpu_requirements: form.resources.cpuCores,
        ram_requirements: form.resources.ramMb,
        storage_requirements: form.resources.storageMb,
        network_requirements: true,
        checkpoint_frequency: 300,
        auto_restart_on_failure: true,
        max_retries: 3,
        priority: "normal"
      };

      // Include payment verification in job submission
      const jobResult = await invoke<string>('submit_gpu_job_with_payment', {
        jobSpecification,
        gpuId: form.gpuId,
        paymentTransactionHash,
        durationHours: form.duration,
        renterId: walletAddress,
        renterName: 'GPU Rental User'
      });

      const jobData = JSON.parse(jobResult);
      onJobSubmitted(jobData.job_id || jobData.booking_id);

      // Reset form
      setForm({
        jobName: '',
        jobType: 'machine_learning',
        gpuId: '',
        duration: 1,
        resources: {
          gpuMemoryMb: 8000,
          cpuCores: 4,
          ramMb: 16000,
          storageMb: 50000
        },
        jobConfig: {
          dockerImage: 'pytorch/pytorch:latest',
          environmentVariables: {},
          command: 'python',
          args: [],
          inputFiles: [],
          expectedOutputs: []
        }
      });
      
      setPaymentCompleted(false);
      setPaymentTransactionHash('');
      
    } catch (error) {
      console.error('Job submission failed:', error);
      setValidationErrors([`Job submission failed: ${error}`]);
    } finally {
      setIsSubmitting(false);
    }
  };

  const getFilteredGpus = () => {
    return availableGpus.filter(gpu => gpu.is_available_for_rent);
  };

  return (
    <div className="real-job-submission">
      <div className="submission-header">
        <h1>GPU Job Submission</h1>
        <p>Submit your GPU computing job with real dGPU token payments</p>
      </div>

      {/* Wallet Section */}
      <section className="wallet-section">
        <h2>Wallet Connection</h2>
        {!walletConnected ? (
          <div className="wallet-connect">
            <p>Connect your Phantom wallet to proceed with payment</p>
            <button onClick={() => setShowWallet(true)} className="connect-wallet-btn">
              Connect Phantom Wallet
            </button>
          </div>
        ) : (
          <div className="wallet-connected">
            <div className="wallet-info">
              <div className="wallet-detail">
                <label>Address:</label>
                <code>{walletAddress}</code>
              </div>
              <div className="wallet-detail">
                <label>dGPU Balance:</label>
                <span className="balance">{walletBalance.toFixed(6)} dGPU</span>
              </div>
            </div>
            <div className="wallet-actions">
              <button onClick={() => setShowWallet(true)} className="manage-wallet-btn">
                Manage Wallet
              </button>
            </div>
          </div>
        )}
      </section>

      {/* Job Submission Form */}
      <form onSubmit={handleSubmit} className="job-form">
        {/* Basic Job Information */}
        <section className="form-section">
          <h3>Job Information</h3>
          
          <div className="form-row">
            <label>Job Name:</label>
            <input
              type="text"
              value={form.jobName}
              onChange={(e) => setForm(prev => ({ ...prev, jobName: e.target.value }))}
              placeholder="Enter job name"
              required
            />
          </div>

          <div className="form-row">
            <label>Job Type:</label>
            <select
              value={form.jobType}
              onChange={(e) => setForm(prev => ({ ...prev, jobType: e.target.value as any }))}
              required
            >
              <option value="machine_learning">Machine Learning</option>
              <option value="cryptocurrency_mining">Cryptocurrency Mining</option>
              <option value="scientific_computing">Scientific Computing</option>
              <option value="rendering">3D Rendering</option>
              <option value="custom_compute">Custom Compute</option>
            </select>
          </div>

          <div className="form-row">
            <label>Duration (hours):</label>
            <input
              type="number"
              value={form.duration}
              onChange={(e) => setForm(prev => ({ ...prev, duration: parseFloat(e.target.value) }))}
              min="0.1"
              step="0.1"
              required
            />
          </div>
        </section>

        {/* GPU Selection */}
        <section className="form-section">
          <h3>GPU Selection</h3>
          
          <div className="gpu-selection">
            {getFilteredGpus().map(gpu => (
              <div
                key={gpu.id}
                className={`gpu-option ${form.gpuId === gpu.id ? 'selected' : ''}`}
                onClick={() => setForm(prev => ({ ...prev, gpuId: gpu.id }))}
              >
                <div className="gpu-info">
                  <h4>{gpu.name} ({gpu.model})</h4>
                  <p>VRAM: {gpu.vram_free_mb}MB free / {gpu.vram_total_mb}MB total</p>
                  <p>Rate: {gpu.current_hourly_rate_dgpu} dGPU/hour</p>
                  {gpu.utilization_gpu_percent !== undefined && (
                    <p>Utilization: {gpu.utilization_gpu_percent}%</p>
                  )}
                </div>
                <div className="gpu-status">
                  {form.gpuId === gpu.id && <span className="selected-indicator">Selected</span>}
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* Resource Configuration */}
        <section className="form-section">
          <h3>Resource Allocation</h3>
          
          <div className="resource-controls">
            <div className="form-row">
              <label>GPU Memory (MB):</label>
              <input
                type="number"
                value={form.resources.gpuMemoryMb}
                onChange={(e) => setForm(prev => ({ 
                  ...prev, 
                  resources: { ...prev.resources, gpuMemoryMb: parseInt(e.target.value) }
                }))}
                min="1000"
                max="50000"
              />
            </div>
            
            <div className="form-row">
              <label>CPU Cores:</label>
              <input
                type="number"
                value={form.resources.cpuCores}
                onChange={(e) => setForm(prev => ({ 
                  ...prev, 
                  resources: { ...prev.resources, cpuCores: parseInt(e.target.value) }
                }))}
                min="1"
                max="32"
              />
            </div>
            
            <div className="form-row">
              <label>RAM (MB):</label>
              <input
                type="number"
                value={form.resources.ramMb}
                onChange={(e) => setForm(prev => ({ 
                  ...prev, 
                  resources: { ...prev.resources, ramMb: parseInt(e.target.value) }
                }))}
                min="1000"
                max="128000"
              />
            </div>
            
            <div className="form-row">
              <label>Storage (MB):</label>
              <input
                type="number"
                value={form.resources.storageMb}
                onChange={(e) => setForm(prev => ({ 
                  ...prev, 
                  resources: { ...prev.resources, storageMb: parseInt(e.target.value) }
                }))}
                min="1000"
                max="500000"
              />
            </div>
          </div>
        </section>

        {/* Job Configuration */}
        <section className="form-section">
          <h3>Job Configuration</h3>
          
          <div className="form-row">
            <label>Docker Image:</label>
            <input
              type="text"
              value={form.jobConfig.dockerImage}
              onChange={(e) => setForm(prev => ({ 
                ...prev, 
                jobConfig: { ...prev.jobConfig, dockerImage: e.target.value }
              }))}
              placeholder="e.g., pytorch/pytorch:latest"
              required
            />
          </div>
          
          <div className="form-row">
            <label>Command:</label>
            <input
              type="text"
              value={form.jobConfig.command}
              onChange={(e) => setForm(prev => ({ 
                ...prev, 
                jobConfig: { ...prev.jobConfig, command: e.target.value }
              }))}
              placeholder="e.g., python"
              required
            />
          </div>
          
          <div className="form-row">
            <label>Arguments (comma-separated):</label>
            <input
              type="text"
              value={form.jobConfig.args.join(', ')}
              onChange={(e) => setForm(prev => ({ 
                ...prev, 
                jobConfig: { ...prev.jobConfig, args: e.target.value.split(',').map(arg => arg.trim()) }
              }))}
              placeholder="e.g., train.py, --epochs, 100"
            />
          </div>
        </section>

        {/* Cost and Payment */}
        <section className="form-section">
          <h3>Cost and Payment</h3>
          
          {form.gpuId && (
            <div className="cost-calculation">
              <div className="cost-details">
                <div className="cost-row">
                  <span>GPU Hourly Rate:</span>
                  <span>{availableGpus.find(gpu => gpu.id === form.gpuId)?.current_hourly_rate_dgpu || 0} dGPU/hour</span>
                </div>
                <div className="cost-row">
                  <span>Duration:</span>
                  <span>{form.duration} hours</span>
                </div>
                <div className="cost-row total">
                  <span>Total Cost:</span>
                  <span>{calculateJobCost().toFixed(6)} dGPU</span>
                </div>
              </div>
            </div>
          )}

          {/* Payment Status */}
          {paymentCompleted ? (
            <div className="payment-status completed">
              <h4>Payment Completed</h4>
              <div className="payment-info">
                <div className="payment-detail">
                  <label>Transaction Hash:</label>
                  <code>{paymentTransactionHash}</code>
                </div>
                <div className="payment-detail">
                  <label>Amount:</label>
                  <span>{calculateJobCost().toFixed(6)} dGPU</span>
                </div>
                <div className="payment-detail">
                  <label>Status:</label>
                  <span className="confirmed">Confirmed on Blockchain</span>
                </div>
              </div>
            </div>
          ) : (
            <div className="payment-status pending">
              <h4>Payment Required</h4>
              <p>Complete payment before submitting the job</p>
              <div className="payment-actions">
                <button 
                  type="button"
                  onClick={processPayment}
                  className="pay-button"
                  disabled={!walletConnected || calculateJobCost() === 0}
                >
                  Pay {calculateJobCost().toFixed(6)} dGPU
                </button>
              </div>
            </div>
          )}
        </section>

        {/* Validation Errors */}
        {validationErrors.length > 0 && (
          <div className="validation-errors">
            <h4>Please fix the following errors:</h4>
            <ul>
              {validationErrors.map((error, index) => (
                <li key={index}>{error}</li>
              ))}
            </ul>
          </div>
        )}

        {/* Submit Button */}
        <section className="form-section">
          <button
            type="submit"
            className={`submit-button ${paymentCompleted ? 'ready' : 'not-ready'}`}
            disabled={isSubmitting || !paymentCompleted || getFilteredGpus().length === 0}
          >
            {isSubmitting ? 'Submitting Job...' : 
             paymentCompleted ? 'Submit GPU Job' : 
             'Payment Required'}
          </button>
        </section>
      </form>

      {/* Wallet Modal */}
      {showWallet && (
        <div className="wallet-modal">
          <div className="wallet-modal-content">
            <button 
              className="close-modal"
              onClick={() => setShowWallet(false)}
            >
              Close
            </button>
            <RealPhantomWallet
              onWalletConnected={handleWalletConnected}
              onWalletDisconnected={handleWalletDisconnected}
              onPaymentComplete={handlePaymentComplete}
              onWalletError={handleWalletError}
            />
          </div>
        </div>
      )}
    </div>
  );
}; 