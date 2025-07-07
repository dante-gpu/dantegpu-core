import React, { useState, useEffect } from 'react';
import { invoke } from '@tauri-apps/api/tauri';

interface ProofOfWorkProps {
  jobId: string;
  gpuId: string;
  onProofGenerated: (proof: WorkProof) => void;
  onVerificationComplete: (isValid: boolean) => void;
}

interface WorkProof {
  jobId: string;
  gpuId: string;
  computeHash: string;
  performanceMetrics: PerformanceMetrics;
  timestamp: string;
  nonce: number;
  difficulty: number;
  signature: string;
  benchmarkResults: BenchmarkResult[];
}

interface PerformanceMetrics {
  executionTimeMs: number;
  gpuUtilizationPercent: number;
  memoryUsageMb: number;
  powerConsumptionW: number;
  temperatureC: number;
  computeOperationsPerSecond: number;
  hashRate: number;
}

interface BenchmarkResult {
  testName: string;
  expectedResult: string;
  actualResult: string;
  isValid: boolean;
  executionTime: number;
  difficulty: number;
}

interface VerificationStatus {
  status: 'generating' | 'verifying' | 'completed' | 'failed';
  progress: number;
  message: string;
  proof?: WorkProof;
}

// Predefined benchmark tests for different GPU workload types
const benchmarkTests = {
  machine_learning: [
    {
      name: 'Matrix Multiplication',
      description: 'Perform large matrix multiplication',
      expectedHash: '0x7a2b9c8f...',
      difficulty: 4
    },
    {
      name: 'Neural Network Forward Pass',
      description: 'Execute CNN forward pass',
      expectedHash: '0x4f8c2a9b...',
      difficulty: 5
    }
  ],
  cryptocurrency_mining: [
    {
      name: 'SHA-256 Hashing',
      description: 'Compute SHA-256 hashes',
      expectedHash: '0x000012ab...',
      difficulty: 6
    },
    {
      name: 'Scrypt Algorithm',
      description: 'Execute Scrypt computation',
      expectedHash: '0x0000a4c2...',
      difficulty: 7
    }
  ],
  scientific_computing: [
    {
      name: 'FFT Computation',
      description: 'Fast Fourier Transform',
      expectedHash: '0x9f2e8c1a...',
      difficulty: 5
    },
    {
      name: 'Monte Carlo Simulation',
      description: 'Random number generation and analysis',
      expectedHash: '0x3c4e7b9f...',
      difficulty: 6
    }
  ],
  rendering: [
    {
      name: 'Ray Tracing',
      description: 'Basic ray tracing computation',
      expectedHash: '0x8e4c9a7b...',
      difficulty: 5
    },
    {
      name: 'Shader Compilation',
      description: 'Compile and execute GPU shaders',
      expectedHash: '0x2f8a9c4e...',
      difficulty: 4
    }
  ]
};

