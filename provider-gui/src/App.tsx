import React, { useState, useEffect, useRef } from 'react';
import { invoke } from '@tauri-apps/api/tauri';
import { listen, Event as TauriEvent } from '@tauri-apps/api/event';

interface LogEntry {
  id: number;
  log_type: 'status' | 'stdout' | 'stderr' | 'error';
  message: string;
  timestamp: string;
}

// --- BEGIN NEW INTERFACES ---
interface GpuInfo {
  id: string;
  name: string;
  model: string;
  vram_total_mb: number;
  vram_free_mb: number;
  utilization_gpu_percent?: number;
  temperature_c?: number;
  power_draw_w?: number;
  is_available_for_rent: boolean;
  current_hourly_rate_dgpu: number | null;
}

interface ProviderSettings {
  default_hourly_rate_dgpu: number;
  preferred_currency: string; // e.g., "USD", "EUR", "DCORE"
  min_job_duration_minutes: number;
  max_concurrent_jobs: number;
}

interface LocalJob {
  id: string;
  name: string;
  status: 'running' | 'completed' | 'failed' | 'queued';
  progress_percent: number;
  started_at: string;
  estimated_time_remaining_seconds?: number;
}

interface NetworkInfo {
  connection_status: 'connected' | 'disconnected' | 'connecting';
  ip_address?: string;
  upload_speed_mbps?: number;
  download_speed_mbps?: number;
}

interface FinancialSummary {
  wallet_balance_dgpu: number;
  total_earned_dgpu: number;
  pending_payout_dgpu: number;
  last_payout_at?: string;
}

// === GPU RENTAL SYSTEM INTERFACES ===

interface GpuRentalListing {
  id: string;
  gpu_id: string;
  provider_id: string;
  provider_name: string;
  gpu_name: string;
  gpu_model: string;
  gpu_architecture: string;
  vram_gb: number;
  compute_units: number;
  base_clock_mhz: number;
  memory_clock_mhz: number;
  performance_score: number;
  location: string;
  availability_status: string;
  hourly_rate_usd: number;
  hourly_rate_dgpu: number;
  minimum_rental_hours: number;
  maximum_rental_hours: number;
  supported_frameworks: string[];
  container_support: boolean;
  ssh_access: boolean;
  jupyter_notebook: boolean;
  tensorboard: boolean;
  custom_docker_images: boolean;
  data_persistence: boolean;
  internet_access: boolean;
  verification_status: string;
  rating: number;
  total_reviews: number;
  total_rental_hours: number;
  provider_response_time_minutes: number;
  created_at: string;
  updated_at: string;
  tags: string[];
  special_offers: string[];
}

interface GpuRentalBooking {
  id: string;
  listing_id: string;
  renter_id: string;
  renter_name: string;
  provider_id: string;
  gpu_id: string;
  booking_status: string;
  booking_type: string;
  start_time: string;
  end_time: string;
  duration_hours: number;
  hourly_rate_usd: number;
  hourly_rate_dgpu: number;
  total_cost_usd: number;
  total_cost_dgpu: number;
  payment_status: string;
  payment_method: string;
  escrow_transaction_id?: string;
  job_specifications: JobSpecification;
  container_config: ContainerConfiguration;
  resource_allocation: ResourceAllocation;
  current_job_id?: string;
  ssh_connection_info?: SshConnectionInfo;
  monitoring_endpoints: string[];
  file_uploads: FileUpload[];
  results_download: string[];
  booking_notes: string;
  cancellation_policy: string;
  auto_extend: boolean;
  extension_hours: number;
  created_at: string;
  updated_at: string;
  confirmed_at?: string;
  started_at?: string;
  completed_at?: string;
  cancelled_at?: string;
}

interface JobSpecification {
  job_name: string;
  job_description: string;
  framework: string;
  python_version: string;
  conda_environment?: string;
  pip_requirements: string[];
  environment_variables: Record<string, string>;
  startup_script?: string;
  data_sources: DataSource[];
  expected_outputs: string[];
  estimated_completion_time?: number;
  gpu_memory_requirements: number;
  cpu_requirements: number;
  ram_requirements: number;
  storage_requirements: number;
  network_requirements: boolean;
  checkpoint_frequency: number;
  auto_restart_on_failure: boolean;
  max_retries: number;
  priority: string;
}

interface ContainerConfiguration {
  base_image: string;
  custom_dockerfile?: string;
  port_mappings: PortMapping[];
  volume_mounts: VolumeMount[];
  resource_limits: ResourceLimits;
  security_context: SecurityContext;
  networking_mode: string;
  gpu_access: boolean;
  privileged_mode: boolean;
  shared_memory_size: number;
  ulimits: Record<string, number>;
}

interface ResourceAllocation {
  allocated_gpu_memory_mb: number;
  allocated_cpu_cores: number;
  allocated_ram_mb: number;
  allocated_storage_gb: number;
  allocated_network_bandwidth_mbps: number;
  gpu_utilization_limit: number;
  cpu_utilization_limit: number;
  memory_utilization_limit: number;
  process_limit: number;
  file_descriptor_limit: number;
  network_connections_limit: number;
}

interface SshConnectionInfo {
  hostname: string;
  port: number;
  username: string;
  private_key?: string;
  public_key: string;
  password?: string;
  connection_url: string;
  jupyter_url?: string;
  tensorboard_url?: string;
  monitoring_url?: string;
}

interface FileUpload {
  id: string;
  filename: string;
  file_size_bytes: number;
  file_type: string;
  upload_url: string;
  download_url: string;
  checksum: string;
  upload_status: string;
  created_at: string;
  expires_at?: string;
}

interface DataSource {
  id: string;
  name: string;
  source_type: string;
  source_url: string;
  access_credentials?: Record<string, string>;
  size_bytes: number;
  format: string;
  description: string;
  preprocessing_required: boolean;
}

interface PortMapping {
  host_port: number;
  container_port: number;
  protocol: string;
  description: string;
}

interface VolumeMount {
  host_path: string;
  container_path: string;
  read_only: boolean;
  volume_type: string;
}

interface ResourceLimits {
  max_cpu_cores: number;
  max_memory_mb: number;
  max_storage_gb: number;
  max_gpu_memory_mb: number;
  max_network_bandwidth_mbps: number;
  max_processes: number;
  max_file_descriptors: number;
  max_execution_time_hours: number;
}

interface SecurityContext {
  run_as_user: number;
  run_as_group: number;
  fs_group: number;
  capabilities_add: string[];
  capabilities_drop: string[];
  read_only_root_filesystem: boolean;
  allow_privilege_escalation: boolean;
  seccomp_profile?: string;
  selinux_options?: Record<string, string>;
}

interface RentalMarketplace {
  available_listings: GpuRentalListing[];
  active_bookings: GpuRentalBooking[];
  booking_history: GpuRentalBooking[];
  user_favorites: string[];
  price_alerts: PriceAlert[];
  search_filters: SearchFilters;
  marketplace_stats: MarketplaceStats;
}

interface PriceAlert {
  id: string;
  user_id: string;
  gpu_model: string;
  max_price_usd: number;
  max_price_dgpu: number;
  location_preference?: string;
  minimum_rating: number;
  alert_frequency: string;
  is_active: boolean;
  created_at: string;
}

interface SearchFilters {
  gpu_models: string[];
  min_vram_gb?: number;
  max_vram_gb?: number;
  min_price_usd?: number;
  max_price_usd?: number;
  min_price_dgpu?: number;
  max_price_dgpu?: number;
  locations: string[];
  frameworks: string[];
  availability_status: string[];
  verification_status: string[];
  min_rating?: number;
  sort_by: string;
  results_per_page: number;
  current_page: number;
}

interface MarketplaceStats {
  total_listings: number;
  available_listings: number;
  average_hourly_rate_usd: number;
  average_hourly_rate_dgpu: number;
  total_rental_hours_today: number;
  total_revenue_today_usd: number;
  total_revenue_today_dgpu: number;
  most_popular_gpu_models: string[];
  average_booking_duration: number;
  user_satisfaction_rating: number;
  dispute_rate: number;
  top_performing_providers: string[];
}

interface ProviderEarnings {
  provider_id: string;
  current_balance_usd: number;
  current_balance_dgpu: number;
  pending_earnings_usd: number;
  pending_earnings_dgpu: number;
  total_lifetime_earnings_usd: number;
  total_lifetime_earnings_dgpu: number;
  earnings_today_usd: number;
  earnings_today_dgpu: number;
  earnings_this_week_usd: number;
  earnings_this_week_dgpu: number;
  earnings_this_month_usd: number;
  earnings_this_month_dgpu: number;
  total_rental_hours: number;
  total_completed_bookings: number;
  average_hourly_rate_usd: number;
  average_hourly_rate_dgpu: number;
  provider_rating: number;
  response_time_minutes: number;
  cancellation_rate: number;
  dispute_rate: number;
  payout_schedule: string;
  next_payout_date: string;
  payout_method: string;
  tax_information: TaxInformation;
  performance_metrics: ProviderPerformanceMetrics;
}

interface TaxInformation {
  tax_id?: string;
  business_type: string;
  country: string;
  state_province: string;
  tax_rate: number;
  tax_exemption: boolean;
  documents_submitted: boolean;
  verification_status: string;
}

interface ProviderPerformanceMetrics {
  gpu_utilization_average: number;
  uptime_percentage: number;
  job_success_rate: number;
  customer_satisfaction: number;
  response_time_hours: number;
  issue_resolution_time_hours: number;
  repeat_customer_rate: number;
  referral_rate: number;
  revenue_growth_rate: number;
  market_share_percentage: number;
}
// --- END NEW INTERFACES ---

let logIdCounter = 0;

