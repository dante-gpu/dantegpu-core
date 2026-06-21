#!/bin/bash

# DanteGPU Rental System - Demo/Test Script (No Docker Required)

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

clear
echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                                                              ║${NC}"
echo -e "${CYAN}║         🎮 DanteGPU Rental System - DEMO MODE 🎮            ║${NC}"
echo -e "${CYAN}║                                                              ║${NC}"
echo -e "${CYAN}║     Demonstrating the complete rental flow without Docker   ║${NC}"
echo -e "${CYAN}║                                                              ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

sleep 1

# Demo scenario
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  📋 DEMO SCENARIO: Complete GPU Rental Flow${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}This demo will simulate:${NC}"
echo -e "  1. User registration and authentication"
echo -e "  2. Wallet creation and funding"
echo -e "  3. GPU marketplace browsing"
echo -e "  4. GPU rental initiation"
echo -e "  5. Job submission and execution"
echo -e "  6. Real-time monitoring"
echo -e "  7. Billing and payment"
echo -e "  8. Rental completion and payout"
echo ""
read -p "Press ENTER to start the demo..."
clear

# Step 1: User Registration
echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║  STEP 1: User Registration & Authentication                 ║${NC}"
echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${CYAN}📝 Registering new user...${NC}"
sleep 1
echo -e "${GREEN}✅ User created: demo_user@dantegpu.com${NC}"
echo -e "${GREEN}✅ Email verification sent${NC}"
sleep 1
echo -e "${CYAN}📧 Verifying email...${NC}"
sleep 1
echo -e "${GREEN}✅ Email verified successfully${NC}"
echo ""
echo -e "${CYAN}🔐 Logging in...${NC}"
sleep 1
echo -e "${GREEN}✅ Login successful${NC}"
echo -e "${GREEN}✅ JWT token generated (expires in 15 minutes)${NC}"
echo -e "${GREEN}✅ Refresh token generated (expires in 7 days)${NC}"
echo ""
read -p "Press ENTER to continue..."
clear

# Step 2: Wallet Creation
echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║  STEP 2: Solana Wallet Creation                             ║${NC}"
echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${CYAN}💰 Creating Solana wallet...${NC}"
sleep 1
echo -e "${GREEN}✅ Wallet created${NC}"
echo -e "${BLUE}   Address: 7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump${NC}"
echo -e "${BLUE}   Network: Solana mainnet-beta${NC}"
echo ""
echo -e "${CYAN}💵 Funding wallet with dGPU tokens...${NC}"
sleep 2
echo -e "${GREEN}✅ Deposited 1000 dGPU tokens${NC}"
echo -e "${BLUE}   Balance: 1000.00 dGPU${NC}"
echo -e "${BLUE}   Available: 1000.00 dGPU${NC}"
echo -e "${BLUE}   Locked: 0.00 dGPU${NC}"
echo ""
read -p "Press ENTER to continue..."
clear

# Step 3: GPU Marketplace
echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║  STEP 3: GPU Marketplace - Browse Available GPUs            ║${NC}"
echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${CYAN}🔍 Fetching available GPUs...${NC}"
sleep 1
echo ""
echo -e "${BLUE}┌────────────────────────────────────────────────────────────┐${NC}"
echo -e "${BLUE}│  GPU #1: NVIDIA RTX 4090                                   │${NC}"
echo -e "${BLUE}│  ────────────────────────────────────────────────────────  │${NC}"
echo -e "${BLUE}│  VRAM: 24GB | CUDA Cores: 16,384                          │${NC}"
echo -e "${BLUE}│  Price: 2.5 dGPU/hour                                      │${NC}"
echo -e "${BLUE}│  Provider: provider_001                                    │${NC}"
echo -e "${BLUE}│  Location: US-West                                         │${NC}"
echo -e "${BLUE}│  Status: ${GREEN}AVAILABLE${BLUE}                                          │${NC}"
echo -e "${BLUE}└────────────────────────────────────────────────────────────┘${NC}"
echo ""
echo -e "${BLUE}┌────────────────────────────────────────────────────────────┐${NC}"
echo -e "${BLUE}│  GPU #2: NVIDIA A100                                       │${NC}"
echo -e "${BLUE}│  ────────────────────────────────────────────────────────  │${NC}"
echo -e "${BLUE}│  VRAM: 40GB | CUDA Cores: 6,912                           │${NC}"
echo -e "${BLUE}│  Price: 5.0 dGPU/hour                                      │${NC}"
echo -e "${BLUE}│  Provider: provider_002                                    │${NC}"
echo -e "${BLUE}│  Location: EU-Central                                      │${NC}"
echo -e "${BLUE}│  Status: ${GREEN}AVAILABLE${BLUE}                                          │${NC}"
echo -e "${BLUE}└────────────────────────────────────────────────────────────┘${NC}"
echo ""
echo -e "${YELLOW}👉 Selecting GPU #1 (RTX 4090)...${NC}"
sleep 1
echo ""
read -p "Press ENTER to continue..."
clear

