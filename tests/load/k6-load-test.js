import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const loginDuration = new Trend('login_duration');
const apiDuration = new Trend('api_duration');
const requestCount = new Counter('request_count');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 100 },   // Ramp up to 100 users
    { duration: '5m', target: 100 },   // Stay at 100 users
    { duration: '2m', target: 500 },   // Ramp up to 500 users
    { duration: '5m', target: 500 },   // Stay at 500 users
    { duration: '2m', target: 1000 },  // Ramp up to 1000 users
    { duration: '5m', target: 1000 },  // Stay at 1000 users
    { duration: '2m', target: 0 },     // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% < 500ms, 99% < 1s
    http_req_failed: ['rate<0.01'],                  // Error rate < 1%
    errors: ['rate<0.05'],                           // Custom error rate < 5%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

// Test data
const testUsers = [];
for (let i = 0; i < 1000; i++) {
  testUsers.push({
    email: `loadtest_${i}_${Date.now()}@example.com`,
    password: 'LoadTest123!',
    firstName: `Load${i}`,
    lastName: 'Test',
  });
}

export function setup() {
  console.log('Setting up load test...');
  
  // Register test users
  const registeredUsers = [];
  for (let i = 0; i < 10; i++) {
    const user = testUsers[i];
    const registerRes = http.post(`${BASE_URL}/api/v1/auth/register`, JSON.stringify({
      email: user.email,
      password: user.password,
      first_name: user.firstName,
      last_name: user.lastName,
    }), {
      headers: { 'Content-Type': 'application/json' },
    });

    if (registerRes.status === 201) {
      registeredUsers.push(user);
    }
  }

  console.log(`Registered ${registeredUsers.length} test users`);
  return { users: registeredUsers };
}

export default function (data) {
  const user = data.users[Math.floor(Math.random() * data.users.length)];

  group('Authentication Flow', () => {
    // Login
    const loginStart = Date.now();
    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
      email: user.email,
      password: user.password,
    }), {
      headers: { 'Content-Type': 'application/json' },
    });

    const loginSuccess = check(loginRes, {
      'login status is 200': (r) => r.status === 200,
      'login has access token': (r) => JSON.parse(r.body).access_token !== undefined,
    });

    errorRate.add(!loginSuccess);
    loginDuration.add(Date.now() - loginStart);
    requestCount.add(1);

    if (!loginSuccess) {
      return;
    }

    const accessToken = JSON.parse(loginRes.body).access_token;
    const headers = {
      'Authorization': `Bearer ${accessToken}`,
      'Content-Type': 'application/json',
    };

    sleep(1);

    // Get user profile
    group('User Profile', () => {
      const profileStart = Date.now();
      const profileRes = http.get(`${BASE_URL}/api/v1/auth/profile`, { headers });

      check(profileRes, {
        'profile status is 200': (r) => r.status === 200,
        'profile has email': (r) => JSON.parse(r.body).email === user.email,
      });

      apiDuration.add(Date.now() - profileStart);
      requestCount.add(1);
    });

    sleep(1);

    // List GPUs
    group('GPU Marketplace', () => {
      const gpuStart = Date.now();
      const gpuRes = http.get(`${BASE_URL}/api/v1/gpus?available=true`, { headers });

      const gpuSuccess = check(gpuRes, {
        'gpu list status is 200': (r) => r.status === 200,
        'gpu list has data': (r) => JSON.parse(r.body).gpus !== undefined,
      });

      errorRate.add(!gpuSuccess);
      apiDuration.add(Date.now() - gpuStart);
      requestCount.add(1);
    });

    sleep(1);

    // Get wallet
    group('Wallet Operations', () => {
      const walletStart = Date.now();
      const walletRes = http.get(`${BASE_URL}/api/v1/wallet`, { headers });

      check(walletRes, {
        'wallet status is 200 or 404': (r) => r.status === 200 || r.status === 404,
      });

      apiDuration.add(Date.now() - walletStart);
      requestCount.add(1);
    });

    sleep(1);

    // List jobs
    group('Job Management', () => {
      const jobsStart = Date.now();
      const jobsRes = http.get(`${BASE_URL}/api/v1/jobs`, { headers });

      check(jobsRes, {
        'jobs list status is 200': (r) => r.status === 200,
      });

      apiDuration.add(Date.now() - jobsStart);
      requestCount.add(1);
    });

    sleep(2);
  });
}

export function teardown(data) {
  console.log('Cleaning up load test...');
  // Cleanup would go here if needed
}

export function handleSummary(data) {
  return {
    'load-test-summary.json': JSON.stringify(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;

  let summary = '\n';
  summary += `${indent}Test Summary:\n`;
  summary += `${indent}=============\n\n`;

  // Requests
  summary += `${indent}Total Requests: ${data.metrics.http_reqs.values.count}\n`;
  summary += `${indent}Request Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)}/s\n\n`;

  // Duration
  summary += `${indent}Response Time:\n`;
  summary += `${indent}  avg: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
  summary += `${indent}  min: ${data.metrics.http_req_duration.values.min.toFixed(2)}ms\n`;
  summary += `${indent}  max: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms\n`;
  summary += `${indent}  p(95): ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += `${indent}  p(99): ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n\n`;

  // Errors
  const errorRate = (data.metrics.http_req_failed.values.rate * 100).toFixed(2);
  summary += `${indent}Error Rate: ${errorRate}%\n`;
  summary += `${indent}Failed Requests: ${data.metrics.http_req_failed.values.passes}\n\n`;

  // Custom metrics
  if (data.metrics.login_duration) {
    summary += `${indent}Login Duration (avg): ${data.metrics.login_duration.values.avg.toFixed(2)}ms\n`;
  }
  if (data.metrics.api_duration) {
    summary += `${indent}API Duration (avg): ${data.metrics.api_duration.values.avg.toFixed(2)}ms\n`;
  }

  return summary;
}

