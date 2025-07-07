import React, { useState, useEffect } from 'react';
import { invoke } from '@tauri-apps/api/tauri';
import './JobSubmission.css';
import './JobSubmission.css';

interface JobSubmissionProps {
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
  duration: number; // in hours
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
  pricing: {
    maxHourlyRate: number;
    currency: 'USD' | 'DGPU' | 'SOL';
    totalBudget: number;
  };
}

interface JobTemplate {
  id: string;
  name: string;
  description: string;
  jobType: JobSubmissionForm['jobType'];
  defaultConfig: Partial<JobSubmissionForm>;
  requiredGpuSpecs: {
    minVramMb: number;
    preferredModels: string[];
  };
}

const jobTemplates: JobTemplate[] = [
  {
    id: 'ml-training',
    name: 'Machine Learning Training',
    description: 'Train neural networks with PyTorch or TensorFlow',
    jobType: 'machine_learning',
    defaultConfig: {
      jobConfig: {
        dockerImage: 'pytorch/pytorch:latest',
        environmentVariables: { 'CUDA_VISIBLE_DEVICES': '0' },
        command: 'python',
        args: ['train.py'],
        inputFiles: ['dataset.zip', 'model.py', 'train.py'],
        expectedOutputs: ['model.pth', 'training_log.txt']
      },
      resources: {
        gpuMemoryMb: 8000,
        cpuCores: 4,
        ramMb: 16000,
        storageMb: 50000
      }
    },
    requiredGpuSpecs: {
      minVramMb: 6000,
      preferredModels: ['RTX 4090', 'RTX 4080', 'RTX 3090']
    }
  },
  {
    id: 'crypto-mining',
    name: 'Cryptocurrency Mining',
    description: 'Mine cryptocurrencies with optimized miners',
    jobType: 'cryptocurrency_mining',
    defaultConfig: {
      jobConfig: {
        dockerImage: 'crypto-miner:latest',
        environmentVariables: { 'POOL_URL': 'stratum+tcp://pool.example.com:4444' },
        command: 'miner',
        args: ['--algo', 'ethash', '--pool', '${POOL_URL}'],
        inputFiles: ['miner.conf'],
        expectedOutputs: ['mining_stats.json', 'shares.log']
      },
      resources: {
        gpuMemoryMb: 4000,
        cpuCores: 2,
        ramMb: 8000,
        storageMb: 10000
      }
    },
    requiredGpuSpecs: {
      minVramMb: 4000,
      preferredModels: ['RTX 4070', 'RTX 3080', 'RTX 3070']
    }
  },
  {
    id: 'scientific-compute',
    name: 'Scientific Computing',
    description: 'Run CUDA-accelerated scientific simulations',
    jobType: 'scientific_computing',
    defaultConfig: {
      jobConfig: {
        dockerImage: 'nvidia/cuda:11.8-runtime-ubuntu20.04',
        environmentVariables: { 'CUDA_VISIBLE_DEVICES': '0' },
        command: 'nvidia-smi',
        args: ['--query-gpu=memory.used,memory.total', '--format=csv'],
        inputFiles: ['simulation_data.csv', 'compute_script.py'],
        expectedOutputs: ['results.csv', 'performance_metrics.json']
      },
      resources: {
        gpuMemoryMb: 12000,
        cpuCores: 8,
        ramMb: 32000,
        storageMb: 100000
      }
    },
    requiredGpuSpecs: {
      minVramMb: 8000,
      preferredModels: ['RTX 4090', 'RTX 3090', 'A6000']
    }
  },
  {
    id: 'rendering',
    name: '3D Rendering',
    description: 'GPU-accelerated 3D rendering with Blender or similar',
    jobType: 'rendering',
    defaultConfig: {
      jobConfig: {
        dockerImage: 'blender:latest',
        environmentVariables: { 'BLENDER_GPU': 'CUDA' },
        command: 'blender',
        args: ['-b', 'scene.blend', '-o', 'output_', '-f', '1'],
        inputFiles: ['scene.blend', 'textures.zip'],
        expectedOutputs: ['output_0001.png', 'render_stats.json']
      },
      resources: {
        gpuMemoryMb: 6000,
        cpuCores: 4,
        ramMb: 16000,
        storageMb: 25000
      }
    },
    requiredGpuSpecs: {
      minVramMb: 4000,
      preferredModels: ['RTX 4080', 'RTX 3080', 'RTX 3070']
    }
  }
];