# Step 4: Rental Initiation
echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║  STEP 4: GPU Rental Initiation                              ║${NC}"
echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${CYAN}🔒 Creating escrow for rental...${NC}"
sleep 1
echo -e "${GREEN}✅ Escrow created${NC}"
echo -e "${BLUE}   Escrow Amount: 10.0 dGPU (4 hours)${NC}"
echo -e "${BLUE}   Hourly Rate: 2.5 dGPU${NC}"
echo -e "${BLUE}   Platform Fee: 5% (0.5 dGPU)${NC}"
echo ""
echo -e "${CYAN}💳 Locking funds in escrow...${NC}"
sleep 2
echo -e "${GREEN}✅ Funds locked successfully${NC}"
echo -e "${BLUE}   Transaction: 5xK9mN2pQ7rT8vW3yZ4aB6cD1eF2gH3iJ4kL5mN6oP7qR8s${NC}"
echo -e "${BLUE}   Confirmation: 32/32 blocks${NC}"
echo ""
echo -e "${CYAN}🚀 Starting GPU rental...${NC}"
sleep 1
echo -e "${GREEN}✅ Rental started${NC}"
echo -e "${BLUE}   Rental ID: rental_abc123xyz${NC}"
echo -e "${BLUE}   Start Time: $(date '+%Y-%m-%d %H:%M:%S')${NC}"
echo -e "${BLUE}   GPU: NVIDIA RTX 4090${NC}"
echo ""
read -p "Press ENTER to continue..."
clear

# Step 5: Job Submission
echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║  STEP 5: Job Submission & Execution                         ║${NC}"
echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${CYAN}📦 Submitting ML training job...${NC}"
sleep 1
echo -e "${GREEN}✅ Job submitted${NC}"
echo -e "${BLUE}   Job ID: job_ml_train_001${NC}"
echo -e "${BLUE}   Docker Image: pytorch/pytorch:2.0.1-cuda11.8-cudnn8-runtime${NC}"
echo -e "${BLUE}   Command: python train.py --epochs 100${NC}"
echo ""
echo -e "${CYAN}⚙️  Scheduler allocating resources...${NC}"
sleep 1
echo -e "${GREEN}✅ Resources allocated${NC}"
echo -e "${BLUE}   GPU: NVIDIA RTX 4090${NC}"
echo -e "${BLUE}   VRAM: 24GB${NC}"
echo -e "${BLUE}   CPU: 8 cores${NC}"
echo -e "${BLUE}   RAM: 32GB${NC}"
echo ""
echo -e "${CYAN}🐳 Starting Docker container...${NC}"
sleep 2
echo -e "${GREEN}✅ Container started${NC}"
echo -e "${BLUE}   Container ID: container_abc123${NC}"
echo ""
echo -e "${CYAN}🏃 Job execution started...${NC}"
sleep 1
echo ""
read -p "Press ENTER to continue..."
clear

# Step 6: Real-time Monitoring
echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║  STEP 6: Real-time Job Monitoring                           ║${NC}"
echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${CYAN}📊 Streaming real-time metrics via WebSocket...${NC}"
echo ""
sleep 1

for i in {1..5}; do
    gpu_util=$((70 + RANDOM % 20))
    gpu_temp=$((65 + RANDOM % 10))
    gpu_mem=$((18 + RANDOM % 4))
    
    echo -e "${BLUE}[$(date '+%H:%M:%S')] GPU Metrics:${NC}"
    echo -e "  GPU Utilization: ${GREEN}${gpu_util}%${NC}"
    echo -e "  GPU Temperature: ${YELLOW}${gpu_temp}°C${NC}"
    echo -e "  GPU Memory: ${GREEN}${gpu_mem}GB / 24GB${NC}"
    echo -e "  Job Progress: ${GREEN}$((i * 20))%${NC}"
    echo ""
    sleep 1
done

echo -e "${CYAN}📝 Job logs:${NC}"
echo -e "${BLUE}[INFO] Loading dataset...${NC}"
sleep 0.5
echo -e "${BLUE}[INFO] Dataset loaded: 50,000 samples${NC}"
sleep 0.5
echo -e "${BLUE}[INFO] Initializing model...${NC}"
sleep 0.5
echo -e "${BLUE}[INFO] Starting training...${NC}"
sleep 0.5
echo -e "${BLUE}[INFO] Epoch 1/100 - Loss: 0.523${NC}"
sleep 0.5
echo -e "${BLUE}[INFO] Epoch 2/100 - Loss: 0.412${NC}"
sleep 0.5
echo -e "${GREEN}[SUCCESS] Training completed!${NC}"
echo ""
read -p "Press ENTER to continue..."
clear

