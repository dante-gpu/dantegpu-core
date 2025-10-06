#!/usr/bin/env node

/**
 * DanteGPU Platform - Environment Validation Script
 * Validates all required environment variables before starting services
 */

const fs = require('fs');
const path = require('path');

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
};

// Required environment variables by category
const requiredEnvVars = {
  application: [
    'NODE_ENV',
    'APP_NAME',
    'APP_PORT',
  ],
  database: [
    'DB_HOST',
    'DB_PORT',
    'DB_NAME',
    'DB_USER',
    'DB_PASSWORD',
  ],
  redis: [
    'REDIS_HOST',
    'REDIS_PORT',
    'REDIS_PASSWORD',
  ],
  nats: [
    'NATS_URL',
    'NATS_SYSTEM_USER',
    'NATS_SYSTEM_PASSWORD',
  ],
  consul: [
    'CONSUL_HTTP_ADDR',
    'CONSUL_MASTER_TOKEN',
  ],
  minio: [
    'MINIO_ENDPOINT',
    'MINIO_ROOT_USER',
    'MINIO_ROOT_PASSWORD',
  ],
  solana: [
    'SOLANA_RPC_URL',
    'DGPU_TOKEN_MINT',
    'PLATFORM_WALLET_ADDRESS',
  ],
  jwt: [
    'JWT_SECRET',
    'JWT_REFRESH_SECRET',
  ],
  encryption: [
    'ENCRYPTION_KEY',
    'WALLET_ENCRYPTION_KEY',
  ],
};

// Optional but recommended variables
const recommendedEnvVars = [
  'SMTP_HOST',
  'SMTP_USER',
  'SENTRY_DSN',
  'GRAFANA_ADMIN_PASSWORD',
];

// Validation rules
const validationRules = {
  JWT_SECRET: (value) => value.length >= 32 || 'JWT_SECRET must be at least 32 characters',
  JWT_REFRESH_SECRET: (value) => value.length >= 32 || 'JWT_REFRESH_SECRET must be at least 32 characters',
  ENCRYPTION_KEY: (value) => value.length === 32 || 'ENCRYPTION_KEY must be exactly 32 characters',
  WALLET_ENCRYPTION_KEY: (value) => value.length >= 32 || 'WALLET_ENCRYPTION_KEY must be at least 32 characters',
  DB_PORT: (value) => !isNaN(value) && parseInt(value) > 0 || 'DB_PORT must be a valid port number',
  REDIS_PORT: (value) => !isNaN(value) && parseInt(value) > 0 || 'REDIS_PORT must be a valid port number',
  APP_PORT: (value) => !isNaN(value) && parseInt(value) > 0 || 'APP_PORT must be a valid port number',
  PLATFORM_FEE_PERCENTAGE: (value) => !isNaN(value) && parseFloat(value) >= 0 && parseFloat(value) <= 100 || 'PLATFORM_FEE_PERCENTAGE must be between 0 and 100',
  BCRYPT_ROUNDS: (value) => !isNaN(value) && parseInt(value) >= 10 && parseInt(value) <= 15 || 'BCRYPT_ROUNDS should be between 10 and 15',
};

// Load .env file
function loadEnvFile() {
  const envPath = path.join(process.cwd(), '.env');
  
  if (!fs.existsSync(envPath)) {
    console.error(`${colors.red}Error: .env file not found!${colors.reset}`);
    console.log(`${colors.yellow}Please copy .env.example to .env and fill in the values.${colors.reset}`);
    process.exit(1);
  }
  
  const envContent = fs.readFileSync(envPath, 'utf8');
  const envVars = {};
  
  envContent.split('\n').forEach(line => {
    line = line.trim();
    if (line && !line.startsWith('#')) {
      const [key, ...valueParts] = line.split('=');
      const value = valueParts.join('=').trim();
      if (key && value) {
        envVars[key.trim()] = value;
      }
    }
  });
  
  return envVars;
}

