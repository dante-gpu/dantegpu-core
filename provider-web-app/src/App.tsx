import React, { useEffect, useRef, useState } from 'react';
import { Terminal, Zap, Cpu, Database, Shield, Server, Cloud, DollarSign, BarChart3, Activity } from 'lucide-react';
import { initializeAnimations } from './animation';
import './index.css';

// Type definitions for Terminal Stream
interface TerminalLogEntry {
  timestamp: string;
  source: string;
  command: string;
  output: string;
  type: string;
  color: string;
  step_id?: string;
  progress?: number;
}

const App: React.FC = () => {
  const [blurEnabled, setBlurEnabled] = useState(true);
  const [soundEnabled, setSoundEnabled] = useState(false);
  const [animationsInitialized, setAnimationsInitialized] = useState(false);
  const [terminalLogs, setTerminalLogs] = useState<string[]>([
    'DanteGPU Platform Starting...',
    'Loading core services...',
    'Service Discovery (Consul) initialized',
    'NATS JetStream connected',
    'API Gateway (Siger) started on port 8000',
    'Auth Service (Minos) ready',
    'Provider Registry (Statius) online',
    'Scheduler Orchestrator (Matilda) running',
    'Storage Service (Cacciaguida) connected',
    'Billing Service (Geryon) initialized',
    'Monitoring Service (Lucia) collecting metrics',
    'Platform ready for GPU providers',
    'Agora marketplace active'
  ]);
  const [isTestRunning, setIsTestRunning] = useState(false);

  const terminalRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Initialize GSAP animations only once
    if (!animationsInitialized) {
      initializeAnimations();
      setAnimationsInitialized(true);
    }
    
    // Auto-scroll terminal
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
    }
  }, [terminalLogs]);

  useEffect(() => {
    const connectWebSocket = () => {
      // Connect to our new terminal streaming service
      const ws = new WebSocket('ws://localhost:8888/ws');

      ws.onopen = () => {
        setTerminalLogs(prev => [...prev.slice(-12), '[Terminal Stream] Connected to DanteGPU Terminal Streaming Service']);
      };

      ws.onmessage = (event) => {
        try {
          const logEntry: TerminalLogEntry = JSON.parse(event.data);
          const timestamp = new Date(logEntry.timestamp).toLocaleTimeString();
          
          // Enhanced log formatting with progress and step tracking
          let logLine = `[${timestamp}] [${logEntry.source}] ${logEntry.output}`;
          
          // Add progress indicator if available
          if (logEntry.progress !== undefined && logEntry.progress > 0) {
            const progressBar = '█'.repeat(Math.floor(logEntry.progress / 5)) + '░'.repeat(20 - Math.floor(logEntry.progress / 5));
            logLine += ` [${progressBar}] ${logEntry.progress.toFixed(1)}%`;
          }
          
          // Add step ID if available
          if (logEntry.step_id) {
            logLine += ` (${logEntry.step_id})`;
          }
          
          setTerminalLogs(prev => [...prev.slice(-30), logLine]);
        } catch {
          // If it's not JSON, just display the raw message
          setTerminalLogs(prev => [...prev.slice(-30), event.data]);
        }
      };

      ws.onclose = () => {
        setTerminalLogs(prev => [...prev.slice(-30), '[Terminal Stream] Connection closed. Attempting to reconnect in 5 seconds...']);
        setTimeout(connectWebSocket, 5000);
      };

      ws.onerror = (error) => {
        setTerminalLogs(prev => [...prev.slice(-30), `[Terminal Stream] Error: ${error.type}.`]);
        ws.close();
      };
    };

    if (animationsInitialized) {
        connectWebSocket();
    }

    // Cleanup logic is handled by the reconnection flow
  }, [animationsInitialized]);

  const toggleBlur = () => {
    const svgGroup = document.querySelector('g[clip-path="url(#clip)"]');
    if (blurEnabled) {
      svgGroup?.removeAttribute('filter');
      setBlurEnabled(false);
    } else {
      svgGroup?.setAttribute('filter', 'url(#blur)');
      setBlurEnabled(true);
    }
  };

  const startComprehensiveTest = async () => {
    if (isTestRunning) return;
    
    setIsTestRunning(true);
    setTerminalLogs(prev => [...prev.slice(-30), '[System] Starting comprehensive rental system test...']);
    
    try {
      const response = await fetch('http://localhost:8888/api/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (response.ok) {
        setTerminalLogs(prev => [...prev.slice(-30), '[System] Comprehensive test initiated successfully']);
      } else {
        setTerminalLogs(prev => [...prev.slice(-30), '[System] Failed to start test - check terminal streaming service']);
      }
    } catch (error) {
      setTerminalLogs(prev => [...prev.slice(-30), `[System] Error starting test: ${error}`]);
    }
    
    // Reset button state after 5 seconds
    setTimeout(() => setIsTestRunning(false), 5000);
  };

  const services = [
    { icon: <Zap className="w-4 h-4" />, name: 'Gateway', tech: 'Siger' },
    { icon: <Shield className="w-4 h-4" />, name: 'Auth', tech: 'Minos' },
    { icon: <Server className="w-4 h-4" />, name: 'Registry', tech: 'Statius' },
    { icon: <Cpu className="w-4 h-4" />, name: 'Scheduler', tech: 'Matilda' },
    { icon: <Activity className="w-4 h-4" />, name: 'Daemon', tech: 'Leah' },
    { icon: <Cloud className="w-4 h-4" />, name: 'Storage', tech: 'Cacciaguida' },
    { icon: <DollarSign className="w-4 h-4" />, name: 'Billing', tech: 'Geryon' },
    { icon: <BarChart3 className="w-4 h-4" />, name: 'Monitoring', tech: 'Lucia' },
    { icon: <Database className="w-4 h-4" />, name: 'Blockchain', tech: 'Solana' }
  ];

  return (
    <div className="min-h-screen bg-gray-100 relative overflow-x-hidden">
      {/* Background Gradients */}
      <div className="bg-gradients">
        <div className="bg-gradient bg-gradient-1"></div>
        <div className="bg-gradient bg-gradient-2"></div>
        <div className="bg-gradient bg-gradient-3"></div>
      </div>
      <div className="gradient-overlay"></div>

      {/* Header */}
      <header className="fixed top-0 left-0 right-0 z-50 pointer-events-none">
        <div className="blur-toggle">
          <span className="blur-btn" onClick={toggleBlur}>
            7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump
          </span>
        </div>
        
        <nav>
          <div className="nav-item nav-top-left">DanteGPU Platform</div>
          <div className="nav-item nav-top-right">
            <span>
              <a href="mailto:info@dantegpu.com" className="email-link">info@dantegpu.com</a> — 2024-2025
            </span>
            <div className="sound-toggle" onClick={() => setSoundEnabled(!soundEnabled)}>
              <svg className="sound-wave" viewBox="0 0 20 12">
                <path 
                  className={`wave-line ${soundEnabled ? 'wave-animated' : ''}`} 
                  d="M2 6 Q4 2 6 6 Q8 10 10 6 Q12 2 14 6 Q16 10 18 6" 
                />
              </svg>
            </div>
          </div>
          <div className="nav-item nav-bottom-left">GPU as a Service</div>
          <div className="nav-item nav-bottom-center">Scroll to explore services</div>
          <div className="nav-item nav-bottom-right">Distributed Computing</div>
        </nav>
      </header>

      {/* Hero Section */}
      <section className="hero-section">
        <h1 className="hero-title">Distributed GPU Network</h1>
        
        <div className="hero-content">
          <div className="hero-nav">
            {services.map((service, index) => (
              <div key={index} className="hero-nav-item">
                <div className="flex items-center gap-2">
                  {service.icon}
                  <span>{service.name}</span>
                </div>
              </div>
            ))}
          </div>

          {/* Terminal Component */}
          <div className="terminal-container">
            <div className="terminal-header">
              <div className="terminal-controls">
                <div className="terminal-dot terminal-red"></div>
                <div className="terminal-dot terminal-yellow"></div>
                <div className="terminal-dot terminal-green"></div>
              </div>
              <div className="terminal-title">
                <Terminal className="w-4 h-4" />
                <span>DanteGPU Platform Logs</span>
              </div>
              <button 
                className={`terminal-test-btn ${isTestRunning ? 'running' : ''}`}
                onClick={startComprehensiveTest}
                disabled={isTestRunning}
              >
                <Zap className="w-3 h-3" />
                {isTestRunning ? 'Running...' : 'Start Test'}
              </button>
            </div>
            <div className="terminal-content" ref={terminalRef}>
              {terminalLogs.map((log, index) => (
                <div key={index} className="terminal-line">
                  <span className="terminal-prompt">dante@gpu-platform:~$</span>
                  <span className="terminal-text">{log}</span>
                </div>
              ))}
              <div className="terminal-cursor"></div>
            </div>
          </div>

          <div className="hero-text-content">
            <div className="hero-text">
              DanteGPU democratizes access to high-performance computing by creating a distributed network where GPU providers can monetize their unused resources. Our platform enables AI developers to access powerful GPU capabilities on a pay-per-use basis, eliminating hefty subscription costs and centralized control.
            </div>
            <div className="hero-text">
              Built on Solana blockchain technology, DanteGPU features the Agora marketplace for AI agents, real-time resource allocation, and transparent dGPU token transactions. Join our ecosystem of providers and developers reshaping the future of distributed computing and artificial intelligence.
            </div>
          </div>
        </div>
      </section>

      <div className="scroll-space"></div>

      {/* Animation Section */}
      <section className="animation-section">
        <div className="footer-container">
          <div className="svg-container">
            <svg className="spectrum-svg" viewBox="0 0 1567 584" preserveAspectRatio="none" fill="none">
              <g clipPath="url(#clip)" filter="url(#blur)">
                <path d="M1219 584H1393V184H1219V584Z" fill="url(#grad0)" />
                <path d="M1045 584H1219V104H1045V584Z" fill="url(#grad1)" />
                <path d="M348 584H174L174 184H348L348 584Z" fill="url(#grad2)" />
                <path d="M522 584H348L348 104H522L522 584Z" fill="url(#grad3)" />
                <path d="M697 584H522L522 54H697L697 584Z" fill="url(#grad4)" />
                <path d="M870 584H1045V54H870V584Z" fill="url(#grad5)" />
                <path d="M870 584H697L697 0H870L870 584Z" fill="url(#grad6)" />
                <path d="M174 585H0.000183105L-3.75875e-06 295H174L174 585Z" fill="url(#grad7)" />
                <path d="M1393 584H1567V294H1393V584Z" fill="url(#grad8)" />
              </g>
              <defs>
                <filter id="blur" x="-30" y="-30" width="1627" height="644" filterUnits="userSpaceOnUse" colorInterpolationFilters="sRGB">
                  <feFlood floodOpacity="0" result="BackgroundImageFix" />
                  <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
                  <feGaussianBlur stdDeviation="15" result="effect1_foregroundBlur" />
                </filter>
                {Array.from({ length: 9 }, (_, i) => (
                  <linearGradient key={i} id={`grad${i}`} x1="50%" y1="584" x2="50%" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#340B05" />
                    <stop offset="0.182709" stopColor="#0358F7" />
                    <stop offset="0.283673" stopColor="#5092C7" />
                    <stop offset="0.413484" stopColor="#E1ECFE" />
                    <stop offset="0.586565" stopColor="#FFD400" />
                    <stop offset="0.682722" stopColor="#FA3D1D" />
                    <stop offset="0.802892" stopColor="#FD02F5" />
                    <stop offset="1" stopColor="#FFC0FD" stopOpacity={0} />
                  </linearGradient>
                ))}
                <clipPath id="clip">
                  <rect width="1567" height="584" fill="white" />
                </clipPath>
              </defs>
            </svg>
          </div>

          <div className="main-title split-text">
            Where Computing Power Meets<br />
            Distributed Innovation
          </div>

          <div className="text-grid">
            <div className="text-column">
              <div className="wavelength-label level-1 split-text">API<br />Gateway<br />Siger</div>
            </div>
            <div className="text-column">
              <div className="wavelength-label level-2 split-text">Auth<br />Service<br />Minos</div>
            </div>
            <div className="text-column">
              <div className="wavelength-label level-3 split-text">Provider<br />Registry<br />Statius</div>
            </div>
            <div className="text-column">
              <div className="wavelength-label level-4 split-text">Scheduler<br />Orchestrator<br />Matilda</div>
            </div>
            <div className="text-column">
              <div className="wavelength-label level-5 split-text">Provider<br />Daemon<br />Leah</div>
            </div>
            <div className="text-column">
              <div className="wavelength-label level-4 split-text">Storage<br />Service<br />Cacciaguida</div>
            </div>
            <div className="text-column">
              <div className="wavelength-label level-3 split-text">Billing<br />Payment<br />Geryon</div>
            </div>
            <div className="text-column">
              <div className="wavelength-label level-2 split-text">Monitoring<br />Logging<br />Lucia</div>
            </div>
            <div className="text-column">
              <div className="wavelength-label level-1 split-text">Blockchain<br />Integration<br />Solana</div>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
};

export default App;