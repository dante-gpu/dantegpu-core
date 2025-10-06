import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../services/api';
import { toast } from 'react-hot-toast';

// Wallet hooks
export function useWallet() {
  return useQuery({
    queryKey: ['wallet'],
    queryFn: () => apiClient.getWallet(),
    retry: 1,
  });
}

export function useCreateWallet() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: () => apiClient.createWallet(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wallet'] });
      toast.success('Wallet created successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create wallet');
    },
  });
}

export function useTransactions() {
  return useQuery({
    queryKey: ['transactions'],
    queryFn: () => apiClient.getTransactions(),
  });
}

export function useWithdraw() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ amount, address }: { amount: number; address: string }) =>
      apiClient.withdraw(amount, address),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wallet'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      toast.success('Withdrawal initiated successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Withdrawal failed');
    },
  });
}

// GPU hooks
export function useGPUs(filters?: {
  min_memory?: number;
  max_rate?: number;
  gpu_model?: string;
}) {
  return useQuery({
    queryKey: ['gpus', filters],
    queryFn: () => apiClient.listGPUs(filters),
    staleTime: 30000, // 30 seconds
  });
}

export function useGPUDetails(id: string) {
  return useQuery({
    queryKey: ['gpu', id],
    queryFn: () => apiClient.getGPUDetails(id),
    enabled: !!id,
  });
}

// Job hooks
export function useJobs(status?: string) {
  return useQuery({
    queryKey: ['jobs', status],
    queryFn: () => apiClient.listJobs(status),
    refetchInterval: 5000, // Refetch every 5 seconds for active jobs
  });
}

export function useJob(id: string) {
  return useQuery({
    queryKey: ['job', id],
    queryFn: () => apiClient.getJob(id),
    enabled: !!id,
    refetchInterval: (data) => {
      // Refetch every 2 seconds if job is running
      return data?.status === 'running' ? 2000 : false;
    },
  });
}

export function useJobLogs(id: string) {
  return useQuery({
    queryKey: ['job-logs', id],
    queryFn: () => apiClient.getJobLogs(id),
    enabled: !!id,
    refetchInterval: 3000, // Refetch every 3 seconds
  });
}

export function useSubmitJob() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: {
      name: string;
      description: string;
      docker_image: string;
      command: string[];
      environment: Record<string, string>;
      gpu_capability_id: string;
      max_duration_hours: number;
      dataset_urls?: string[];
    }) => apiClient.submitJob(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
      toast.success('Job submitted successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to submit job');
    },
  });
}

export function useCancelJob() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (jobId: string) => apiClient.cancelJob(jobId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
      toast.success('Job cancelled successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to cancel job');
    },
  });
}

// Billing hooks
export function useStartRental() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: {
      provider_id: string;
      gpu_capability_id: string;
      job_id: string;
      hourly_rate: number;
      estimated_minutes: number;
    }) => apiClient.startRental(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wallet'] });
      toast.success('Rental started successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to start rental');
    },
  });
}

export function useBillingHistory() {
  return useQuery({
    queryKey: ['billing-history'],
    queryFn: () => apiClient.getBillingHistory(),
  });
}

// User profile hooks
export function useProfile() {
  return useQuery({
    queryKey: ['profile'],
    queryFn: () => apiClient.getProfile(),
  });
}

export function useUpdateProfile() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: {
      first_name?: string;
      last_name?: string;
      phone?: string;
    }) => apiClient.updateProfile(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] });
      toast.success('Profile updated successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to update profile');
    },
  });
}

// Notification hooks
export function useNotifications() {
  return useQuery({
    queryKey: ['notifications'],
    queryFn: () => apiClient.getNotifications(),
    refetchInterval: 30000, // Refetch every 30 seconds
  });
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.markNotificationRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });
}

// API Key hooks
export function useAPIKeys() {
  return useQuery({
    queryKey: ['api-keys'],
    queryFn: () => apiClient.listAPIKeys(),
  });
}

export function useCreateAPIKey() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: {
      name: string;
      description?: string;
      scopes: string[];
      expires_in?: number;
    }) => apiClient.createAPIKey(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
      toast.success('API key created successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to create API key');
    },
  });
}

export function useRevokeAPIKey() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (keyId: string) => apiClient.revokeAPIKey(keyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
      toast.success('API key revoked successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to revoke API key');
    },
  });
}

// 2FA hooks
export function useEnable2FA() {
  return useMutation({
    mutationFn: ({ method, phone }: { method: 'totp' | 'sms' | 'email'; phone?: string }) =>
      apiClient.enable2FA(method, phone),
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to enable 2FA');
    },
  });
}

export function useVerify2FASetup() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ code, secret }: { code: string; secret: string }) =>
      apiClient.verify2FASetup(code, secret),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] });
      toast.success('2FA enabled successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to verify 2FA');
    },
  });
}

export function useDisable2FA() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ password, code }: { password: string; code: string }) =>
      apiClient.disable2FA(password, code),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile'] });
      toast.success('2FA disabled successfully');
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to disable 2FA');
    },
  });
}