// Validate environment variables
function validateEnv() {
  console.log(`${colors.blue}╔════════════════════════════════════════════════════════════╗${colors.reset}`);
  console.log(`${colors.blue}║     DanteGPU Platform - Environment Validation            ║${colors.reset}`);
  console.log(`${colors.blue}╚════════════════════════════════════════════════════════════╝${colors.reset}\n`);
  
  const envVars = loadEnvFile();
  const errors = [];
  const warnings = [];
  
  // Check required variables
  console.log(`${colors.blue}Checking required environment variables...${colors.reset}\n`);
  
  Object.entries(requiredEnvVars).forEach(([category, vars]) => {
    console.log(`${colors.blue}${category.toUpperCase()}:${colors.reset}`);
    
    vars.forEach(varName => {
      const value = envVars[varName];
      
      if (!value || value === 'your_' + varName.toLowerCase() + '_here' || value.includes('your_')) {
        errors.push(`${varName} is not set or using placeholder value`);
        console.log(`  ${colors.red}✗${colors.reset} ${varName}`);
      } else {
        // Run validation rule if exists
        if (validationRules[varName]) {
          const validationResult = validationRules[varName](value);
          if (validationResult !== true) {
            errors.push(`${varName}: ${validationResult}`);
            console.log(`  ${colors.red}✗${colors.reset} ${varName} - ${validationResult}`);
          } else {
            console.log(`  ${colors.green}✓${colors.reset} ${varName}`);
          }
        } else {
          console.log(`  ${colors.green}✓${colors.reset} ${varName}`);
        }
      }
    });
    
    console.log('');
  });
  
  // Check recommended variables
  console.log(`${colors.blue}Checking recommended environment variables...${colors.reset}\n`);
  
  recommendedEnvVars.forEach(varName => {
    const value = envVars[varName];
    
    if (!value || value.includes('your_')) {
      warnings.push(`${varName} is not set (recommended for production)`);
      console.log(`  ${colors.yellow}⚠${colors.reset} ${varName}`);
    } else {
      console.log(`  ${colors.green}✓${colors.reset} ${varName}`);
    }
  });
  
  console.log('');
  
  // Print summary
  console.log(`${colors.blue}════════════════════════════════════════════════════════════${colors.reset}\n`);
  
  if (errors.length === 0 && warnings.length === 0) {
    console.log(`${colors.green}✓ All environment variables are properly configured!${colors.reset}\n`);
    return true;
  }
  
  if (errors.length > 0) {
    console.log(`${colors.red}Errors found (${errors.length}):${colors.reset}`);
    errors.forEach(error => console.log(`  ${colors.red}•${colors.reset} ${error}`));
    console.log('');
  }
  
  if (warnings.length > 0) {
    console.log(`${colors.yellow}Warnings (${warnings.length}):${colors.reset}`);
    warnings.forEach(warning => console.log(`  ${colors.yellow}•${colors.reset} ${warning}`));
    console.log('');
  }
  
  if (errors.length > 0) {
    console.log(`${colors.red}Please fix the errors before starting the application.${colors.reset}\n`);
    return false;
  }
  
  if (warnings.length > 0) {
    console.log(`${colors.yellow}Warnings can be ignored for development, but should be addressed for production.${colors.reset}\n`);
  }
  
  return true;
}

// Generate secure random values
function generateSecureValues() {
  const crypto = require('crypto');
  
  console.log(`${colors.blue}Generated secure values (save these to your .env file):${colors.reset}\n`);
  
  console.log(`JWT_SECRET=${crypto.randomBytes(32).toString('hex')}`);
  console.log(`JWT_REFRESH_SECRET=${crypto.randomBytes(32).toString('hex')}`);
  console.log(`ENCRYPTION_KEY=${crypto.randomBytes(32).toString('hex')}`);
  console.log(`WALLET_ENCRYPTION_KEY=${crypto.randomBytes(32).toString('hex')}`);
  console.log(`SESSION_SECRET=${crypto.randomBytes(32).toString('hex')}`);
  console.log(`WEBHOOK_SECRET=${crypto.randomBytes(32).toString('hex')}`);
  console.log(`CONSUL_MASTER_TOKEN=${crypto.randomBytes(16).toString('hex')}`);
  console.log(`CONSUL_AGENT_TOKEN=${crypto.randomBytes(16).toString('hex')}`);
  console.log(`CONSUL_ENCRYPT_KEY=${crypto.randomBytes(16).toString('base64')}`);
  console.log('');
}

// Main
const args = process.argv.slice(2);

if (args.includes('--generate')) {
  generateSecureValues();
} else {
  const isValid = validateEnv();
  process.exit(isValid ? 0 : 1);
}