function App() {
  const [daemonStatus, setDaemonStatus] = useState<string>('OFFLINE');
  const [daemonActive, setDaemonActive] = useState<boolean>(false);
  const [daemonError, setDaemonError] = useState<string | null>(null);
  const [daemonLogs, setDaemonLogs] = useState<LogEntry[]>([]);
  const [systemInfo, setSystemInfo] = useState<string>('');
  const [processInfo, setProcessInfo] = useState<string>('');
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const logsEndRef = useRef<null | HTMLDivElement>(null);

  // --- BEGIN NEW STATE VARIABLES ---
  const [gpus, setGpus] = useState<GpuInfo[]>([]);
  const [providerSettings, setProviderSettings] = useState<ProviderSettings | null>(null);
  const [localJobs, setLocalJobs] = useState<LocalJob[]>([]);
  const [networkStatus, setNetworkStatus] = useState<NetworkInfo | null>(null);
  const [financialSummary, setFinancialSummary] = useState<FinancialSummary | null>(null);
  
  const [selectedGpu, setSelectedGpu] = useState<GpuInfo | null>(null);
  const [gpuRentalModalOpen, setGpuRentalModalOpen] = useState<boolean>(false);
  const [newRentalRate, setNewRentalRate] = useState<string>("");
  
  // === GPU RENTAL SYSTEM STATE ===
  const [rentalMarketplace, setRentalMarketplace] = useState<RentalMarketplace | null>(null);
  const [providerEarnings, setProviderEarnings] = useState<ProviderEarnings | null>(null);
  const [activeBookings, setActiveBookings] = useState<GpuRentalBooking[]>([]);
  const [bookingHistory, setBookingHistory] = useState<GpuRentalBooking[]>([]);
  const [availableListings, setAvailableListings] = useState<GpuRentalListing[]>([]);
  const [searchFilters, setSearchFilters] = useState<SearchFilters>({
    gpu_models: [],
    min_vram_gb: undefined,
    max_vram_gb: undefined,
    min_price_usd: undefined,
    max_price_usd: undefined,
    min_price_dgpu: undefined,
    max_price_dgpu: undefined,
    locations: [],
    frameworks: [],
    availability_status: [],
    verification_status: [],
    min_rating: undefined,
    sort_by: 'price_low',
    results_per_page: 20,
    current_page: 1,
  });
  const [selectedBooking, setSelectedBooking] = useState<GpuRentalBooking | null>(null);
  const [newJobSpec, setNewJobSpec] = useState<JobSpecification>({
    job_name: '',
    job_description: '',
    framework: 'pytorch',
    python_version: '3.9',
    conda_environment: undefined,
    pip_requirements: [],
    environment_variables: {},
    startup_script: undefined,
    data_sources: [],
    expected_outputs: [],
    estimated_completion_time: undefined,
    gpu_memory_requirements: 8192,
    cpu_requirements: 4,
    ram_requirements: 16384,
    storage_requirements: 50,
    network_requirements: true,
    checkpoint_frequency: 300,
    auto_restart_on_failure: true,
    max_retries: 3,
    priority: 'normal',
  });
  const [activeView, setActiveView] = useState<'overview' | 'rental-marketplace' | 'provider-dashboard' | 'bookings' | 'earnings' | 'job-submission'>('overview');
  const [showListingModal, setShowListingModal] = useState(false);
  const [showBookingModal, setShowBookingModal] = useState(false);
  const [showJobModal, setShowJobModal] = useState(false);
  const [selectedListing, setSelectedListing] = useState<GpuRentalListing | null>(null);
  const [bookingForm, setBookingForm] = useState({
    renter_name: '',
    duration_hours: 4,
    booking_type: 'immediate',
  });
  const [earningsRefreshInterval, setEarningsRefreshInterval] = useState<number | null>(null);
  // --- END NEW STATE VARIABLES ---

  const addLog = (type: LogEntry['log_type'], message: string) => {
    const newLog: LogEntry = {
      id: logIdCounter++,
      log_type: type,
      message,
      timestamp: new Date().toISOString(),
    };
    setDaemonLogs((prevLogs) => [...prevLogs.slice(-200), newLog]);
  };

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [daemonLogs]);

  useEffect(() => {
    const unlisteners: Array<() => void> = [];

    const setupListener = async <T,>(eventName: string, handler: (event: TauriEvent<T>) => void) => {
      const unlisten = await listen<T>(eventName, handler);
      unlisteners.push(unlisten);
    };

    // Enhanced daemon log listener with proper interface
    setupListener<LogEntry>('daemon_log', (event) => {
      const logEntry = event.payload;
      setDaemonLogs((prevLogs) => [...prevLogs.slice(-200), logEntry]);
      
      // Update daemon status based on log content
      if (logEntry.log_type === 'status') {
        setDaemonStatus(logEntry.message);
        if (logEntry.message.includes(' All systems operational')) {
          setDaemonActive(true);
          setDaemonError(null);
        } else if (logEntry.message.includes('') || logEntry.message.includes('error')) {
          setDaemonActive(false);
          setDaemonError(logEntry.message);
        }
      }
    });

    // Legacy listeners for backward compatibility
    setupListener<string>('daemon-status', (event) => {
      const statusPayload = event.payload;
      setDaemonStatus(statusPayload);
      setDaemonError(null);
      if (statusPayload.toLowerCase().includes('running') || statusPayload.toLowerCase().includes('started')) {
        setDaemonActive(true);
      } else if (statusPayload.toLowerCase().includes('stopped') || statusPayload.toLowerCase().includes('killed') || statusPayload.toLowerCase().includes('not started') || statusPayload.toLowerCase().includes('offline')) {
        setDaemonActive(false);
      }
      addLog('status', statusPayload);
    });

    setupListener<string>('daemon-stdout', (event) => {
      addLog('stdout', event.payload);
    });

    setupListener<string>('daemon-stderr', (event) => {
      addLog('stderr', event.payload);
    });

    setupListener<string>('daemon-error', (event) => {
      const errorMsg = `Error: ${event.payload}`;
      setDaemonStatus(errorMsg);
      setDaemonError(event.payload); 
      setDaemonActive(false);
      addLog('error', errorMsg);
    });
    
    addLog('status', 'Provider GUI initialized. Daemon is OFFLINE.');

    // --- BEGIN FETCHING INTEGRATED DATA ---
    const fetchIntegratedData = async () => {
      try {
        const data = await invoke<any>('get_daemon_integrated_data');
        
        // Update daemon status
        const newDaemonStatus = data.daemon_status.toUpperCase();
        setDaemonStatus(newDaemonStatus);
        
        if (data.daemon_status === 'online') {
          setDaemonActive(true);
          setDaemonError(null);
          
          // Update all data when daemon is online
          if (data.gpus && data.gpus.length > 0) {
            setGpus(data.gpus);
            addLog('status', `Live: ${data.gpus.length} GPUs detected and ready for rental`);
          }
          
          if (data.jobs) {
            setLocalJobs(data.jobs);
            if (data.jobs.length > 0) {
              addLog('status', `Live: ${data.jobs.length} active jobs running`);
            }
          }
          
          if (data.financial_summary) {
            setFinancialSummary(data.financial_summary);
            addLog('status', `Live: Financial data updated - Balance: ${data.financial_summary.current_balance_dgpu} DGPU`);
          }
          
          if (data.system_health) {
            addLog('status', `Live: System health - CPU: ${data.system_health.cpu_usage_percent}%, Memory: ${data.system_health.memory_usage_percent}%`);
          }
          
          if (data.network_status) {
            setNetworkStatus(data.network_status);
            addLog('status', `Live: Network connected - ${data.network_status.connection_type}`);
          }
        } else {
          setDaemonActive(false);
          setGpus([]);
          setLocalJobs([]);
          setFinancialSummary(null);
          setNetworkStatus(null);
          addLog('status', `Daemon is ${data.daemon_status} - Services unavailable`);
        }
      } catch (err) {
        addLog('error', `Failed to fetch integrated data: ${err}`);
        setDaemonStatus('ERROR');
        setDaemonActive(false);
        setDaemonError(err as string);
      }
    };

    // Set up automatic data fetching
    fetchIntegratedData();
    const dataInterval = setInterval(fetchIntegratedData, 3000); // Check every 3 seconds

    // TODO: Consider adding listeners for real-time updates to this data if backend supports it.
    // e.g., await listen('gpu-update', (event) => { setGpus(event.payload); });
     // --- END FETCHING INTEGRATED DATA ---

    return () => {
      unlisteners.forEach(unlisten => unlisten());
      clearInterval(dataInterval);
    };
  }, []); // Run once on mount and set up polling

  const handleStartDaemon = async () => {
    addLog('status', 'Attempting to start daemon...');
    setDaemonError(null);
    setIsLoading(true);
    try {
      const result = await invoke('start_daemon');
      addLog('status', `Success: ${result}`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      const UImessage = `Failed to send start command: ${errorMessage}`;
      setDaemonStatus(UImessage);
      setDaemonError(errorMessage);
      setDaemonActive(false);
      addLog('error', UImessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleStopDaemon = async () => {
    addLog('status', 'Attempting to stop daemon...');
    setDaemonError(null);
    setIsLoading(true);
    try {
      const result = await invoke('stop_daemon');
      addLog('status', `Success: ${result}`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      const UImessage = `Failed to send stop command: ${errorMessage}`;
      setDaemonStatus(UImessage);
      setDaemonError(errorMessage);
      addLog('error', UImessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleRestartDaemon = async () => {
    addLog('status', 'Attempting to restart daemon...');
    setIsLoading(true);
    try {
      const result = await invoke('restart_daemon');
      addLog('status', `Success: ${result}`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      addLog('error', `Failed to restart daemon: ${errorMessage}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleGetSystemInfo = async () => {
    try {
      const result = await invoke('get_system_info');
      setSystemInfo(result as string);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      addLog('error', ` Failed to get system info: ${errorMessage}`);
    }
  };

  const handleGetProcessInfo = async () => {
    try {
      const result = await invoke('get_process_info');
      setProcessInfo(result as string);
      addLog('status', `Process info: ${result}`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      addLog('error', `Failed to get process info: ${errorMessage}`);
    }
  };

  const handleCheckDockerServices = async () => {
    try {
      const result = await invoke('check_docker_services');
      addLog('status', `Docker services: ${result}`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      addLog('error', `Failed to check Docker services: ${errorMessage}`);
    }
  };

  const handleClearLogs = async () => {
    try {
      setDaemonLogs([]);
      await invoke('clear_logs');
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      addLog('error', `Failed to clear logs: ${errorMessage}`);
    }
  };

  const handleCheckEnvironment = async () => {
    try {
      const result = await invoke('check_system_environment');
      addLog('status', `Environment: ${result}`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      addLog('error', `Failed to check environment: ${errorMessage}`);
    }
  };

  const handleCheckPorts = async () => {
    try {
      const result = await invoke('check_required_ports');
      addLog('status', `Ports: ${result}`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      addLog('error', `Failed to check ports: ${errorMessage}`);
    }
  };

  const getLogClass = (type: LogEntry['log_type']) => {
    switch (type) {
      case 'stdout': return 'log-stdout';
      case 'stderr': return 'log-stderr';
      case 'error': return 'log-error';
      case 'status': return 'log-status';
      default: return '';
    }
  };

  const getLogIcon = (type: LogEntry['log_type']) => {
    switch (type) {
      case 'stdout': return '';
      case 'stderr': return '';      case 'error': return '';
      case 'status': return '';
      default: return '';
    }
  };

  const getStatusDisplayClass = () => {
    if (daemonError) return 'status-error';
    if (daemonActive) return 'status-on';
    return 'status-off';
  };

  // --- BEGIN HANDLERS FOR NEW FEATURES ---
  const handleOpenGpuRentalModal = (gpu: GpuInfo) => {
    setSelectedGpu(gpu);
    setNewRentalRate(gpu.current_hourly_rate_dgpu?.toString() || providerSettings?.default_hourly_rate_dgpu.toString() || "0");
    setGpuRentalModalOpen(true);
  };

  const handleCloseGpuRentalModal = () => {
    setGpuRentalModalOpen(false);
    setSelectedGpu(null);
    setNewRentalRate("");
  };

  const handleUpdateGpuRental = async () => {
    if (!selectedGpu) return;
    const rate = parseFloat(newRentalRate);
    if (isNaN(rate) || rate < 0) {
      addLog('error', "Invalid rental rate provided.");
      return;
    }
    try {
      await invoke('update_gpu_rental_settings', { 
        gpuId: selectedGpu.id, 
        isAvailable: !selectedGpu.is_available_for_rent, // Toggle availability
        hourlyRate: rate 
      });
      addLog('status', `Successfully updated rental settings for GPU ${selectedGpu.name}. Toggled availability.`);
      // Refresh GPU list
      const updatedGpus = await invoke<GpuInfo[]>('get_detected_gpus');
      setGpus(updatedGpus);
    } catch (err) {
      addLog('error', `Failed to update GPU ${selectedGpu.name} rental settings: ${err}`);
    }
    handleCloseGpuRentalModal();
  };
  
  const handleToggleGpuAvailability = async (gpu: GpuInfo) => {
    try {
      await invoke('update_gpu_rental_settings', { 
        gpuId: gpu.id, 
        isAvailable: !gpu.is_available_for_rent,
        hourlyRate: gpu.current_hourly_rate_dgpu // Keep current rate or use default if null
      });
      addLog('status', `Toggled availability for GPU ${gpu.name}.`);
      const updatedGpus = await invoke<GpuInfo[]>('get_detected_gpus');
      setGpus(updatedGpus);
    } catch (err) {
      addLog('error', `Failed to toggle availability for GPU ${gpu.name}: ${err}`);
    }
  };

  const handleSaveProviderSettings = async () => {
    if (!providerSettings) return;
    try {
      await invoke('save_provider_settings', { settings: providerSettings });
      addLog('status', 'Provider settings saved.');
    } catch (err) {
      addLog('error', `Failed to save provider settings: ${err}`);
    }
  };

  // === GPU RENTAL SYSTEM HANDLERS ===

  const fetchRentalData = async () => {
    if (!daemonActive) return;
    
    try {
      const marketplace = await invoke<RentalMarketplace>('get_rental_marketplace');
      setRentalMarketplace(marketplace);
      setAvailableListings(marketplace.available_listings);
      setActiveBookings(marketplace.active_bookings);
      setBookingHistory(marketplace.booking_history);
      addLog('status', 'Fetched rental marketplace data.');
    } catch (err) {
      addLog('error', `Failed to fetch rental marketplace: ${err}`);
    }

    try {
      const earnings = await invoke<ProviderEarnings>('get_provider_earnings');
      setProviderEarnings(earnings);
      addLog('status', 'Fetched provider earnings data.');
    } catch (err) {
      addLog('error', `Failed to fetch provider earnings: ${err}`);
    }
  };

  const handleCreateGpuListing = async (gpuId: string) => {
    if (!selectedGpu) return;
    
    try {
      const listing = await invoke<GpuRentalListing>('create_gpu_rental_listing', {
        gpuId,
        hourlyRateUsd: 4.5,
        hourlyRateDgpu: 1.2,
        minimumRentalHours: 1,
        maximumRentalHours: 168,
        supportedFrameworks: ['pytorch', 'tensorflow', 'cuda', 'jupyter'],
        specialOffers: ['First time rental discount', 'Extended rental bonus'],
      });
      
      setAvailableListings(prev => [...prev, listing]);
      addLog('status', `Created GPU rental listing: ${listing.id}`);
      
      // Refresh marketplace data
      await fetchRentalData();
    } catch (err) {
      addLog('error', `Failed to create GPU listing: ${err}`);
    }
  };

  const handleCreateBooking = async (listingId: string) => {
    if (!selectedListing) return;
    
    try {
      const booking = await invoke<GpuRentalBooking>('create_gpu_rental_booking', {
        listingId,
        renterId: 'test_user_123',
        renterName: bookingForm.renter_name,
        durationHours: bookingForm.duration_hours,
        bookingType: bookingForm.booking_type,
        jobSpecification: newJobSpec,
      });
      
      setActiveBookings(prev => [...prev, booking]);
      addLog('status', `Created booking: ${booking.id} for $${booking.total_cost_usd.toFixed(2)}`);
      setShowBookingModal(false);
      
      // Refresh marketplace data
      await fetchRentalData();
    } catch (err) {
      addLog('error', `Failed to create booking: ${err}`);
    }
  };

  const handleStartRentalJob = async (bookingId: string) => {
    try {
      const result = await invoke<string>('start_rental_job', { bookingId });
      addLog('status', result);
      
      // Refresh bookings
      await fetchRentalData();
    } catch (err) {
      addLog('error', `Failed to start rental job: ${err}`);
    }
  };

  const handleCompleteRentalJob = async (bookingId: string) => {
    try {
      const result = await invoke<string>('complete_rental_job', { bookingId });
      addLog('status', result);
      
      // Refresh bookings and earnings
      await fetchRentalData();
    } catch (err) {
      addLog('error', `Failed to complete rental job: ${err}`);
    }
  };

  const handleSearchGpuRentals = async (filters: SearchFilters) => {
    try {
      const results = await invoke<GpuRentalListing[]>('search_gpu_rentals', { filters });
      setAvailableListings(results);
      addLog('status', `Found ${results.length} GPU rentals matching filters.`);
    } catch (err) {
      addLog('error', `Failed to search GPU rentals: ${err}`);
    }
  };

  const handleViewChange = (view: typeof activeView) => {
    setActiveView(view);
    addLog('status', `Switched to ${view} view.`);
  };

  const handleOpenListingModal = (listing: GpuRentalListing) => {
    setSelectedListing(listing);
    setShowListingModal(true);
  };

  const handleCloseListingModal = () => {
    setSelectedListing(null);
    setShowListingModal(false);
  };

  const handleOpenBookingModal = (listing: GpuRentalListing) => {
    setSelectedListing(listing);
    setShowBookingModal(true);
  };

  const handleCloseBookingModal = () => {
    setSelectedListing(null);
    setShowBookingModal(false);
  };

  const handleOpenJobModal = (booking: GpuRentalBooking) => {
    setSelectedBooking(booking);
    setShowJobModal(true);
  };

  const handleCloseJobModal = () => {
    setSelectedBooking(null);
    setShowJobModal(false);
  };

  const formatCurrency = (amount: number, currency: 'usd' | 'dgpu') => {
    if (currency === 'usd') {
      return `$${amount.toFixed(2)}`;
    } else {
      return `${amount.toFixed(2)} DGPU`;
    }
  };

  const formatDuration = (hours: number) => {
    if (hours < 24) {
      return `${hours}h`;
    } else {
      const days = Math.floor(hours / 24);
      const remainingHours = hours % 24;
      return `${days}d ${remainingHours}h`;
    }
  };

  const getBookingStatusColor = (status: string) => {
    switch (status) {
      case 'active': return '#4CAF50';
      case 'completed': return '#2196F3';
      case 'pending': return '#FF9800';
      case 'cancelled': return '#F44336';
      case 'disputed': return '#9C27B0';
      default: return '#757575';
    }
  };

  const getPaymentStatusColor = (status: string) => {
    switch (status) {
      case 'paid': return '#4CAF50';
      case 'pending': return '#FF9800';
      case 'refunded': return '#2196F3';
      case 'disputed': return '#F44336';
      default: return '#757575';
    }
  };

  // Auto-refresh earnings data every 30 seconds
  useEffect(() => {
    if (daemonActive && activeView === 'earnings') {
      const interval = setInterval(() => {
        fetchRentalData();
      }, 30000);
      setEarningsRefreshInterval(interval);
      
      return () => {
        if (earningsRefreshInterval) {
          clearInterval(earningsRefreshInterval);
        }
      };
    }
  }, [daemonActive, activeView]);

  // Fetch rental data when daemon becomes active
  useEffect(() => {
    if (daemonActive) {
      fetchRentalData();
    }
  }, [daemonActive]);
  // --- END HANDLERS FOR NEW FEATURES ---

  return (
    <div className="app-container">
      <header className="header">
        <h1>Dante GPU Provider</h1>
        <p>Comprehensive GPU rental management platform with real-time monitoring.</p>
        
        {/* Navigation Tabs */}
        <nav className="navigation-tabs">
          <button 
            className={activeView === 'overview' ? 'nav-tab active' : 'nav-tab'}
            onClick={() => handleViewChange('overview')}
          >
            Overview
          </button>
          <button 
            className={activeView === 'rental-marketplace' ? 'nav-tab active' : 'nav-tab'}
            onClick={() => handleViewChange('rental-marketplace')}
          >
            Rental Marketplace
          </button>
          <button 
            className={activeView === 'provider-dashboard' ? 'nav-tab active' : 'nav-tab'}
            onClick={() => handleViewChange('provider-dashboard')}
          >
            Provider Dashboard
          </button>
          <button 
            className={activeView === 'bookings' ? 'nav-tab active' : 'nav-tab'}
            onClick={() => handleViewChange('bookings')}
          >
            Bookings
          </button>
          <button 
            className={activeView === 'earnings' ? 'nav-tab active' : 'nav-tab'}
            onClick={() => handleViewChange('earnings')}
          >
            Earnings
          </button>
          <button 
            className={activeView === 'job-submission' ? 'nav-tab active' : 'nav-tab'}
            onClick={() => handleViewChange('job-submission')}
          >
            Job Submission
          </button>
        </nav>
      </header>

      <section className="controls-card">
        <div className={`status-display ${getStatusDisplayClass()}`}>
          Daemon Status: {daemonStatus}
        </div>
        <div className="controls">
          <button onClick={handleStartDaemon} disabled={daemonActive || isLoading}>
            {isLoading ? 'Starting...' : 'Start Daemon'}
          </button>
          <button onClick={handleStopDaemon} disabled={!daemonActive || isLoading}>
            {isLoading ? 'Stopping...' : 'Stop Daemon'}
          </button>
          <button onClick={handleRestartDaemon} disabled={isLoading}>
            {isLoading ? 'Restarting...' : 'Restart Daemon'}
          </button>
        </div>
      </section>

      {/* === COMPREHENSIVE VIEW SYSTEM === */}
      {activeView === 'overview' && (
        <>
          {/* System Management Controls */}
          <section className="card">
            <h2>System Management</h2>
            <div className="controls">
              <button onClick={handleGetSystemInfo} className="secondary">
                System Info
              </button>
              <button onClick={handleGetProcessInfo} className="secondary">
                Process Info
              </button>
              <button onClick={handleCheckDockerServices} className="secondary">
                Docker Services
              </button>
              <button onClick={handleCheckEnvironment} className="secondary">
                Environment
              </button>
              <button onClick={handleCheckPorts} className="secondary">
                Port Status
              </button>
              <button onClick={handleClearLogs} className="secondary">
                Clear Logs
              </button>
            </div>
            
            {/* System Info Display */}
            {systemInfo && (
              <div className="system-info-display">
                <h4>System Information</h4>
                <pre>{systemInfo}</pre>
              </div>
            )}
            
            {/* Process Info Display */}
            {processInfo && (
              <div className="process-info-display">
                <h4>Process Information</h4>
                <p>{processInfo}</p>
              </div>
            )}
          </section>

          {/* GPU Management */}
          <section className="card">
            <h2>GPU Management</h2>
            {daemonActive ? (
              gpus.length > 0 ? (
                <div className="gpu-list">
                  {gpus.map(gpu => (
                    <div key={gpu.id} className="gpu-item card">
                      <h3>{gpu.name} ({gpu.model})</h3>
                      <p>VRAM: {gpu.vram_free_mb}MB Free / {gpu.vram_total_mb}MB Total</p>
                      {gpu.utilization_gpu_percent !== undefined && <p>Util: {gpu.utilization_gpu_percent}%</p>}
                      {gpu.temperature_c !== undefined && <p>Temp: {gpu.temperature_c}°C</p>}
                      {gpu.power_draw_w !== undefined && <p>Power: {gpu.power_draw_w}W</p>}
                      <p>Status: {gpu.is_available_for_rent ? 
                        <span className="status-rentable">Rentable @ {gpu.current_hourly_rate_dgpu} dGPU/hr</span> : 
                        <span className="status-private">Private</span>}
                      </p>
                      <div className="controls">
                        <button onClick={() => handleOpenGpuRentalModal(gpu)}>
                          {gpu.is_available_for_rent ? 'Edit Rental' : 'Make Rentable'}
                        </button>
                         <button onClick={() => handleToggleGpuAvailability(gpu)}>
                          {gpu.is_available_for_rent ? 'Make Private' : 'Make Rentable (No Price Change)'}
                        </button>
                        <button onClick={() => handleCreateGpuListing(gpu.id)} className="primary">
                          Create Marketplace Listing
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p>No GPUs detected or data not yet loaded.</p>
              )
            ) : (
              <p>Daemon is offline. Start the daemon to manage GPUs.</p>
            )}
          </section>

          {/* Quick Stats Overview */}
          {providerEarnings && (
            <section className="card">
              <h2>Quick Stats Overview</h2>
              <div className="stats-grid">
                <div className="stat-card">
                  <h4>Today's Earnings</h4>
                  <p className="stat-value">{formatCurrency(providerEarnings.earnings_today_usd, 'usd')}</p>
                  <p className="stat-subtitle">{formatCurrency(providerEarnings.earnings_today_dgpu, 'dgpu')}</p>
                </div>
                <div className="stat-card">
                  <h4>Active Bookings</h4>
                  <p className="stat-value">{activeBookings.length}</p>
                  <p className="stat-subtitle">Current jobs running</p>
                </div>
                <div className="stat-card">
                  <h4>Total Completed</h4>
                  <p className="stat-value">{providerEarnings.total_completed_bookings}</p>
                  <p className="stat-subtitle">Lifetime bookings</p>
                </div>
                <div className="stat-card">
                  <h4>Provider Rating</h4>
                  <p className="stat-value">{providerEarnings.provider_rating.toFixed(1)}/5.0</p>
                  <p className="stat-subtitle">Customer satisfaction</p>
                </div>
              </div>
            </section>
          )}
        </>
      )}

      {activeView === 'rental-marketplace' && (
        <>
          {/* Rental Marketplace */}
          <section className="card">
            <h2>GPU Rental Marketplace</h2>
            {daemonActive ? (
              <>
                {/* Search and Filters */}
                <div className="marketplace-filters">
                  <div className="filter-row">
                    <label>Sort by:</label>
                    <select 
                      value={searchFilters.sort_by} 
                      onChange={(e) => setSearchFilters(prev => ({...prev, sort_by: e.target.value}))}
                    >
                      <option value="price_low">Price: Low to High</option>
                      <option value="price_high">Price: High to Low</option>
                      <option value="rating">Highest Rated</option>
                      <option value="performance">Best Performance</option>
                    </select>
                    <button onClick={() => handleSearchGpuRentals(searchFilters)} className="primary">
                      Search GPUs
                    </button>
                  </div>
                  <div className="filter-row">
                    <label>Min VRAM (GB):</label>
                    <input 
                      type="number" 
                      value={searchFilters.min_vram_gb || ''} 
                      onChange={(e) => setSearchFilters(prev => ({...prev, min_vram_gb: e.target.value ? parseInt(e.target.value) : undefined}))}
                      placeholder="Any"
                    />
                    <label>Max Price (USD/hr):</label>
                    <input 
                      type="number" 
                      value={searchFilters.max_price_usd || ''} 
                      onChange={(e) => setSearchFilters(prev => ({...prev, max_price_usd: e.target.value ? parseFloat(e.target.value) : undefined}))}
                      placeholder="Any"
                      step="0.01"
                    />
                  </div>
                </div>

                {/* Available Listings */}
                <div className="marketplace-listings">
                  <h3>Available GPU Rentals ({availableListings.length})</h3>
                  {availableListings.length > 0 ? (
                    <div className="listings-grid">
                      {availableListings.map(listing => (
                        <div key={listing.id} className="listing-card">
                          <h4>{listing.gpu_name} ({listing.gpu_model})</h4>
                          <div className="listing-specs">
                            <p>VRAM: {listing.vram_gb}GB</p>
                            <p>Cores: {listing.compute_units}</p>
                            <p>Performance: {listing.performance_score.toFixed(1)}/100</p>
                            <p>Location: {listing.location}</p>
                          </div>
                          <div className="listing-pricing">
                            <p className="price-usd">{formatCurrency(listing.hourly_rate_usd, 'usd')}/hr</p>
                            <p className="price-dgpu">{formatCurrency(listing.hourly_rate_dgpu, 'dgpu')}/hr</p>
                          </div>
                          <div className="listing-features">
                            {listing.jupyter_notebook && <span className="feature-tag">Jupyter</span>}
                            {listing.tensorboard && <span className="feature-tag">TensorBoard</span>}
                            {listing.ssh_access && <span className="feature-tag">SSH</span>}
                            {listing.container_support && <span className="feature-tag">Docker</span>}
                          </div>
                          <div className="listing-rating">
                            <span>Rating: {listing.rating.toFixed(1)}/5.0 ({listing.total_reviews} reviews)</span>
                          </div>
                          <div className="listing-actions">
                            <button onClick={() => handleOpenListingModal(listing)} className="secondary">
                              View Details
                            </button>
                            <button onClick={() => handleOpenBookingModal(listing)} className="primary">
                              Book Now
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p>No GPU rentals available. Create a listing to get started.</p>
                  )}
                </div>
              </>
            ) : (
              <p>Daemon is offline. Start the daemon to access the marketplace.</p>
            )}
          </section>
        </>
      )}

      {activeView === 'provider-dashboard' && (
        <>
          {/* Provider Dashboard */}
          <section className="card">
            <h2>Provider Dashboard</h2>
            {daemonActive && providerEarnings ? (
              <>
                {/* Provider Stats */}
                <div className="provider-stats">
                  <div className="stats-row">
                    <div className="stat-card">
                      <h4>Current Balance</h4>
                      <p className="stat-value">{formatCurrency(providerEarnings.current_balance_usd, 'usd')}</p>
                      <p className="stat-subtitle">{formatCurrency(providerEarnings.current_balance_dgpu, 'dgpu')}</p>
                    </div>
                    <div className="stat-card">
                      <h4>Pending Earnings</h4>
                      <p className="stat-value">{formatCurrency(providerEarnings.pending_earnings_usd, 'usd')}</p>
                      <p className="stat-subtitle">{formatCurrency(providerEarnings.pending_earnings_dgpu, 'dgpu')}</p>
                    </div>
                    <div className="stat-card">
                      <h4>Lifetime Earnings</h4>
                      <p className="stat-value">{formatCurrency(providerEarnings.total_lifetime_earnings_usd, 'usd')}</p>
                      <p className="stat-subtitle">{formatCurrency(providerEarnings.total_lifetime_earnings_dgpu, 'dgpu')}</p>
                    </div>
                  </div>
                  <div className="stats-row">
                    <div className="stat-card">
                      <h4>Total Rental Hours</h4>
                      <p className="stat-value">{providerEarnings.total_rental_hours.toLocaleString()}</p>
                      <p className="stat-subtitle">GPU time provided</p>
                    </div>
                    <div className="stat-card">
                      <h4>Average Hourly Rate</h4>
                      <p className="stat-value">{formatCurrency(providerEarnings.average_hourly_rate_usd, 'usd')}</p>
                      <p className="stat-subtitle">{formatCurrency(providerEarnings.average_hourly_rate_dgpu, 'dgpu')}</p>
                    </div>
                    <div className="stat-card">
                      <h4>Response Time</h4>
                      <p className="stat-value">{providerEarnings.response_time_minutes} min</p>
                      <p className="stat-subtitle">Average response</p>
                    </div>
                  </div>
                </div>

                {/* Performance Metrics */}
                <div className="performance-metrics">
                  <h3>Performance Metrics</h3>
                  <div className="metrics-grid">
                    <div className="metric-item">
                      <span>GPU Utilization:</span>
                      <span>{(providerEarnings.performance_metrics.gpu_utilization_average * 100).toFixed(1)}%</span>
                    </div>
                    <div className="metric-item">
                      <span>Uptime:</span>
                      <span>{(providerEarnings.performance_metrics.uptime_percentage * 100).toFixed(1)}%</span>
                    </div>
                    <div className="metric-item">
                      <span>Job Success Rate:</span>
                      <span>{(providerEarnings.performance_metrics.job_success_rate * 100).toFixed(1)}%</span>
                    </div>
                    <div className="metric-item">
                      <span>Customer Satisfaction:</span>
                      <span>{providerEarnings.performance_metrics.customer_satisfaction.toFixed(1)}/5.0</span>
                    </div>
                    <div className="metric-item">
                      <span>Repeat Customer Rate:</span>
                      <span>{(providerEarnings.performance_metrics.repeat_customer_rate * 100).toFixed(1)}%</span>
                    </div>
                    <div className="metric-item">
                      <span>Revenue Growth:</span>
                      <span>{(providerEarnings.performance_metrics.revenue_growth_rate * 100).toFixed(1)}%</span>
                    </div>
                  </div>
                </div>

                {/* Payout Information */}
                <div className="payout-info">
                  <h3>Payout Information</h3>
                  <p>Schedule: {providerEarnings.payout_schedule}</p>
                  <p>Next Payout: {providerEarnings.next_payout_date}</p>
                  <p>Method: {providerEarnings.payout_method}</p>
                  <p>Tax Verification: {providerEarnings.tax_information.verification_status}</p>
                </div>
              </>
            ) : (
              <p>Daemon is offline or earnings data not available.</p>
            )}
          </section>
        </>
      )}

      {activeView === 'bookings' && (
        <>
          {/* Bookings Management */}
          <section className="card">
            <h2>Bookings Management</h2>
            {daemonActive ? (
              <>
                {/* Active Bookings */}
                <div className="bookings-section">
                  <h3>Active Bookings ({activeBookings.length})</h3>
                  {activeBookings.length > 0 ? (
                    <div className="bookings-list">
                      {activeBookings.map(booking => (
                        <div key={booking.id} className="booking-card">
                          <div className="booking-header">
                            <h4>Booking #{booking.id.slice(-8)}</h4>
                            <span 
                              className="booking-status"
                              style={{backgroundColor: getBookingStatusColor(booking.booking_status)}}
                            >
                              {booking.booking_status.toUpperCase()}
                            </span>
                          </div>
                          <div className="booking-details">
                            <p>Renter: {booking.renter_name}</p>
                            <p>GPU: {booking.gpu_id}</p>
                            <p>Duration: {formatDuration(booking.duration_hours)}</p>
                            <p>Rate: {formatCurrency(booking.hourly_rate_usd, 'usd')}/hr</p>
                            <p>Total: {formatCurrency(booking.total_cost_usd, 'usd')}</p>
                            <p>Payment: <span style={{color: getPaymentStatusColor(booking.payment_status)}}>{booking.payment_status}</span></p>
                          </div>
                          <div className="booking-actions">
                            <button onClick={() => handleOpenJobModal(booking)} className="secondary">
                              View Details
                            </button>
                            {booking.booking_status === 'confirmed' && (
                              <button onClick={() => handleStartRentalJob(booking.id)} className="primary">
                                Start Job
                              </button>
                            )}
                            {booking.booking_status === 'active' && (
                              <button onClick={() => handleCompleteRentalJob(booking.id)} className="success">
                                Complete Job
                              </button>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p>No active bookings.</p>
                  )}
                </div>

                {/* Booking History */}
                <div className="bookings-section">
                  <h3>Recent Booking History ({bookingHistory.length})</h3>
                  {bookingHistory.length > 0 ? (
                    <div className="history-list">
                      {bookingHistory.slice(0, 10).map(booking => (
                        <div key={booking.id} className="history-item">
                          <div className="history-summary">
                            <span>#{booking.id.slice(-8)}</span>
                            <span>{booking.renter_name}</span>
                            <span>{formatDuration(booking.duration_hours)}</span>
                            <span>{formatCurrency(booking.total_cost_usd, 'usd')}</span>
                            <span 
                              style={{color: getBookingStatusColor(booking.booking_status)}}
                            >
                              {booking.booking_status}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p>No booking history available.</p>
                  )}
                </div>
              </>
            ) : (
              <p>Daemon is offline. Start the daemon to view bookings.</p>
            )}
          </section>
        </>
      )}

      {activeView === 'earnings' && (
        <>
          {/* Detailed Earnings */}
          <section className="card">
            <h2>Detailed Earnings & Analytics</h2>
            {daemonActive && providerEarnings ? (
              <>
                {/* Earnings Breakdown */}
                <div className="earnings-breakdown">
                  <h3>Earnings Breakdown</h3>
                  <div className="earnings-grid">
                    <div className="earnings-period">
                      <h4>Today</h4>
                      <p className="earning-amount">{formatCurrency(providerEarnings.earnings_today_usd, 'usd')}</p>
                      <p className="earning-tokens">{formatCurrency(providerEarnings.earnings_today_dgpu, 'dgpu')}</p>
                    </div>
                    <div className="earnings-period">
                      <h4>This Week</h4>
                      <p className="earning-amount">{formatCurrency(providerEarnings.earnings_this_week_usd, 'usd')}</p>
                      <p className="earning-tokens">{formatCurrency(providerEarnings.earnings_this_week_dgpu, 'dgpu')}</p>
                    </div>
                    <div className="earnings-period">
                      <h4>This Month</h4>
                      <p className="earning-amount">{formatCurrency(providerEarnings.earnings_this_month_usd, 'usd')}</p>
                      <p className="earning-tokens">{formatCurrency(providerEarnings.earnings_this_month_dgpu, 'dgpu')}</p>
                    </div>
                    <div className="earnings-period">
                      <h4>Lifetime</h4>
                      <p className="earning-amount">{formatCurrency(providerEarnings.total_lifetime_earnings_usd, 'usd')}</p>
                      <p className="earning-tokens">{formatCurrency(providerEarnings.total_lifetime_earnings_dgpu, 'dgpu')}</p>
                    </div>
                  </div>
                </div>

                {/* Balance Information */}
                <div className="balance-info">
                  <h3>Balance Information</h3>
                  <div className="balance-cards">
                    <div className="balance-card available">
                      <h4>Available Balance</h4>
                      <p className="balance-amount">{formatCurrency(providerEarnings.current_balance_usd, 'usd')}</p>
                      <p className="balance-tokens">{formatCurrency(providerEarnings.current_balance_dgpu, 'dgpu')}</p>
                      <p className="balance-note">Ready for withdrawal</p>
                    </div>
                    <div className="balance-card pending">
                      <h4>Pending Balance</h4>
                      <p className="balance-amount">{formatCurrency(providerEarnings.pending_earnings_usd, 'usd')}</p>
                      <p className="balance-tokens">{formatCurrency(providerEarnings.pending_earnings_dgpu, 'dgpu')}</p>
                      <p className="balance-note">From active bookings</p>
                    </div>
                  </div>
                </div>

                {/* Provider Performance */}
                <div className="provider-performance">
                  <h3>Performance Analytics</h3>
                  <div className="performance-charts">
                    <div className="chart-placeholder">
                      <h4>Revenue Trend</h4>
                      <p>Chart showing daily/weekly revenue would go here</p>
                    </div>
                    <div className="chart-placeholder">
                      <h4>GPU Utilization</h4>
                      <p>Chart showing GPU usage over time would go here</p>
                    </div>
                  </div>
                </div>
              </>
            ) : (
              <p>Daemon is offline or earnings data not available.</p>
            )}
          </section>
        </>
      )}

      {activeView === 'job-submission' && (
        <>
          {/* Job Submission Interface */}
          <section className="card">
            <h2>Job Submission Interface</h2>
            {daemonActive ? (
              <>
                <div className="job-submission-form">
                  <h3>Submit New GPU Job</h3>
                  <div className="form-section">
                    <h4>Job Specification</h4>
                    <div className="form-row">
                      <label>Job Name:</label>
                      <input 
                        type="text" 
                        value={newJobSpec.job_name}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, job_name: e.target.value}))}
                        placeholder="My ML Training Job"
                      />
                    </div>
                    <div className="form-row">
                      <label>Description:</label>
                      <textarea 
                        value={newJobSpec.job_description}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, job_description: e.target.value}))}
                        placeholder="Describe your machine learning job..."
                        rows={3}
                      />
                    </div>
                    <div className="form-row">
                      <label>Framework:</label>
                      <select 
                        value={newJobSpec.framework}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, framework: e.target.value}))}
                      >
                        <option value="pytorch">PyTorch</option>
                        <option value="tensorflow">TensorFlow</option>
                        <option value="keras">Keras</option>
                        <option value="scikit-learn">Scikit-Learn</option>
                        <option value="custom">Custom</option>
                      </select>
                      <label>Python Version:</label>
                      <select 
                        value={newJobSpec.python_version}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, python_version: e.target.value}))}
                      >
                        <option value="3.9">Python 3.9</option>
                        <option value="3.10">Python 3.10</option>
                        <option value="3.11">Python 3.11</option>
                      </select>
                    </div>
                  </div>

                  <div className="form-section">
                    <h4>Resource Requirements</h4>
                    <div className="form-row">
                      <label>GPU Memory (MB):</label>
                      <input 
                        type="number" 
                        value={newJobSpec.gpu_memory_requirements}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, gpu_memory_requirements: parseInt(e.target.value)}))}
                        min="1024"
                        max="36864"
                      />
                      <label>CPU Cores:</label>
                      <input 
                        type="number" 
                        value={newJobSpec.cpu_requirements}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, cpu_requirements: parseInt(e.target.value)}))}
                        min="1"
                        max="16"
                      />
                    </div>
                    <div className="form-row">
                      <label>RAM (MB):</label>
                      <input 
                        type="number" 
                        value={newJobSpec.ram_requirements}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, ram_requirements: parseInt(e.target.value)}))}
                        min="1024"
                        max="65536"
                      />
                      <label>Storage (GB):</label>
                      <input 
                        type="number" 
                        value={newJobSpec.storage_requirements}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, storage_requirements: parseInt(e.target.value)}))}
                        min="10"
                        max="1000"
                      />
                    </div>
                  </div>

                  <div className="form-section">
                    <h4>Job Configuration</h4>
                    <div className="form-row">
                      <label>Priority:</label>
                      <select 
                        value={newJobSpec.priority}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, priority: e.target.value}))}
                      >
                        <option value="low">Low</option>
                        <option value="normal">Normal</option>
                        <option value="high">High</option>
                      </select>
                      <label>Max Retries:</label>
                      <input 
                        type="number" 
                        value={newJobSpec.max_retries}
                        onChange={(e) => setNewJobSpec(prev => ({...prev, max_retries: parseInt(e.target.value)}))}
                        min="0"
                        max="10"
                      />
                    </div>
                    <div className="form-row">
                      <label>
                        <input 
                          type="checkbox" 
                          checked={newJobSpec.auto_restart_on_failure}
                          onChange={(e) => setNewJobSpec(prev => ({...prev, auto_restart_on_failure: e.target.checked}))}
                        />
                        Auto-restart on failure
                      </label>
                      <label>
                        <input 
                          type="checkbox" 
                          checked={newJobSpec.network_requirements}
                          onChange={(e) => setNewJobSpec(prev => ({...prev, network_requirements: e.target.checked}))}
                        />
                        Requires internet access
                      </label>
                    </div>
                  </div>

                  <div className="form-actions">
                    <button className="primary" onClick={() => setShowJobModal(true)}>
                      Submit Job
                    </button>
                    <button className="secondary" onClick={() => {
                      setNewJobSpec({
                        job_name: '',
                        job_description: '',
                        framework: 'pytorch',
                        python_version: '3.9',
                        conda_environment: undefined,
                        pip_requirements: [],
                        environment_variables: {},
                        startup_script: undefined,
                        data_sources: [],
                        expected_outputs: [],
                        estimated_completion_time: undefined,
                        gpu_memory_requirements: 8192,
                        cpu_requirements: 4,
                        ram_requirements: 16384,
                        storage_requirements: 50,
                        network_requirements: true,
                        checkpoint_frequency: 300,
                        auto_restart_on_failure: true,
                        max_retries: 3,
                        priority: 'normal',
                      });
                    }}>
                      Reset Form
                    </button>
                  </div>
                </div>
              </>
            ) : (
              <p>Daemon is offline. Start the daemon to submit jobs.</p>
            )}
          </section>
        </>
      )}

      {/* Original sections remain available in overview */}
      {activeView === 'overview' && (
        <>
          {/* Local Job Monitoring */}
          <section className="card">
            <h2>Local Job Monitoring</h2>
            {daemonActive ? (
              localJobs.length > 0 ? (
                 <div className="job-list">
                  {localJobs.map(job => (
                    <div key={job.id} className="job-item card">
                      <h4>{job.name} (ID: {job.id})</h4>
                      <p>Status: <span className={`job-status-${job.status}`}>{job.status}</span></p>
                      <p>Progress: {job.progress_percent}%</p>
                      {/* Basic progress bar */}
                      <div style={{ width: '100%', backgroundColor: '#eee' }}>
                        <div style={{ width: `${job.progress_percent}%`, backgroundColor: 'green', height: '10px' }}></div>
                      </div>
                      <p>Started: {new Date(job.started_at).toLocaleString()}</p>
                      {job.estimated_time_remaining_seconds && <p>ETA: {job.estimated_time_remaining_seconds}s</p>}
                    </div>
                  ))}
                </div>
              ) : (
                <p>No local jobs active or data not yet loaded.</p>
              )
            ) : (
              <p>Daemon is offline. Start the daemon to see local jobs.</p>
            )}
          </section>

          {/* System & Financial Overview */}
          <section className="card">
            <h2>System & Financial Overview</h2>
            {daemonActive ? (
              <>
                <div className="overview-item">
                  <h4>Network Status</h4>
                  {networkStatus ? (
                    <>
                      <p>Connection: {networkStatus.connection_status}</p>
                      {networkStatus.ip_address && <p>IP: {networkStatus.ip_address}</p>}
                    </>
                  ) : <p>Loading network info...</p>}
                </div>
                <div className="overview-item">
                  <h4>Financial Summary</h4>
                  {financialSummary ? (
                    <>
                      <p>Wallet Balance: {financialSummary.wallet_balance_dgpu.toFixed(2)} dGPU</p>
                      <p>Total Earned: {financialSummary.total_earned_dgpu.toFixed(2)} dGPU</p>
                      <p>Pending Payout: {financialSummary.pending_payout_dgpu.toFixed(2)} dGPU</p>
                      {financialSummary.last_payout_at && <p>Last Payout: {new Date(financialSummary.last_payout_at).toLocaleString()}</p>}
                    </>
                  ) : <p>Loading financial summary...</p>}
                </div>
              </>
            ) : (
              <p>Daemon is offline. Start the daemon to see system and financial overview.</p>
            )}
          </section>

          {/* Provider Settings */}
          <section className="card">
            <h2>Provider Settings</h2>
            {daemonActive ? (
              providerSettings ? (
                <div>
                  <div>
                    <label htmlFor="defaultRate">Default Hourly Rate (dGPU): </label>
                    <input 
                      type="number" 
                      id="defaultRate" 
                      value={providerSettings.default_hourly_rate_dgpu}
                      onChange={(e) => setProviderSettings({...providerSettings, default_hourly_rate_dgpu: parseFloat(e.target.value)})}
                      min="0"
                      step="0.01"
                    />
                  </div>
                  <button onClick={handleSaveProviderSettings}>Save Settings</button>
                </div>
              ) : (
                <p>Loading provider settings...</p>
              )
            ) : (
              <p>Daemon is offline. Start the daemon to configure settings.</p>
            )}
          </section>
        </>
      )}



      {/* --- BEGIN RENTAL MODAL --- */}
      {gpuRentalModalOpen && selectedGpu && (
        <div className="modal-backdrop">
          <div className="modal-content card">
            <h3>Set Rental Rate for {selectedGpu.name}</h3>
            <p>Current Status: {selectedGpu.is_available_for_rent ? "Rentable" : "Private"}</p>
            <div>
              <label htmlFor="rentalRate">New Hourly Rate (dGPU): </label>
              <input 
                type="number" 
                id="rentalRate" 
                value={newRentalRate}
                onChange={(e) => setNewRentalRate(e.target.value)}
                min="0"
                step="0.01"
              />
            </div>
            <div className="modal-actions">
              <button onClick={handleUpdateGpuRental}>
                {selectedGpu.is_available_for_rent ? 'Update & Keep Rentable' : 'Set Rate & Make Rentable'}
              </button>
              {selectedGpu.is_available_for_rent && (
                 <button onClick={async () => {
                    if (!selectedGpu) return;
                    try {
                      await invoke('update_gpu_rental_settings', { 
                        gpuId: selectedGpu.id, 
                        isAvailable: false, 
                        hourlyRate: selectedGpu.current_hourly_rate_dgpu 
                      });
                      addLog('status', `GPU ${selectedGpu.name} set to Private.`);
                      const updatedGpus = await invoke<GpuInfo[]>('get_detected_gpus');
                      setGpus(updatedGpus);
                    } catch (err) {
                      addLog('error', `Failed to set GPU ${selectedGpu.name} to private: ${err}`);
                    }
                    handleCloseGpuRentalModal();
                  }}>
                  Make Private
                </button>
              )}
              <button onClick={handleCloseGpuRentalModal}>Cancel</button>
            </div>
          </div>
        </div>
      )}
      {/* --- END RENTAL MODAL --- */}
      
      {/* --- BEGIN LOCAL JOB MONITORING SECTION --- */}
      <section className="card">
        <h2>Local Job Monitoring</h2>
        {daemonActive ? (
          localJobs.length > 0 ? (
             <div className="job-list">
              {localJobs.map(job => (
                <div key={job.id} className="job-item card">
                  <h4>{job.name} (ID: {job.id})</h4>
                  <p>Status: <span className={`job-status-${job.status}`}>{job.status}</span></p>
                  <p>Progress: {job.progress_percent}%</p>
                  {/* Basic progress bar */}
                  <div style={{ width: '100%', backgroundColor: '#eee' }}>
                    <div style={{ width: `${job.progress_percent}%`, backgroundColor: 'green', height: '10px' }}></div>
                  </div>
                  <p>Started: {new Date(job.started_at).toLocaleString()}</p>
                  {job.estimated_time_remaining_seconds && <p>ETA: {job.estimated_time_remaining_seconds}s</p>}
                </div>
              ))}
            </div>
          ) : (
            <p>No local jobs active or data not yet loaded.</p>
          )
        ) : (
          <p>Daemon is offline. Start the daemon to see local jobs.</p>
        )}
      </section>
      {/* --- END LOCAL JOB MONITORING SECTION --- */}

      {/* --- BEGIN SYSTEM & FINANCIAL OVERVIEW SECTION --- */}
      <section className="card">
        <h2>System & Financial Overview</h2>
        {daemonActive ? (
          <>
            <div className="overview-item">
              <h4>Network Status</h4>
              {networkStatus ? (
                <>
                  <p>Connection: {networkStatus.connection_status}</p>
                  {networkStatus.ip_address && <p>IP: {networkStatus.ip_address}</p>}
                  {/* Add more network details if available */}
                </>
              ) : <p>Loading network info...</p>}
            </div>
            <div className="overview-item">
              <h4>Financial Summary</h4>
              {financialSummary ? (
                <>
                  <p>Wallet Balance: {financialSummary.wallet_balance_dgpu.toFixed(2)} dGPU</p>
                  <p>Total Earned: {financialSummary.total_earned_dgpu.toFixed(2)} dGPU</p>
                  <p>Pending Payout: {financialSummary.pending_payout_dgpu.toFixed(2)} dGPU</p>
                  {financialSummary.last_payout_at && <p>Last Payout: {new Date(financialSummary.last_payout_at).toLocaleString()}</p>}
                </>
              ) : <p>Loading financial summary...</p>}
            </div>
          </>
        ) : (
          <p>Daemon is offline. Start the daemon to see system and financial overview.</p>
        )}
      </section>
      {/* --- END SYSTEM & FINANCIAL OVERVIEW SECTION --- */}

      {/* --- BEGIN PROVIDER SETTINGS SECTION --- */}
      <section className="card">
        <h2>Provider Settings</h2>
        {daemonActive ? (
          providerSettings ? (
            <div>
              <div>
                <label htmlFor="defaultRate">Default Hourly Rate (dGPU): </label>
                <input 
                  type="number" 
                  id="defaultRate" 
                  value={providerSettings.default_hourly_rate_dgpu}
                  onChange={(e) => setProviderSettings({...providerSettings, default_hourly_rate_dgpu: parseFloat(e.target.value)})}
                  min="0"
                  step="0.01"
                />
              </div>
              {/* Add more settings inputs here based on ProviderSettings interface */}
              <button onClick={handleSaveProviderSettings}>Save Settings</button>
            </div>
          ) : (
            <p>Loading provider settings...</p>
          )
        ) : (
          <p>Daemon is offline. Start the daemon to configure settings.</p>
        )}
      </section>
      {/* --- END PROVIDER SETTINGS SECTION --- */}

      {/* === COMPREHENSIVE MODAL SYSTEM === */}
      
      {/* Listing Details Modal */}
      {showListingModal && selectedListing && (
        <div className="modal-backdrop">
          <div className="modal-content card listing-modal">
            <h3>GPU Rental Details</h3>
            <div className="listing-details-comprehensive">
              <div className="detail-section">
                <h4>GPU Specifications</h4>
                <div className="detail-grid">
                  <div className="detail-item">
                    <span className="detail-label">Model:</span>
                    <span className="detail-value">{selectedListing.gpu_model}</span>
                  </div>
                  <div className="detail-item">
                    <span className="detail-label">Architecture:</span>
                    <span className="detail-value">{selectedListing.gpu_architecture}</span>
                  </div>
                  <div className="detail-item">
                    <span className="detail-label">VRAM:</span>
                    <span className="detail-value">{selectedListing.vram_gb}GB</span>
                  </div>
                  <div className="detail-item">
                    <span className="detail-label">Compute Units:</span>
                    <span className="detail-value">{selectedListing.compute_units}</span>
                  </div>
                  <div className="detail-item">
                    <span className="detail-label">Base Clock:</span>
                    <span className="detail-value">{selectedListing.base_clock_mhz}MHz</span>
                  </div>
                  <div className="detail-item">
                    <span className="detail-label">Memory Clock:</span>
                    <span className="detail-value">{selectedListing.memory_clock_mhz}MHz</span>
                  </div>
                  <div className="detail-item">
                    <span className="detail-label">Performance Score:</span>
                    <span className="detail-value">{selectedListing.performance_score}/100</span>
                  </div>
                </div>
              </div>
              
              <div className="detail-section">
                <h4>Pricing & Availability</h4>
                <div className="pricing-details">
                  <div className="price-item">
                    <span className="price-label">Hourly Rate (USD):</span>
                    <span className="price-value">{formatCurrency(selectedListing.hourly_rate_usd, 'usd')}</span>
                  </div>
                  <div className="price-item">
                    <span className="price-label">Hourly Rate (DGPU):</span>
                    <span className="price-value">{formatCurrency(selectedListing.hourly_rate_dgpu, 'dgpu')}</span>
                  </div>
                  <div className="price-item">
                    <span className="price-label">Min Duration:</span>
                    <span className="price-value">{selectedListing.minimum_rental_hours}h</span>
                  </div>
                  <div className="price-item">
                    <span className="price-label">Max Duration:</span>
                    <span className="price-value">{selectedListing.maximum_rental_hours}h</span>
                  </div>
                </div>
              </div>
              
              <div className="detail-section">
                <h4>Provider Information</h4>
                <div className="provider-details">
                  <p><strong>Provider:</strong> {selectedListing.provider_name}</p>
                  <p><strong>Location:</strong> {selectedListing.location}</p>
                  <p><strong>Rating:</strong> {selectedListing.rating.toFixed(1)}/5.0 ({selectedListing.total_reviews} reviews)</p>
                  <p><strong>Response Time:</strong> {selectedListing.provider_response_time_minutes} minutes</p>
                  <p><strong>Total Rental Hours:</strong> {selectedListing.total_rental_hours.toLocaleString()}</p>
                </div>
              </div>
              
              <div className="detail-section">
                <h4>Supported Features</h4>
                <div className="features-grid">
                  {selectedListing.supported_frameworks.map(framework => (
                    <span key={framework} className="feature-chip">{framework}</span>
                  ))}
                </div>
                <div className="features-list">
                  {selectedListing.jupyter_notebook && <span className="feature-enabled">Jupyter Notebook</span>}
                  {selectedListing.tensorboard && <span className="feature-enabled">TensorBoard</span>}
                  {selectedListing.ssh_access && <span className="feature-enabled">SSH Access</span>}
                  {selectedListing.container_support && <span className="feature-enabled">Docker Support</span>}
                  {selectedListing.custom_docker_images && <span className="feature-enabled">Custom Images</span>}
                  {selectedListing.data_persistence && <span className="feature-enabled">Data Persistence</span>}
                  {selectedListing.internet_access && <span className="feature-enabled">Internet Access</span>}
                </div>
              </div>
              
              <div className="detail-section">
                <h4>Special Offers</h4>
                <div className="offers-list">
                  {selectedListing.special_offers.map((offer, index) => (
                    <div key={index} className="offer-item">{offer}</div>
                  ))}
                </div>
              </div>
            </div>
            
            <div className="modal-actions">
              <button onClick={() => handleOpenBookingModal(selectedListing)} className="primary">
                Book This GPU
              </button>
              <button onClick={handleCloseListingModal} className="secondary">
                Close
              </button>
            </div>
          </div>
        </div>
      )}
      
      {/* Booking Creation Modal */}
      {showBookingModal && selectedListing && (
        <div className="modal-backdrop">
          <div className="modal-content card booking-modal">
            <h3>Create GPU Rental Booking</h3>
            <div className="booking-form">
              <div className="form-section">
                <h4>Booking Details</h4>
                <div className="form-row">
                  <label>Your Name:</label>
                  <input 
                    type="text" 
                    value={bookingForm.renter_name}
                    onChange={(e) => setBookingForm(prev => ({...prev, renter_name: e.target.value}))}
                    placeholder="Enter your name"
                  />
                </div>
                <div className="form-row">
                  <label>Duration (hours):</label>
                  <input 
                    type="number" 
                    value={bookingForm.duration_hours}
                    onChange={(e) => setBookingForm(prev => ({...prev, duration_hours: parseInt(e.target.value)}))}
                    min={selectedListing.minimum_rental_hours}
                    max={selectedListing.maximum_rental_hours}
                  />
                </div>
                <div className="form-row">
                  <label>Booking Type:</label>
                  <select 
                    value={bookingForm.booking_type}
                    onChange={(e) => setBookingForm(prev => ({...prev, booking_type: e.target.value}))}
                  >
                    <option value="immediate">Start Immediately</option>
                    <option value="scheduled">Schedule for Later</option>
                  </select>
                </div>
              </div>
              
              <div className="form-section">
                <h4>Cost Breakdown</h4>
                <div className="cost-breakdown">
                  <div className="cost-item">
                    <span>Hourly Rate (USD):</span>
                    <span>{formatCurrency(selectedListing.hourly_rate_usd, 'usd')}</span>
                  </div>
                  <div className="cost-item">
                    <span>Hourly Rate (DGPU):</span>
                    <span>{formatCurrency(selectedListing.hourly_rate_dgpu, 'dgpu')}</span>
                  </div>
                  <div className="cost-item">
                    <span>Duration:</span>
                    <span>{bookingForm.duration_hours} hours</span>
                  </div>
                  <div className="cost-item total">
                    <span>Total Cost (USD):</span>
                    <span>{formatCurrency(selectedListing.hourly_rate_usd * bookingForm.duration_hours, 'usd')}</span>
                  </div>
                  <div className="cost-item total">
                    <span>Total Cost (DGPU):</span>
                    <span>{formatCurrency(selectedListing.hourly_rate_dgpu * bookingForm.duration_hours, 'dgpu')}</span>
                  </div>
                </div>
              </div>
              
              <div className="form-section">
                <h4>GPU Specifications</h4>
                <div className="gpu-specs-summary">
                  <p><strong>Model:</strong> {selectedListing.gpu_model}</p>
                  <p><strong>VRAM:</strong> {selectedListing.vram_gb}GB</p>
                  <p><strong>Performance:</strong> {selectedListing.performance_score}/100</p>
                </div>
              </div>
            </div>
            
            <div className="modal-actions">
              <button 
                onClick={() => handleCreateBooking(selectedListing.id)} 
                className="primary"
                disabled={!bookingForm.renter_name || bookingForm.duration_hours < selectedListing.minimum_rental_hours}
              >
                Create Booking
              </button>
              <button onClick={handleCloseBookingModal} className="secondary">
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
      
      {/* Job Details Modal */}
      {showJobModal && selectedBooking && (
        <div className="modal-backdrop">
          <div className="modal-content card job-modal">
            <h3>Booking Details & Job Management</h3>
            <div className="job-details-comprehensive">
              <div className="detail-section">
                <h4>Booking Information</h4>
                <div className="booking-info">
                  <p><strong>Booking ID:</strong> {selectedBooking.id}</p>
                  <p><strong>Renter:</strong> {selectedBooking.renter_name}</p>
                  <p><strong>Status:</strong> <span style={{color: getBookingStatusColor(selectedBooking.booking_status)}}>{selectedBooking.booking_status}</span></p>
                  <p><strong>Payment:</strong> <span style={{color: getPaymentStatusColor(selectedBooking.payment_status)}}>{selectedBooking.payment_status}</span></p>
                  <p><strong>Duration:</strong> {formatDuration(selectedBooking.duration_hours)}</p>
                  <p><strong>Total Cost:</strong> {formatCurrency(selectedBooking.total_cost_usd, 'usd')} / {formatCurrency(selectedBooking.total_cost_dgpu, 'dgpu')}</p>
                </div>
              </div>
              
              <div className="detail-section">
                <h4>Job Specifications</h4>
                <div className="job-specs">
                  <p><strong>Name:</strong> {selectedBooking.job_specifications.job_name}</p>
                  <p><strong>Framework:</strong> {selectedBooking.job_specifications.framework}</p>
                  <p><strong>Python Version:</strong> {selectedBooking.job_specifications.python_version}</p>
                  <p><strong>Priority:</strong> {selectedBooking.job_specifications.priority}</p>
                  <p><strong>GPU Memory:</strong> {selectedBooking.job_specifications.gpu_memory_requirements}MB</p>
                  <p><strong>CPU Cores:</strong> {selectedBooking.job_specifications.cpu_requirements}</p>
                  <p><strong>RAM:</strong> {selectedBooking.job_specifications.ram_requirements}MB</p>
                  <p><strong>Storage:</strong> {selectedBooking.job_specifications.storage_requirements}GB</p>
                </div>
              </div>
              
              <div className="detail-section">
                <h4>Resource Allocation</h4>
                <div className="resource-allocation">
                  <div className="resource-item">
                    <span>GPU Memory:</span>
                    <span>{selectedBooking.resource_allocation.allocated_gpu_memory_mb}MB</span>
                  </div>
                  <div className="resource-item">
                    <span>CPU Cores:</span>
                    <span>{selectedBooking.resource_allocation.allocated_cpu_cores}</span>
                  </div>
                  <div className="resource-item">
                    <span>RAM:</span>
                    <span>{selectedBooking.resource_allocation.allocated_ram_mb}MB</span>
                  </div>
                  <div className="resource-item">
                    <span>Storage:</span>
                    <span>{selectedBooking.resource_allocation.allocated_storage_gb}GB</span>
                  </div>
                </div>
              </div>
              
              {selectedBooking.ssh_connection_info && (
                <div className="detail-section">
                  <h4>Connection Information</h4>
                  <div className="connection-info">
                    <p><strong>Hostname:</strong> {selectedBooking.ssh_connection_info.hostname}</p>
                    <p><strong>Port:</strong> {selectedBooking.ssh_connection_info.port}</p>
                    <p><strong>Username:</strong> {selectedBooking.ssh_connection_info.username}</p>
                    <p><strong>Connection URL:</strong> {selectedBooking.ssh_connection_info.connection_url}</p>
                    {selectedBooking.ssh_connection_info.jupyter_url && (
                      <p><strong>Jupyter URL:</strong> <a href={selectedBooking.ssh_connection_info.jupyter_url} target="_blank" rel="noopener noreferrer">{selectedBooking.ssh_connection_info.jupyter_url}</a></p>
                    )}
                    {selectedBooking.ssh_connection_info.tensorboard_url && (
                      <p><strong>TensorBoard URL:</strong> <a href={selectedBooking.ssh_connection_info.tensorboard_url} target="_blank" rel="noopener noreferrer">{selectedBooking.ssh_connection_info.tensorboard_url}</a></p>
                    )}
                  </div>
                </div>
              )}
              
              <div className="detail-section">
                <h4>Timeline</h4>
                <div className="timeline">
                  <div className="timeline-item">
                    <span>Created:</span>
                    <span>{new Date(selectedBooking.created_at).toLocaleString()}</span>
                  </div>
                  {selectedBooking.confirmed_at && (
                    <div className="timeline-item">
                      <span>Confirmed:</span>
                      <span>{new Date(selectedBooking.confirmed_at).toLocaleString()}</span>
                    </div>
                  )}
                  {selectedBooking.started_at && (
                    <div className="timeline-item">
                      <span>Started:</span>
                      <span>{new Date(selectedBooking.started_at).toLocaleString()}</span>
                    </div>
                  )}
                  {selectedBooking.completed_at && (
                    <div className="timeline-item">
                      <span>Completed:</span>
                      <span>{new Date(selectedBooking.completed_at).toLocaleString()}</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
            
            <div className="modal-actions">
              {selectedBooking.booking_status === 'confirmed' && (
                <button onClick={() => handleStartRentalJob(selectedBooking.id)} className="primary">
                  Start Job
                </button>
              )}
              {selectedBooking.booking_status === 'active' && (
                <button onClick={() => handleCompleteRentalJob(selectedBooking.id)} className="success">
                  Complete Job
                </button>
              )}
              <button onClick={handleCloseJobModal} className="secondary">
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      <section className="logs-card">
        <h2 className="logs-header">Daemon Activity Logs</h2>
        <div className="logs-container">
          {daemonLogs.map((log) => (
            <div key={log.id} className={`log-entry ${getLogClass(log.log_type)}`}>
              <span className="log-icon">{getLogIcon(log.log_type)}</span>
              <span className="log-timestamp">
                {new Date(log.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
              </span>
              <span className="log-message">{log.message}</span>
            </div>
          ))}
          <div ref={logsEndRef} />
        </div>
      </section>
    </div>
  );
}

export default App; 