export const ProofOfWork: React.FC<ProofOfWorkProps> = ({ jobId, gpuId, onProofGenerated, onVerificationComplete }) => {
  const [verificationStatus, setVerificationStatus] = useState<VerificationStatus>({
    status: 'generating',
    progress: 0,
    message: 'Initializing proof of work...'
  });
  const [currentBenchmark, setCurrentBenchmark] = useState<string>('');
  const [benchmarkResults, setBenchmarkResults] = useState<BenchmarkResult[]>([]);
  const [performanceData, setPerformanceData] = useState<PerformanceMetrics | null>(null);
  const [isGeneratingProof, setIsGeneratingProof] = useState(false);

  // Start proof of work generation
  const generateProofOfWork = async (jobType: string) => {
    setIsGeneratingProof(true);
    setVerificationStatus({
      status: 'generating',
      progress: 0,
      message: 'Starting proof of work generation...'
    });

    try {
      // Get appropriate benchmark tests for job type
      const tests = benchmarkTests[jobType as keyof typeof benchmarkTests] || benchmarkTests.machine_learning;

      // Execute each benchmark test
      const results: BenchmarkResult[] = [];
      for (let i = 0; i < tests.length; i++) {
        const test = tests[i];
        setCurrentBenchmark(test.name);
        setVerificationStatus(prev => ({
          ...prev,
          progress: (i / tests.length) * 70,
          message: `Running benchmark: ${test.name}`
        }));

        // Execute benchmark on GPU
        const result = await executeBenchmark(test, gpuId);
        results.push(result);
        setBenchmarkResults(prev => [...prev, result]);

        // Small delay to show progress
        await new Promise(resolve => setTimeout(resolve, 1000));
      }

      // Collect performance metrics
      setVerificationStatus(prev => ({
        ...prev,
        progress: 80,
        message: 'Collecting performance metrics...'
      }));
      const metrics = await collectPerformanceMetrics(gpuId, jobId);
      setPerformanceData(metrics);

      // Generate cryptographic proof
      setVerificationStatus(prev => ({
        ...prev,
        progress: 90,
        message: 'Generating cryptographic proof...'
      }));
      const proof = await generateCryptographicProof(
        jobId,
        gpuId,
        results,
        metrics
      );

      // Complete proof generation
      setVerificationStatus({
        status: 'completed',
        progress: 100,
        message: 'Proof of work generated successfully',
        proof
      });

      onProofGenerated(proof);
    } catch (error) {
      console.error('Failed to generate proof of work:', error);
      setVerificationStatus({
        status: 'failed',
        progress: 0,
        message: `Failed to generate proof: ${error}`
      });
    } finally {
      setIsGeneratingProof(false);
    }
  };

  // Execute individual benchmark test
  const executeBenchmark = async (test: any, gpuId: string): Promise<BenchmarkResult> => {
    try {
      const startTime = Date.now();
      
      // Call Tauri backend to execute GPU benchmark
      const result = await invoke<string>('execute_gpu_benchmark', {
        gpuId,
        testName: test.name,
        difficulty: test.difficulty
      });
      
      const executionTime = Date.now() - startTime;
      
      // Verify result against expected hash
      const isValid = await verifyBenchmarkResult(result, test.expectedHash);
      
      return {
        testName: test.name,
        expectedResult: test.expectedHash,
        actualResult: result,
        isValid,
        executionTime,
        difficulty: test.difficulty
      };
    } catch (error) {
      console.error(`Benchmark ${test.name} failed:`, error);
      return {
        testName: test.name,
        expectedResult: test.expectedHash,
        actualResult: '',
        isValid: false,
        executionTime: 0,
        difficulty: test.difficulty
      };
    }
  };

  // Verify benchmark result
  const verifyBenchmarkResult = async (result: string, expectedHash: string): Promise<boolean> => {
    try {
      const isValid = await invoke<boolean>('verify_benchmark_result', {
        result,
        expectedHash
      });
      return isValid;
    } catch (error) {
      console.error('Failed to verify benchmark result:', error);
      return false;
    }
  };

  // Collect performance metrics from GPU
  const collectPerformanceMetrics = async (gpuId: string, jobId: string): Promise<PerformanceMetrics> => {
    try {
      const metrics = await invoke<PerformanceMetrics>('collect_gpu_performance_metrics', {
        gpuId,
        jobId
      });
      return metrics;
    } catch (error) {
      console.error('Failed to collect performance metrics:', error);
      return {
        executionTimeMs: 0,
        gpuUtilizationPercent: 0,
        memoryUsageMb: 0,
        powerConsumptionW: 0,
        temperatureC: 0,
        computeOperationsPerSecond: 0,
        hashRate: 0
      };
    }
  };

  // Generate cryptographic proof
  const generateCryptographicProof = async (
    jobId: string,
    gpuId: string,
    benchmarkResults: BenchmarkResult[],
    metrics: PerformanceMetrics
  ): Promise<WorkProof> => {
    try {
      const proof = await invoke<WorkProof>('generate_cryptographic_proof', {
        jobId,
        gpuId,
        benchmarkResults,
        metrics,
        timestamp: new Date().toISOString()
      });
      return proof;
    } catch (error) {
      console.error('Failed to generate cryptographic proof:', error);
      throw error;
    }
  };

  // Verify existing proof
  const verifyProof = async (proof: WorkProof) => {
    setVerificationStatus({
      status: 'verifying',
      progress: 0,
      message: 'Verifying proof of work...'
    });

    try {
      // Verify cryptographic signature
      setVerificationStatus(prev => ({
        ...prev,
        progress: 30,
        message: 'Verifying cryptographic signature...'
      }));
      const signatureValid = await invoke<boolean>('verify_cryptographic_signature', { proof });
      if (!signatureValid) {
        throw new Error('Invalid cryptographic signature');
      }

      // Verify benchmark results
      setVerificationStatus(prev => ({
        ...prev,
        progress: 60,
        message: 'Verifying benchmark results...'
      }));
      const benchmarkValid = await invoke<boolean>('verify_benchmark_results', {
        results: proof.benchmarkResults
      });
      if (!benchmarkValid) {
        throw new Error('Invalid benchmark results');
      }

      // Verify performance metrics
      setVerificationStatus(prev => ({
        ...prev,
        progress: 90,
        message: 'Verifying performance metrics...'
      }));
      const metricsValid = await invoke<boolean>('verify_performance_metrics', {
        metrics: proof.performanceMetrics,
        gpuId: proof.gpuId
      });
      if (!metricsValid) {
        throw new Error('Invalid performance metrics');
      }

      // Verification successful
      setVerificationStatus({
        status: 'completed',
        progress: 100,
        message: 'Proof verification completed successfully'
      });
      onVerificationComplete(true);
    } catch (error) {
      console.error('Proof verification failed:', error);
      setVerificationStatus({
        status: 'failed',
        progress: 0,
        message: `Verification failed: ${error}`
      });
      onVerificationComplete(false);
    }
  };

  // Auto-start proof generation
  useEffect(() => {
    // Get job type from jobId or default to machine_learning
    const jobType = 'machine_learning'; // This should come from job data
    generateProofOfWork(jobType);
  }, [jobId, gpuId]);

  const getStatusColor = () => {
    switch (verificationStatus.status) {
      case 'generating': return '#007bff';
      case 'verifying': return '#ffc107';
      case 'completed': return '#28a745';
      case 'failed': return '#dc3545';
      default: return '#6c757d';
    }
  };

  const getStatusIcon = () => {
    switch (verificationStatus.status) {
      case 'generating': return '⚙️';
      case 'verifying': return '🔍';
      case 'completed': return '✅';
      case 'failed': return '❌';
      default: return '⏳';
    }
  };

  return (
    <div className="proof-of-work-container">
      <div className="proof-header">
        <h3>Proof of Work Verification</h3>
        <div className="job-info">
          <span>Job ID: {jobId}</span>
          <span>GPU ID: {gpuId}</span>
        </div>
      </div>

      {/* Status Display */}
      <div className="verification-status">
        <div className="status-indicator" style={{ backgroundColor: getStatusColor() }}>
          <span className="status-icon">{getStatusIcon()}</span>
          <span className="status-text">{verificationStatus.status.toUpperCase()}</span>
        </div>
        <div className="progress-bar">
          <div
            className="progress-fill"
            style={{
              width: `${verificationStatus.progress}%`,
              backgroundColor: getStatusColor()
            }}
          />
        </div>
        <div className="status-message">
          {verificationStatus.message}
        </div>
      </div>

      {/* Current Benchmark */}
      {currentBenchmark && (
        <div className="current-benchmark">
          <h4>Current Benchmark: {currentBenchmark}</h4>
        </div>
      )}

      {/* Benchmark Results */}
      {benchmarkResults.length > 0 && (
        <div className="benchmark-results">
          <h4>Benchmark Results</h4>
          <div className="results-list">
            {benchmarkResults.map((result, index) => (
              <div key={index} className="result-item">
                <div className="result-header">
                  <span className="test-name">{result.testName}</span>
                  <span className={`result-status ${result.isValid ? 'valid' : 'invalid'}`}>
                    {result.isValid ? '✅ Valid' : '❌ Invalid'}
                  </span>
                </div>
                <div className="result-details">
                  <div>Execution Time: {result.executionTime}ms</div>
                  <div>Difficulty: {result.difficulty}</div>
                  <div className="result-hash">
                    Result: {result.actualResult.slice(0, 16)}...
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Performance Metrics */}
      {performanceData && (
        <div className="performance-metrics">
          <h4>Performance Metrics</h4>
          <div className="metrics-grid">
            <div className="metric-item">
              <span className="metric-label">Execution Time:</span>
              <span className="metric-value">{performanceData.executionTimeMs}ms</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">GPU Utilization:</span>
              <span className="metric-value">{performanceData.gpuUtilizationPercent}%</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Memory Usage:</span>
              <span className="metric-value">{performanceData.memoryUsageMb}MB</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Power Consumption:</span>
              <span className="metric-value">{performanceData.powerConsumptionW}W</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Temperature:</span>
              <span className="metric-value">{performanceData.temperatureC}°C</span>
            </div>
            <div className="metric-item">
              <span className="metric-label">Hash Rate:</span>
              <span className="metric-value">{performanceData.hashRate.toFixed(2)} H/s</span>
            </div>
          </div>
        </div>
      )}

      {/* Proof Details */}
      {verificationStatus.proof && (
        <div className="proof-details">
          <h4>Cryptographic Proof</h4>
          <div className="proof-info">
            <div className="proof-item">
              <span className="proof-label">Compute Hash:</span>
              <span className="proof-value">{verificationStatus.proof.computeHash}</span>
            </div>
            <div className="proof-item">
              <span className="proof-label">Nonce:</span>
              <span className="proof-value">{verificationStatus.proof.nonce}</span>
            </div>
            <div className="proof-item">
              <span className="proof-label">Difficulty:</span>
              <span className="proof-value">{verificationStatus.proof.difficulty}</span>
            </div>
            <div className="proof-item">
              <span className="proof-label">Signature:</span>
              <span className="proof-value">{verificationStatus.proof.signature.slice(0, 32)}...</span>
            </div>
            <div className="proof-item">
              <span className="proof-label">Timestamp:</span>
              <span className="proof-value">{new Date(verificationStatus.proof.timestamp).toLocaleString()}</span>
            </div>
          </div>
        </div>
      )}

      {/* Action Buttons */}
      <div className="proof-actions">
        {verificationStatus.status === 'failed' && (
          <button
            onClick={() => generateProofOfWork('machine_learning')}
            disabled={isGeneratingProof}
            className="retry-button"
          >
            Retry Proof Generation
          </button>
        )}
        {verificationStatus.proof && (
          <button
            onClick={() => verifyProof(verificationStatus.proof!)}
            className="verify-button"
          >
            Re-verify Proof
          </button>
        )}
      </div>

      <style dangerouslySetInnerHTML={{
        __html: `
        .proof-of-work-container {
          background: #f8f9fa;
          border-radius: 8px;
          padding: 20px;
          margin: 20px 0;
        }
        
        .proof-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 20px;
        }
        
        .proof-header h3 {
          margin: 0;
          color: #333;
        }
        
        .job-info {
          display: flex;
          flex-direction: column;
          font-size: 12px;
          color: #666;
        }
        
        .verification-status {
          background: white;
          border-radius: 6px;
          padding: 15px;
          margin-bottom: 20px;
        }
        
        .status-indicator {
          display: flex;
          align-items: center;
          gap: 10px;
          color: white;
          padding: 8px 12px;
          border-radius: 4px;
          margin-bottom: 10px;
        }
        
        .status-icon {
          font-size: 16px;
        }
        
        .status-text {
          font-weight: bold;
        }
        
        .progress-bar {
          background: #e9ecef;
          border-radius: 10px;
          height: 8px;
          margin-bottom: 10px;
          overflow: hidden;
        }
        
        .progress-fill {
          height: 100%;
          transition: width 0.3s ease;
          border-radius: 10px;
        }
        
        .status-message {
          color: #666;
          font-size: 14px;
        }
        
        .current-benchmark {
          background: #e3f2fd;
          border: 1px solid #2196f3;
          border-radius: 6px;
          padding: 10px;
          margin-bottom: 20px;
        }
        
        .current-benchmark h4 {
          margin: 0;
          color: #1976d2;
        }
        
        .benchmark-results {
          background: white;
          border-radius: 6px;
          padding: 15px;
          margin-bottom: 20px;
        }
        
        .benchmark-results h4 {
          margin: 0 0 15px 0;
          color: #333;
        }
        
        .results-list {
          display: flex;
          flex-direction: column;
          gap: 10px;
        }
        
        .result-item {
          border: 1px solid #dee2e6;
          border-radius: 4px;
          padding: 10px;
        }
        
        .result-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 8px;
        }
        
        .test-name {
          font-weight: bold;
          color: #333;
        }
        
        .result-status.valid {
          color: #28a745;
        }
        
        .result-status.invalid {
          color: #dc3545;
        }
        
        .result-details {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
          gap: 8px;
          font-size: 12px;
          color: #666;
        }
        
        .result-hash {
          font-family: monospace;
          background: #f8f9fa;
          padding: 2px 4px;
          border-radius: 2px;
        }
        
        .performance-metrics {
          background: white;
          border-radius: 6px;
          padding: 15px;
          margin-bottom: 20px;
        }
        
        .performance-metrics h4 {
          margin: 0 0 15px 0;
          color: #333;
        }
        
        .metrics-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
          gap: 10px;
        }
        
        .metric-item {
          display: flex;
          justify-content: space-between;
          padding: 8px;
          background: #f8f9fa;
          border-radius: 4px;
        }
        
        .metric-label {
          font-weight: bold;
          color: #666;
        }
        
        .metric-value {
          color: #333;
        }
        
        .proof-details {
          background: white;
          border-radius: 6px;
          padding: 15px;
          margin-bottom: 20px;
        }
        
        .proof-details h4 {
          margin: 0 0 15px 0;
          color: #333;
        }
        
        .proof-info {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }
        
        .proof-item {
          display: flex;
          justify-content: space-between;
          padding: 8px;
          background: #f8f9fa;
          border-radius: 4px;
        }
        
        .proof-label {
          font-weight: bold;
          color: #666;
        }
        
        .proof-value {
          font-family: monospace;
          color: #333;
          word-break: break-all;
        }
        
        .proof-actions {
          display: flex;
          gap: 10px;
          justify-content: center;
        }
        
        .retry-button, .verify-button {
          background: #007bff;
          color: white;
          border: none;
          padding: 10px 20px;
          border-radius: 4px;
          cursor: pointer;
          font-size: 14px;
        }
        
        .retry-button:hover, .verify-button:hover {
          background: #0056b3;
        }
        
        .retry-button:disabled {
          background: #6c757d;
          cursor: not-allowed;
        }
        `
      }} />
    </div>
  );
};

export type { WorkProof, PerformanceMetrics, BenchmarkResult, VerificationStatus }; 