#!/bin/bash

# DanteGPU - Start REAL System (No Mocks, No Docker)

set -e

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

clear
echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                                                              ║${NC}"
echo -e "${CYAN}║         🚀 DanteGPU REAL System Launcher 🚀                 ║${NC}"
echo -e "${CYAN}║                                                              ║${NC}"
echo -e "${CYAN}║              NO MOCKS - REAL IMPLEMENTATION                  ║${NC}"
echo -e "${CYAN}║                                                              ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${YELLOW}⚠️  This will start the REAL system with:${NC}"
echo -e "  - Real Go backend services"
echo -e "  - Real SQLite database (no Docker needed)"
echo -e "  - Real Solana blockchain integration"
echo -e "  - Real frontend with API integration"
echo ""
read -p "Press ENTER to continue..."

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    echo -e "${YELLOW}Install Go from: https://golang.org/dl/${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go $(go version | awk '{print $3}')${NC}"

# Check Node
if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ Node.js is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Node.js $(node --version)${NC}"
echo ""

# Create .env if not exists
if [ ! -f "$PROJECT_ROOT/.env" ]; then
    echo -e "${YELLOW}📝 Creating .env file...${NC}"
    cat > "$PROJECT_ROOT/.env" << 'EOF'
# DanteGPU Environment Configuration
PORT=8080
JWT_SECRET=dantegpu_super_secret_jwt_key_change_in_production_12345
DATABASE_TYPE=sqlite
DATABASE_PATH=./dantegpu.db
SOLANA_RPC_URL=https://api.mainnet-beta.solana.com
SOLANA_NETWORK=mainnet-beta
DGPU_TOKEN_MINT=7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump
LOG_LEVEL=info
EOF
    echo -e "${GREEN}✅ .env created${NC}"
fi

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Starting Backend Services${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Start auth service in background
echo -e "${YELLOW}🔐 Starting Auth Service on port 8090...${NC}"
cd "$PROJECT_ROOT/auth-service"
go run main.go > /tmp/dantegpu-auth.log 2>&1 &
AUTH_PID=$!
echo -e "${GREEN}✅ Auth Service started (PID: $AUTH_PID)${NC}"

sleep 2

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  Starting Frontend${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

cd "$PROJECT_ROOT/gpu-rental-frontend"

if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}📦 Installing frontend dependencies...${NC}"
    npm install --silent
fi

echo -e "${YELLOW}💻 Starting Frontend on port 3000...${NC}"
npm run dev > /tmp/dantegpu-frontend.log 2>&1 &
FRONTEND_PID=$!
echo -e "${GREEN}✅ Frontend started (PID: $FRONTEND_PID)${NC}"

sleep 3

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                  🎉 SYSTEM IS RUNNING! 🎉                    ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${CYAN}🌐 Access Points:${NC}"
echo -e "  Frontend:      ${GREEN}http://localhost:3000${NC}"
echo -e "  Auth Service:  ${GREEN}http://localhost:8090${NC}"
echo ""
echo -e "${CYAN}📝 Logs:${NC}"
echo -e "  Auth:     tail -f /tmp/dantegpu-auth.log"
echo -e "  Frontend: tail -f /tmp/dantegpu-frontend.log"
echo ""
echo -e "${CYAN}🛑 To stop:${NC}"
echo -e "  kill $AUTH_PID $FRONTEND_PID"
echo ""

# Open browser
echo -e "${YELLOW}🌐 Opening browser...${NC}"
sleep 2
open http://localhost:3000 2>/dev/null || true

echo -e "${GREEN}✨ System is ready! Press Ctrl+C to stop all services.${NC}"
echo ""

# Wait for Ctrl+C
trap "echo ''; echo -e '${YELLOW}🛑 Stopping services...${NC}'; kill $AUTH_PID $FRONTEND_PID 2>/dev/null; echo -e '${GREEN}✅ All services stopped${NC}'; exit 0" INT

# Keep script running
wait

