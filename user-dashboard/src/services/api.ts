import axios, { AxiosInstance, AxiosError } from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

class APIClient {
  private client: AxiosInstance;
  private accessToken: string | null = null;
  private refreshToken: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Load tokens from localStorage
    this.accessToken = localStorage.getItem('access_token');
    this.refreshToken = localStorage.getItem('refresh_token');

    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        if (this.accessToken) {
          config.headers.Authorization = `Bearer ${this.accessToken}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => response,
      async (error: AxiosError) => {
        const originalRequest = error.config as any;

        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true;

          try {
            const { data } = await this.client.post('/auth/refresh', {
              refresh_token: this.refreshToken,
            });

            this.setTokens(data.access_token, data.refresh_token);
            originalRequest.headers.Authorization = `Bearer ${data.access_token}`;

            return this.client(originalRequest);
          } catch (refreshError) {
            this.clearTokens();
            window.location.href = '/login';
            return Promise.reject(refreshError);
          }
        }

        return Promise.reject(error);
      }
    );
  }

  setTokens(accessToken: string, refreshToken: string) {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
  }

  clearTokens() {
    this.accessToken = null;
    this.refreshToken = null;
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
  }

  // Authentication
  async register(data: {
    email: string;
    password: string;
    first_name: string;
    last_name: string;
  }) {
    const response = await this.client.post('/auth/register', data);
    return response.data;
  }

  async login(email: string, password: string) {
    const response = await this.client.post('/auth/login', { email, password });
    if (response.data.access_token) {
      this.setTokens(response.data.access_token, response.data.refresh_token);
    }
    return response.data;
  }

  async logout() {
    await this.client.post('/auth/logout');
    this.clearTokens();
  }

  async verifyEmail(token: string) {
    const response = await this.client.post('/auth/verify-email', { token });
    return response.data;
  }

  async requestPasswordReset(email: string) {
    const response = await this.client.post('/auth/password-reset/request', { email });
    return response.data;
  }

  async resetPassword(token: string, newPassword: string) {
    const response = await this.client.post('/auth/password-reset/reset', {
      token,
      new_password: newPassword,
    });
    return response.data;
  }

  // Wallet
  async createWallet() {
    const response = await this.client.post('/wallet/create');
    return response.data;
  }

  async getWallet() {
    const response = await this.client.get('/wallet');
    return response.data;
  }

  async getTransactions() {
    const response = await this.client.get('/wallet/transactions');
    return response.data;
  }

  async withdraw(amount: number, address: string) {
    const response = await this.client.post('/wallet/withdraw', { amount, address });
    return response.data;
  }

  // GPUs
  async listGPUs(filters?: {
    min_memory?: number;
    max_rate?: number;
    gpu_model?: string;
  }) {
    const response = await this.client.get('/gpus', { params: filters });
    return response.data;
  }

  async getGPUDetails(id: string) {
    const response = await this.client.get(`/gpus/${id}`);
    return response.data;
  }

  // Jobs
  async submitJob(data: {
    name: string;
    description: string;
    docker_image: string;
    command: string[];
    environment: Record<string, string>;
    gpu_capability_id: string;
    max_duration_hours: number;
    dataset_urls?: string[];
  }) {
    const response = await this.client.post('/jobs', data);
    return response.data;
  }

  async listJobs(status?: string) {
    const response = await this.client.get('/jobs', { params: { status } });
    return response.data;
  }

  async getJob(id: string) {
    const response = await this.client.get(`/jobs/${id}`);
    return response.data;
  }

  async cancelJob(jobId: string) {
    const response = await this.client.post('/jobs/cancel', { job_id: jobId });
    return response.data;
  }

  async getJobLogs(id: string) {
    const response = await this.client.get(`/jobs/${id}/logs`);
    return response.data;
  }

  // Billing
  async startRental(data: {
    provider_id: string;
    gpu_capability_id: string;
    job_id: string;
    hourly_rate: number;
    estimated_minutes: number;
  }) {
    const response = await this.client.post('/billing/rental/start', data);
    return response.data;
  }

  async getBillingHistory() {
    const response = await this.client.get('/billing/history');
    return response.data;
  }

  // User Profile
  async getProfile() {
    const response = await this.client.get('/user/profile');
    return response.data;
  }

  async updateProfile(data: {
    first_name?: string;
    last_name?: string;
    phone?: string;
  }) {
    const response = await this.client.put('/user/profile', data);
    return response.data;
  }

  // Notifications
  async getNotifications() {
    const response = await this.client.get('/notifications');
    return response.data;
  }

  async markNotificationRead(id: string) {
    const response = await this.client.put(`/notifications/${id}/read`);
    return response.data;
  }

  // API Keys
  async createAPIKey(data: {
    name: string;
    description?: string;
    scopes: string[];
    expires_in?: number;
  }) {
    const response = await this.client.post('/api-keys', data);
    return response.data;
  }

  async listAPIKeys() {
    const response = await this.client.get('/api-keys');
    return response.data;
  }

  async revokeAPIKey(keyId: string) {
    const response = await this.client.post('/api-keys/revoke', { key_id: keyId });
    return response.data;
  }

  // 2FA
  async enable2FA(method: 'totp' | 'sms' | 'email', phone?: string) {
    const response = await this.client.post('/auth/2fa/enable', { method, phone });
    return response.data;
  }

  async verify2FASetup(code: string, secret: string) {
    const response = await this.client.post('/auth/2fa/verify-setup', { code, secret });
    return response.data;
  }

  async disable2FA(password: string, code: string) {
    const response = await this.client.post('/auth/2fa/disable', { password, code });
    return response.data;
  }
}

export const apiClient = new APIClient();
export default apiClient;

