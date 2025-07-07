import React, { useState, useEffect } from 'react';
// import { invoke } from '@tauri-apps/api/tauri';

interface JobExecutionResultsProps {
  jobId: string;
  onClose: () => void;
}

interface ExecutionResult {
  job_id: string;
  booking_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress_percent: number;
  started_at: string;
  completed_at?: string;
  ssh_connection_info?: {
    hostname: string;
    port: number;
    username: string;
    connection_url: string;
    jupyter_url?: string;
    tensorboard_url?: string;
    monitoring_url?: string;
  };
  container_info?: {
    container_id: string;
    image: string;
    status: string;
    ports: Array<{
      host_port: number;
      container_port: number;
      protocol: string;
    }>;
  };
  resource_allocation: {
    allocated_gpu_memory_mb: number;
    allocated_cpu_cores: number;
    allocated_ram_mb: number;
    allocated_storage_gb: number;
    gpu_utilization_limit: number;
  };
  execution_logs: Array<{
    timestamp: string;
    level: 'INFO' | 'ERROR' | 'WARN' | 'DEBUG';
    message: string;
    source: string;
  }>;
  performance_metrics: {
    gpu_utilization: number;
    memory_usage_mb: number;
    cpu_usage_percent: number;
    network_io_mb: number;
    disk_io_mb: number;
    power_consumption_w?: number;
  };
  proof_of_work?: {
    algorithm: string;
    difficulty: number;
    hash_rate: number;
    verification_hash: string;
    timestamp: string;
    nonce: string;
  };
  blockchain_verification?: {
    transaction_hash: string;
    block_height: number;
    confirmations: number;
    gas_used: number;
    status: 'pending' | 'confirmed' | 'failed';
  };
}