export const JobSubmission: React.FC<JobSubmissionProps> = ({ onJobSubmitted, availableGpus }) => {
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
    },
    pricing: {
      maxHourlyRate: 5.0,
      currency: 'USD',
      totalBudget: 0
    }
  });

  const [selectedTemplate, setSelectedTemplate] = useState<JobTemplate | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [_estimatedCost, setEstimatedCost] = useState<number>(0);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);

  // Calculate estimated cost when form changes
  useEffect(() => {
    const selectedGpu = availableGpus.find(gpu => gpu.id === form.gpuId);
    if (selectedGpu && selectedGpu.current_hourly_rate_dgpu) {
      const rate = selectedGpu.current_hourly_rate_dgpu;
      const cost = rate * form.duration;
      setEstimatedCost(cost);
      setForm(prev => ({ ...prev, pricing: { ...prev.pricing, totalBudget: cost } }));
    }
  }, [form.gpuId, form.duration, availableGpus]);

  const handleTemplateSelect = (template: JobTemplate) => {
    setSelectedTemplate(template);
    setForm(prev => ({
      ...prev,
      jobType: template.jobType,
      ...template.defaultConfig,
      jobName: template.name,
      resources: { ...prev.resources, ...template.defaultConfig.resources },
      jobConfig: { ...prev.jobConfig, ...template.defaultConfig.jobConfig }
    }));
  };

  const getFrameworkFromJobType = (jobType: JobSubmissionForm['jobType']): string => {
    switch (jobType) {
      case 'machine_learning':
        return 'pytorch';
      case 'scientific_computing':
        return 'tensorflow';
      case 'cryptocurrency_mining':
        return 'custom';
      case 'rendering':
        return 'custom';
      default:
        return 'custom';
    }
  };

  const validateForm = (): boolean => {
    const errors: string[] = [];

    if (!form.jobName.trim()) errors.push('Job name is required');
    if (!form.gpuId) errors.push('GPU selection is required');
    if (form.duration <= 0) errors.push('Duration must be positive');
    if (!form.jobConfig.dockerImage.trim()) errors.push('Docker image is required');
    if (!form.jobConfig.command.trim()) errors.push('Command is required');

    const selectedGpu = availableGpus.find(gpu => gpu.id === form.gpuId);
    if (selectedGpu) {
      if (form.resources.gpuMemoryMb > selectedGpu.vram_free_mb) {
        errors.push(`Requested GPU memory (${form.resources.gpuMemoryMb}MB) exceeds available (${selectedGpu.vram_free_mb}MB)`);
      }
    }

    setValidationErrors(errors);
    return errors.length === 0;
  };

    const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [paymentCompleted, setPaymentCompleted] = useState(false);
  const [paymentTransactionHash, setPaymentTransactionHash] = useState<string | null>(null);

  const calculateJobCost = (): number => {
    const selectedGpu = availableGpus.find(gpu => gpu.id === form.gpuId);
    if (!selectedGpu || !selectedGpu.current_hourly_rate_dgpu) return 0;
    return selectedGpu.current_hourly_rate_dgpu * form.duration;
  };

  const handlePaymentComplete = (transactionHash: string) => {
    setPaymentCompleted(true);
    setPaymentTransactionHash(transactionHash);
    setShowPaymentModal(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) return;

    // First, require payment
    if (!paymentCompleted) {
      setShowPaymentModal(true);
      return;
    }

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
        estimated_completion_time: Math.floor(form.duration * 60), // Convert hours to minutes
        gpu_memory_requirements: form.resources.gpuMemoryMb,
        cpu_requirements: form.resources.cpuCores,
        ram_requirements: form.resources.ramMb,
        storage_requirements: form.resources.storageMb,
        network_requirements: true,
        checkpoint_frequency: 300, // 5 minutes
        auto_restart_on_failure: true,
        max_retries: 3,
        priority: "normal"
      };

      const jobResult = await invoke<string>('submit_gpu_job', {
        jobSpecification,
        gpuId: form.gpuId,
        paymentMethod: 'dgpu_tokens',
        durationHours: form.duration,
        renterId: 'user_' + Date.now(),
        renterName: 'GPU Rental User'
      });

      const jobData = JSON.parse(jobResult);
      onJobSubmitted(jobData.job_id || jobData.booking_id);

      // Reset form and payment state
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
        },
        pricing: {
          maxHourlyRate: 5.0,
          currency: 'DGPU',
          totalBudget: 0
        }
      });
      
      setSelectedTemplate(null);
      setPaymentCompleted(false);
      setPaymentTransactionHash(null);
      
    } catch (error) {
      console.error('Job submission failed:', error);
      setValidationErrors([`Job submission failed: ${error}`]);
    } finally {
      setIsSubmitting(false);
    }
  };

  const getFilteredGpus = () => {
    return availableGpus.filter(gpu => {
      if (!gpu.is_available_for_rent) return false;
      
      if (selectedTemplate) {
        const meetsSpecs = gpu.vram_total_mb >= selectedTemplate.requiredGpuSpecs.minVramMb;
        return meetsSpecs;
      }
      
      return true;
    });
  };

  return (
    <div className="job-submission-container">
      <h2>Submit GPU Job</h2>
      
      {/* Job Templates */}
      <div className="job-templates">
        <h3>Quick Start Templates</h3>
        <div className="template-grid">
          {jobTemplates.map(template => (
            <div
              key={template.id}
              className={`template-card ${selectedTemplate?.id === template.id ? 'selected' : ''}`}
              onClick={() => handleTemplateSelect(template)}
            >
              <h4>{template.name}</h4>
              <p>{template.description}</p>
              <div className="template-requirements">
                <small>Min VRAM: {template.requiredGpuSpecs.minVramMb}MB</small>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Job Submission Form */}
      <form onSubmit={handleSubmit} className="job-form">
        {/* Basic Job Information */}
        <div className="form-section">
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
              onChange={(e) => setForm(prev => ({ ...prev, jobType: e.target.value as JobSubmissionForm['jobType'] }))}
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
        </div>

        {/* GPU Selection */}
        <div className="form-section">
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
                  <p>Rate: {gpu.current_hourly_rate_dgpu} DGPU/hour</p>
                  {gpu.utilization_gpu_percent !== undefined && (
                    <p>Utilization: {gpu.utilization_gpu_percent}%</p>
                  )}
                </div>
                <div className="gpu-status">
                  {form.gpuId === gpu.id && <span className="selected-indicator">✓</span>}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Resource Allocation */}
        <div className="form-section">
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
        </div>

        {/* Job Configuration */}
        <div className="form-section">
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
        </div>

        {/* Cost Calculation and Payment */}
        <div className="form-section">
          <h3>Cost Calculation & Payment</h3>
          
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
                  <span>{calculateJobCost().toFixed(2)} dGPU</span>
                </div>
              </div>
            </div>
          )}

          {/* Payment Status */}
          {paymentCompleted ? (
            <div className="payment-status completed">
              <h4>✅ Payment Completed</h4>
              <p>Transaction Hash: <code>{paymentTransactionHash}</code></p>
              <p>Amount: {calculateJobCost().toFixed(2)} dGPU</p>
            </div>
          ) : (
            <div className="payment-status pending">
              <h4>⏳ Payment Required</h4>
              <p>You must complete payment before submitting the job.</p>
              <p>Cost: {calculateJobCost().toFixed(2)} dGPU</p>
            </div>
          )}
        </div>

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
        <div className="form-actions">
          <button
            type="submit"
            disabled={isSubmitting || getFilteredGpus().length === 0}
            className={`submit-button ${paymentCompleted ? 'payment-completed' : 'payment-required'}`}
          >
            {isSubmitting ? 'Submitting Job...' : 
             paymentCompleted ? 'Submit GPU Job' : 
             'Pay & Submit GPU Job'}
          </button>
        </div>

        {/* Payment Modal */}
        {showPaymentModal && (
          <div className="payment-modal-backdrop">
            <div className="payment-modal">
              <h3>Complete Payment</h3>
              <div className="payment-details">
                <p><strong>Job:</strong> {form.jobName}</p>
                <p><strong>GPU:</strong> {availableGpus.find(gpu => gpu.id === form.gpuId)?.name}</p>
                <p><strong>Duration:</strong> {form.duration} hours</p>
                <p><strong>Total Cost:</strong> {calculateJobCost().toFixed(2)} dGPU</p>
              </div>
              <div className="payment-actions">
                <button 
                  onClick={async () => {
                    try {
                      const response = await invoke<string>('process_dgpu_payment', {
                        amountDgpu: calculateJobCost(),
                        recipientAddress: '7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump'
                      });
                      const paymentData = JSON.parse(response);
                      handlePaymentComplete(paymentData.transaction_hash);
                    } catch (error) {
                      console.error('Payment failed:', error);
                      setValidationErrors([`Payment failed: ${error}`]);
                    }
                  }}
                  className="pay-button"
                >
                  Pay {calculateJobCost().toFixed(2)} dGPU
                </button>
                <button 
                  onClick={() => setShowPaymentModal(false)}
                  className="cancel-button"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        )}
      </form>

      
    </div>
  );
}; 