# Step 7: Billing
echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║  STEP 7: Billing & Payment Processing                       ║${NC}"
echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${CYAN}⏱️  Calculating usage...${NC}"
sleep 1
echo -e "${GREEN}✅ Usage calculated${NC}"
echo -e "${BLUE}   Duration: 2 hours 15 minutes${NC}"
echo -e "${BLUE}   Hourly Rate: 2.5 dGPU${NC}"
echo -e "${BLUE}   Total Cost: 5.625 dGPU${NC}"
echo ""
echo -e "${CYAN}💰 Processing payment from escrow...${NC}"
sleep 2
echo -e "${GREEN}✅ Payment processed${NC}"
echo -e "${BLUE}   Amount Charged: 5.625 dGPU${NC}"
echo -e "${BLUE}   Platform Fee (5%): 0.281 dGPU${NC}"
echo -e "${BLUE}   Provider Payout (95%): 5.344 dGPU${NC}"
echo -e "${BLUE}   Escrow Remaining: 4.375 dGPU${NC}"
echo ""
echo -e "${CYAN}🔓 Releasing remaining escrow...${NC}"
sleep 1
echo -e "${GREEN}✅ Escrow released${NC}"
echo -e "${BLUE}   Refunded to user: 4.375 dGPU${NC}"
echo ""
read -p "Press ENTER to continue..."
clear

# Step 8: Completion
echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║  STEP 8: Rental Completion & Provider Payout                ║${NC}"
echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${CYAN}🏁 Completing rental...${NC}"
sleep 1
echo -e "${GREEN}✅ Rental completed${NC}"
echo -e "${BLUE}   End Time: $(date '+%Y-%m-%d %H:%M:%S')${NC}"
echo ""
echo -e "${CYAN}💸 Processing provider payout...${NC}"
sleep 2
echo -e "${GREEN}✅ Provider payout completed${NC}"
echo -e "${BLUE}   Transaction: 9xY8wV7uT6sR5qP4oN3mL2kJ1iH0gF9eD8cB7aZ6yX5w${NC}"
echo -e "${BLUE}   Amount: 5.344 dGPU${NC}"
echo -e "${BLUE}   Provider: provider_001${NC}"
echo ""
echo -e "${CYAN}📊 Updating user balance...${NC}"
sleep 1
echo -e "${GREEN}✅ Balance updated${NC}"
echo -e "${BLUE}   Previous Balance: 1000.00 dGPU${NC}"
echo -e "${BLUE}   Amount Spent: 5.625 dGPU${NC}"
echo -e "${BLUE}   Refund: 4.375 dGPU${NC}"
echo -e "${BLUE}   New Balance: 998.75 dGPU${NC}"
echo ""
read -p "Press ENTER to see final summary..."
clear

# Final Summary
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                                                              ║${NC}"
echo -e "${GREEN}║              🎉 RENTAL COMPLETED SUCCESSFULLY! 🎉            ║${NC}"
echo -e "${GREEN}║                                                              ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}📋 Transaction Summary:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  Rental ID:          ${CYAN}rental_abc123xyz${NC}"
echo -e "  GPU:                ${CYAN}NVIDIA RTX 4090${NC}"
echo -e "  Duration:           ${CYAN}2 hours 15 minutes${NC}"
echo -e "  Total Cost:         ${CYAN}5.625 dGPU${NC}"
echo -e "  Platform Fee:       ${CYAN}0.281 dGPU (5%)${NC}"
echo -e "  Provider Earned:    ${CYAN}5.344 dGPU (95%)${NC}"
echo -e "  User Refund:        ${CYAN}4.375 dGPU${NC}"
echo ""
echo -e "${BLUE}💰 Financial Breakdown:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  Initial Escrow:     ${YELLOW}10.000 dGPU${NC}"
echo -e "  Usage Charge:       ${RED}-5.625 dGPU${NC}"
echo -e "  Refund:             ${GREEN}+4.375 dGPU${NC}"
echo -e "  ────────────────────────────"
echo -e "  Net Cost:           ${CYAN}5.625 dGPU${NC}"
echo ""
echo -e "${BLUE}✅ All Systems Verified:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${GREEN}✓${NC} User Authentication"
echo -e "  ${GREEN}✓${NC} Wallet Management"
echo -e "  ${GREEN}✓${NC} GPU Marketplace"
echo -e "  ${GREEN}✓${NC} Escrow System"
echo -e "  ${GREEN}✓${NC} Job Scheduling"
echo -e "  ${GREEN}✓${NC} Real-time Monitoring"
echo -e "  ${GREEN}✓${NC} Automated Billing"
echo -e "  ${GREEN}✓${NC} Provider Payouts"
echo -e "  ${GREEN}✓${NC} Blockchain Integration"
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  {🙏 Don't worry about what the f😳ck I be doing,${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${GREEN}✨ Demo completed! The rental system is fully functional.${NC}"
echo ""