export const JobExecutionResults: React.FC<JobExecutionResultsProps> = ({ jobId, onClose }) => {
  const [results, setResults] = useState<ExecutionResult | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [_refreshInterval, _setRefreshInterval] = useState<NodeJS.Timeout | null>(null);

  const fetchJobResults = async () => {
    try {
      setError(null);
      // Simulate real job execution data
      const mockResults: ExecutionResult = {
        job_id: jobId,
        booking_id: `booking_${jobId}_${Date.now()}`,
        status: Math.random() > 0.3 ? 'running' : 'completed',
        progress_percent: Math.min(100, Math.random() * 100 + 20),
        started_at: new Date(Date.now() - Math.random() * 3600000).toISOString(),
        completed_at: Math.random() > 0.5 ? new Date().toISOString() : undefined,
        ssh_connection_info: {
          hostname: 'gpu-node-47.dante.network',
          port: 22,
          username: `job_user_${jobId.slice(0, 8)}`,
          connection_url: `ssh://job_user_${jobId.slice(0, 8)}@gpu-node-47.dante.network:22`,
          jupyter_url: `https://gpu-node-47.dante.network:8888/lab?token=${jobId}`,
          tensorboard_url: `https://gpu-node-47.dante.network:6006`,
          monitoring_url: `https://monitor.dante.network/d/gpu-jobs?var-job=${jobId}`
        },
        container_info: {
          container_id: `container_${Math.random().toString(36).substr(2, 12)}`,
          image: 'pytorch/pytorch:2.1.0-cuda12.1-devel',
          status: 'running',
          ports: [
            { host_port: 8888, container_port: 8888, protocol: 'tcp' },
            { host_port: 6006, container_port: 6006, protocol: 'tcp' },
            { host_port: 22, container_port: 22, protocol: 'tcp' }
          ]
        },
        resource_allocation: {
          allocated_gpu_memory_mb: 24576,
          allocated_cpu_cores: 8,
          allocated_ram_mb: 32768,
          allocated_storage_gb: 100,
          gpu_utilization_limit: 95
        },
        execution_logs: [
          {
            timestamp: new Date(Date.now() - 300000).toISOString(),
            level: 'INFO',
            message: 'Container initialization completed successfully',
            source: 'docker-engine'
          },
          {
            timestamp: new Date(Date.now() - 240000).toISOString(),
            level: 'INFO',
            message: 'GPU memory allocation: 24576MB reserved',
            source: 'cuda-runtime'
          },
          {
            timestamp: new Date(Date.now() - 180000).toISOString(),
            level: 'INFO',
            message: 'PyTorch GPU backend initialized with CUDA 12.1',
            source: 'pytorch'
          },
          {
            timestamp: new Date(Date.now() - 120000).toISOString(),
            level: 'INFO',
            message: 'Training pipeline started with batch size 64',
            source: 'user-script'
          },
          {
            timestamp: new Date(Date.now() - 60000).toISOString(),
            level: 'INFO',
            message: 'Epoch 1/10 completed - Loss: 0.3247, Accuracy: 87.3%',
            source: 'user-script'
          }
        ],
        performance_metrics: {
          gpu_utilization: Math.floor(Math.random() * 20 + 75),
          memory_usage_mb: Math.floor(Math.random() * 8000 + 16000),
          cpu_usage_percent: Math.floor(Math.random() * 30 + 40),
          network_io_mb: Math.floor(Math.random() * 500 + 100),
          disk_io_mb: Math.floor(Math.random() * 1000 + 200),
          power_consumption_w: Math.floor(Math.random() * 50 + 200)
        },
        proof_of_work: {
          algorithm: 'SHA-256-GPU',
          difficulty: 16777216,
          hash_rate: Math.floor(Math.random() * 1000000 + 5000000),
          verification_hash: `0x${Math.random().toString(16).substr(2, 64)}`,
          timestamp: new Date().toISOString(),
          nonce: Math.random().toString(36).substr(2, 16)
        },
        blockchain_verification: {
          transaction_hash: `0x${Math.random().toString(16).substr(2, 64)}`,
          block_height: Math.floor(Math.random() * 1000000 + 18000000),
          confirmations: Math.floor(Math.random() * 50 + 10),
          gas_used: Math.floor(Math.random() * 100000 + 50000),
          status: 'confirmed'
        }
      };
      setResults(mockResults);
    } catch (err) {
      setError(`Failed to fetch job results: ${err}`);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchJobResults();
    // Auto-refresh every 10 seconds
    const interval = setInterval(fetchJobResults, 10000);
    _setRefreshInterval(interval);
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [jobId]);

  if (isLoading) {
    return (
      <div className="job-results-container">
        <div className="results-header">
          <h2>Job Execution Results</h2>
          <button onClick={onClose} className="close-button">Close</button>
        </div>
        <div className="loading-state">
          <p>Loading execution results...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="job-results-container">
        <div className="results-header">
          <h2>Job Execution Results</h2>
          <button onClick={onClose} className="close-button">Close</button>
        </div>
        <div className="error-state">
          <p>Error: {error}</p>
          <button onClick={fetchJobResults}>Retry</button>
        </div>
      </div>
    );
  }

  if (!results) return null;

  return (
    <div className="job-results-container">
      <div className="results-header">
        <h2>Job Execution Results - {results.job_id}</h2>
        <div className="header-controls">
          <button onClick={fetchJobResults} className="refresh-button">Refresh</button>
          <button onClick={onClose} className="close-button">Close</button>
        </div>
      </div>
      
      <div className="results-content">
        {/* Status Overview */}
        <section className="results-section">
          <h3>Execution Status</h3>
          <div className="status-grid">
            <div className="status-item">
              <label>Status:</label>
              <span className={`status-badge ${results.status}`}>{results.status.toUpperCase()}</span>
            </div>
            <div className="status-item">
              <label>Progress:</label>
              <span>{results.progress_percent.toFixed(1)}%</span>
            </div>
            <div className="status-item">
              <label>Started:</label>
              <span>{new Date(results.started_at).toLocaleString()}</span>
            </div>
            <div className="status-item">
              <label>Booking ID:</label>
              <span className="monospace">{results.booking_id}</span>
            </div>
          </div>
        </section>

        {/* Connection Information */}
        {results.ssh_connection_info && (
          <section className="results-section">
            <h3>Connection Information</h3>
            <div className="connection-grid">
              <div className="connection-item">
                <label>SSH Connection:</label>
                <code>{results.ssh_connection_info.connection_url}</code>
              </div>
              <div className="connection-item">
                <label>Jupyter Notebook:</label>
                <code>{results.ssh_connection_info.jupyter_url}</code>
              </div>
              <div className="connection-item">
                <label>TensorBoard:</label>
                <code>{results.ssh_connection_info.tensorboard_url}</code>
              </div>
              <div className="connection-item">
                <label>Monitoring:</label>
                <code>{results.ssh_connection_info.monitoring_url}</code>
              </div>
            </div>
          </section>
        )}

        {/* Container Information */}
        {results.container_info && (
          <section className="results-section">
            <h3>Container Information</h3>
            <div className="container-details">
              <div className="detail-row">
                <label>Container ID:</label>
                <code>{results.container_info.container_id}</code>
              </div>
              <div className="detail-row">
                <label>Image:</label>
                <code>{results.container_info.image}</code>
              </div>
              <div className="detail-row">
                <label>Status:</label>
                <span className="container-status">{results.container_info.status}</span>
              </div>
              <div className="detail-row">
                <label>Port Mappings:</label>
                <div className="port-mappings">
                  {results.container_info.ports.map((port, index) => (
                    <code key={index}>{port.host_port}:{port.container_port}/{port.protocol}</code>
                  ))}
                </div>
              </div>
            </div>
          </section>
        )}

        {/* Resource Allocation */}
        <section className="results-section">
          <h3>Resource Allocation</h3>
          <div className="resource-grid">
            <div className="resource-item">
              <label>GPU Memory:</label>
              <span>{results.resource_allocation.allocated_gpu_memory_mb} MB</span>
            </div>
            <div className="resource-item">
              <label>CPU Cores:</label>
              <span>{results.resource_allocation.allocated_cpu_cores}</span>
            </div>
            <div className="resource-item">
              <label>RAM:</label>
              <span>{results.resource_allocation.allocated_ram_mb} MB</span>
            </div>
            <div className="resource-item">
              <label>Storage:</label>
              <span>{results.resource_allocation.allocated_storage_gb} GB</span>
            </div>
            <div className="resource-item">
              <label>GPU Utilization Limit:</label>
              <span>{results.resource_allocation.gpu_utilization_limit}%</span>
            </div>
          </div>
        </section>

        {/* Performance Metrics */}
        <section className="results-section">
          <h3>Real-time Performance Metrics</h3>
          <div className="metrics-grid">
            <div className="metric-item">
              <label>GPU Utilization:</label>
              <span>{results.performance_metrics.gpu_utilization}%</span>
            </div>
            <div className="metric-item">
              <label>Memory Usage:</label>
              <span>{results.performance_metrics.memory_usage_mb} MB</span>
            </div>
            <div className="metric-item">
              <label>CPU Usage:</label>
              <span>{results.performance_metrics.cpu_usage_percent}%</span>
            </div>
            <div className="metric-item">
              <label>Network I/O:</label>
              <span>{results.performance_metrics.network_io_mb} MB</span>
            </div>
            <div className="metric-item">
              <label>Disk I/O:</label>
              <span>{results.performance_metrics.disk_io_mb} MB</span>
            </div>
            {results.performance_metrics.power_consumption_w && (
              <div className="metric-item">
                <label>Power Consumption:</label>
                <span>{results.performance_metrics.power_consumption_w} W</span>
              </div>
            )}
          </div>
        </section>

        {/* Proof of Work */}
        {results.proof_of_work && (
          <section className="results-section">
            <h3>Cryptographic Proof of Work</h3>
            <div className="proof-details">
              <div className="detail-row">
                <label>Algorithm:</label>
                <code>{results.proof_of_work.algorithm}</code>
              </div>
              <div className="detail-row">
                <label>Difficulty:</label>
                <span>{results.proof_of_work.difficulty.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Hash Rate:</label>
                <span>{(results.proof_of_work.hash_rate / 1000000).toFixed(2)} MH/s</span>
              </div>
              <div className="detail-row">
                <label>Verification Hash:</label>
                <code className="hash">{results.proof_of_work.verification_hash}</code>
              </div>
              <div className="detail-row">
                <label>Nonce:</label>
                <code>{results.proof_of_work.nonce}</code>
              </div>
              <div className="detail-row">
                <label>Timestamp:</label>
                <span>{new Date(results.proof_of_work.timestamp).toLocaleString()}</span>
              </div>
            </div>
          </section>
        )}

        {/* Blockchain Verification */}
        {results.blockchain_verification && (
          <section className="results-section">
            <h3>Blockchain Verification</h3>
            <div className="blockchain-details">
              <div className="detail-row">
                <label>Transaction Hash:</label>
                <code className="hash">{results.blockchain_verification.transaction_hash}</code>
              </div>
              <div className="detail-row">
                <label>Block Height:</label>
                <span>{results.blockchain_verification.block_height.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Confirmations:</label>
                <span>{results.blockchain_verification.confirmations}</span>
              </div>
              <div className="detail-row">
                <label>Gas Used:</label>
                <span>{results.blockchain_verification.gas_used.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Status:</label>
                <span className={`verification-status ${results.blockchain_verification.status}`}>
                  {results.blockchain_verification.status.toUpperCase()}
                </span>
              </div>
            </div>
          </section>
        )}

        {/* Execution Logs */}
        <section className="results-section">
          <h3>Execution Logs</h3>
          <div className="logs-container">
            {results.execution_logs.map((log, index) => (
              <div key={index} className={`log-entry ${log.level.toLowerCase()}`}>
                <span className="log-timestamp">{new Date(log.timestamp).toLocaleTimeString()}</span>
                <span className="log-level">[{log.level}]</span>
                <span className="log-source">{log.source}:</span>
                <span className="log-message">{log.message}</span>
              </div>
            ))}
          </div>
        </section>
      </div>

      <style dangerouslySetInnerHTML={{
        __html: `
        .job-results-container {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: rgba(0, 0, 0, 0.8);
          display: flex;
          flex-direction: column;
          z-index: 1000;
        }
        
        .results-header {
          background: #1a1a1a;
          color: white;
          padding: 20px;
          display: flex;
          justify-content: space-between;
          align-items: center;
          border-bottom: 1px solid #333;
        }
        
        .results-header h2 {
          margin: 0;
          font-family: Monaco, Menlo, monospace;
          font-size: 18px;
        }
        
        .header-controls {
          display: flex;
          gap: 10px;
        }
        
        .refresh-button, .close-button {
          background: #333;
          color: white;
          border: 1px solid #555;
          padding: 8px 16px;
          border-radius: 4px;
          cursor: pointer;
          font-family: monospace;
        }
        
        .refresh-button:hover, .close-button:hover {
          background: #444;
        }
        
        .results-content {
          background: #f5f5f5;
          flex: 1;
          overflow-y: auto;
          padding: 20px;
        }
        
        .results-section {
          background: white;
          border: 1px solid #ddd;
          border-radius: 4px;
          margin-bottom: 20px;
          padding: 20px;
        }
        
        .results-section h3 {
          margin: 0 0 15px 0;
          font-family: monospace;
          font-size: 16px;
          color: #333;
          border-bottom: 1px solid #eee;
          padding-bottom: 10px;
        }
        
        .status-grid, .connection-grid, .resource-grid, .metrics-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
          gap: 15px;
        }
        
        .status-item, .connection-item, .resource-item, .metric-item {
          display: flex;
          flex-direction: column;
          gap: 5px;
        }
        
        .status-item label, .connection-item label, .resource-item label, .metric-item label {
          font-weight: bold;
          font-size: 12px;
          color: #666;
          text-transform: uppercase;
        }
        
        .status-badge {
          padding: 4px 8px;
          border-radius: 4px;
          font-family: monospace;
          font-size: 12px;
          font-weight: bold;
          text-align: center;
        }
        
        .status-badge.running {
          background: #ffc107;
          color: #000;
        }
        
        .status-badge.completed {
          background: #28a745;
          color: white;
        }
        
        .status-badge.failed {
          background: #dc3545;
          color: white;
        }
        
        .status-badge.pending {
          background: #6c757d;
          color: white;
        }
        
        .monospace, code {
          font-family: Monaco, Menlo, monospace;
          background: #f8f9fa;
          padding: 2px 4px;
          border-radius: 3px;
          font-size: 12px;
          word-break: break-all;
        }
        
        .hash {
          font-size: 10px;
          word-break: break-all;
        }
        
        .container-details, .proof-details, .blockchain-details {
          display: flex;
          flex-direction: column;
          gap: 10px;
        }
        
        .detail-row {
          display: flex;
          align-items: center;
          gap: 10px;
          min-height: 24px;
        }
        
        .detail-row label {
          font-weight: bold;
          min-width: 150px;
          font-size: 12px;
          color: #666;
        }
        
        .port-mappings {
          display: flex;
          gap: 10px;
          flex-wrap: wrap;
        }
        
        .logs-container {
          background: #1a1a1a;
          border-radius: 4px;
          padding: 15px;
          max-height: 400px;
          overflow-y: auto;
          font-family: monospace;
        }
        
        .log-entry {
          display: flex;
          gap: 10px;
          padding: 4px 0;
          font-size: 12px;
          line-height: 1.4;
        }
        
        .log-timestamp {
          color: #666;
          min-width: 80px;
        }
        
        .log-level {
          min-width: 60px;
          font-weight: bold;
        }
        
        .log-entry.info .log-level {
          color: #28a745;
        }
        
        .log-entry.error .log-level {
          color: #dc3545;
        }
        
        .log-entry.warn .log-level {
          color: #ffc107;
        }
        
        .log-entry.debug .log-level {
          color: #6c757d;
        }
        
        .log-source {
          color: #007bff;
          min-width: 120px;
        }
        
        .log-message {
          color: #fff;
          flex: 1;
        }
        
        .verification-status.confirmed {
          color: #28a745;
          font-weight: bold;
        }
        
        .verification-status.pending {
          color: #ffc107;
          font-weight: bold;
        }
        
        .verification-status.failed {
          color: #dc3545;
          font-weight: bold;
        }
        
        .loading-state, .error-state {
          text-align: center;
          padding: 40px;
          color: white;
        }
        
        .error-state button {
          background: #dc3545;
          color: white;
          border: none;
          padding: 10px 20px;
          border-radius: 4px;
          cursor: pointer;
          margin-top: 15px;
        }
        `
      }} />
    </div>
  );
}; 