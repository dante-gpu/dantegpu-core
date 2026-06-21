#!/bin/bash

# DanteGPU Project Launcher - Interactive Startup

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

clear
echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                                                              ║${NC}"
echo -e "${CYAN}║           🚀 DanteGPU Project Launcher 🚀                   ║${NC}"
echo -e "${CYAN}║                                                              ║${NC}"
echo -e "${CYAN}║         Interactive Project Startup & Testing               ║${NC}"
echo -e "${CYAN}║                                                              ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check Docker
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Step 1: Checking Docker${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running!${NC}"
    echo ""
    echo -e "${YELLOW}Please start Docker Desktop:${NC}"
    echo -e "  1. Open Docker Desktop from Applications"
    echo -e "  2. Wait for Docker to start (~30 seconds)"
    echo -e "  3. Run this script again"
    echo ""
    echo -e "${CYAN}Opening Docker Desktop for you...${NC}"
    open -a Docker
    echo ""
    echo -e "${YELLOW}Waiting for Docker to start...${NC}"
    
    # Wait for Docker
    for i in {1..60}; do
        if docker info >/dev/null 2>&1; then
            echo -e "${GREEN}✅ Docker is now running!${NC}"
            sleep 2
            break
        fi
        echo -n "."
        sleep 1
    done
    
    if ! docker info >/dev/null 2>&1; then
        echo -e "${RED}❌ Docker failed to start. Please start it manually and run this script again.${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✅ Docker is running${NC}"
fi

echo ""
read -p "Press ENTER to continue..."
clear

# Menu
while true; do
    clear
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║              DanteGPU Project - Main Menu                    ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BLUE}What would you like to do?${NC}"
    echo ""
    echo -e "  ${GREEN}1${NC}) 🚀 Start Full System (Infrastructure + Backend + Frontend)"
    echo -e "  ${GREEN}2${NC}) 🐳 Start Infrastructure Only (Docker services)"
    echo -e "  ${GREEN}3${NC}) 💻 Start Frontend Only (User Dashboard)"
    echo -e "  ${GREEN}4${NC}) 🔧 Start Backend Services"
    echo -e "  ${GREEN}5${NC}) 📊 Open Monitoring Dashboards"
    echo -e "  ${GREEN}6${NC}) 🧪 Run Demo (No Docker needed)"
    echo -e "  ${GREEN}7${NC}) 📖 View Documentation"
    echo -e "  ${GREEN}8${NC}) 🛑 Stop All Services"
    echo -e "  ${GREEN}9${NC}) ❌ Exit"
    echo ""
    read -p "Enter your choice (1-9): " choice
    
    case $choice in
        1)
            clear
            echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${MAGENTA}║           Starting Full DanteGPU System                      ║${NC}"
            echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
            echo ""
            
            # Create .env if not exists
            if [ ! -f "$PROJECT_ROOT/.env" ]; then
                echo -e "${YELLOW}📝 Creating .env file...${NC}"
                cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
                sed -i.bak 's/your_secure_password_here/dante_password/g' "$PROJECT_ROOT/.env"
                sed -i.bak 's/your_redis_password_here//g' "$PROJECT_ROOT/.env"
                rm -f "$PROJECT_ROOT/.env.bak"
                echo -e "${GREEN}✅ .env file created${NC}"
            fi
            
            # Start infrastructure
            echo -e "${BLUE}🐳 Starting infrastructure services...${NC}"
            cd "$PROJECT_ROOT"
            docker-compose up -d postgres redis nats consul prometheus grafana loki
            
            echo -e "${YELLOW}⏳ Waiting for services to be ready (30 seconds)...${NC}"
            sleep 30
            
            echo -e "${GREEN}✅ Infrastructure started!${NC}"
            echo ""
            echo -e "${BLUE}📊 Service URLs:${NC}"
            echo -e "  PostgreSQL:  ${GREEN}localhost:5432${NC}"
            echo -e "  Redis:       ${GREEN}localhost:6379${NC}"
            echo -e "  NATS:        ${GREEN}localhost:4222${NC}"
            echo -e "  Consul:      ${GREEN}http://localhost:8500${NC}"
            echo -e "  Prometheus:  ${GREEN}http://localhost:9090${NC}"
            echo -e "  Grafana:     ${GREEN}http://localhost:3000${NC} (admin/admin)"
            echo ""
            
            # Start frontend
            echo -e "${BLUE}💻 Starting User Dashboard...${NC}"
            if [ -d "$PROJECT_ROOT/user-dashboard" ]; then
                cd "$PROJECT_ROOT/user-dashboard"
                
                if [ ! -d "node_modules" ]; then
                    echo -e "${YELLOW}📦 Installing dependencies...${NC}"
                    npm install
                fi
                
                echo -e "${GREEN}✅ Starting frontend on http://localhost:5173${NC}"
                echo ""
                echo -e "${CYAN}Opening browser...${NC}"
                sleep 2
                open http://localhost:5173 2>/dev/null || true
                
                # Start in background
                npm run dev &
                FRONTEND_PID=$!
                echo -e "${GREEN}✅ Frontend started (PID: $FRONTEND_PID)${NC}"
            fi
            
            echo ""
            echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${GREEN}║                  🎉 SYSTEM IS READY! 🎉                      ║${NC}"
            echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
            echo ""
            echo -e "${BLUE}🌐 Access Points:${NC}"
            echo -e "  User Dashboard:  ${GREEN}http://localhost:5173${NC}"
            echo -e "  Grafana:         ${GREEN}http://localhost:3000${NC}"
            echo -e "  Consul:          ${GREEN}http://localhost:8500${NC}"
            echo ""
            echo -e "${YELLOW}Note: Backend services need to be started separately (Option 4)${NC}"
            echo ""
            read -p "Press ENTER to return to menu..."
            ;;
            
        2)
            clear
            echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${MAGENTA}║           Starting Infrastructure Services                   ║${NC}"
            echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
            echo ""
            
            cd "$PROJECT_ROOT"
            docker-compose up -d postgres redis nats consul prometheus grafana loki
            
            echo -e "${YELLOW}⏳ Waiting for services...${NC}"
            sleep 15
            
            echo ""
            docker-compose ps
            echo ""
            echo -e "${GREEN}✅ Infrastructure services started!${NC}"
            echo ""
            read -p "Press ENTER to return to menu..."
            ;;
            
        3)
            clear
            echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${MAGENTA}║              Starting User Dashboard                         ║${NC}"
            echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
            echo ""
            
            if [ -d "$PROJECT_ROOT/user-dashboard" ]; then
                cd "$PROJECT_ROOT/user-dashboard"
                
                if [ ! -d "node_modules" ]; then
                    echo -e "${YELLOW}📦 Installing dependencies...${NC}"
                    npm install
                fi
                
                echo -e "${GREEN}🚀 Starting frontend...${NC}"
                echo -e "${CYAN}Opening http://localhost:5173${NC}"
                echo ""
                open http://localhost:5173 2>/dev/null || true
                npm run dev
            else
                echo -e "${RED}❌ user-dashboard directory not found${NC}"
                read -p "Press ENTER to return to menu..."
            fi
            ;;
            
        4)
            clear
            echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${MAGENTA}║              Backend Services Instructions                   ║${NC}"
            echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
            echo ""
            echo -e "${YELLOW}Backend services need to be started in separate terminals:${NC}"
            echo ""
            echo -e "${BLUE}1. API Gateway:${NC}"
            echo -e "   ${CYAN}cd api-gateway && go run cmd/main.go${NC}"
            echo ""
            echo -e "${BLUE}2. Auth Service:${NC}"
            echo -e "   ${CYAN}cd auth-service && python main.py${NC}"
            echo ""
            echo -e "${BLUE}3. Billing Service:${NC}"
            echo -e "   ${CYAN}cd billing-service && go run internal/main.go${NC}"
            echo ""
            echo -e "${BLUE}4. Provider Registry:${NC}"
            echo -e "   ${CYAN}cd provider-registry && go run internal/main.go${NC}"
            echo ""
            echo -e "${BLUE}5. Scheduler:${NC}"
            echo -e "   ${CYAN}cd scheduler && go run internal/main.go${NC}"
            echo ""
            read -p "Press ENTER to return to menu..."
            ;;
            
        5)
            clear
            echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${MAGENTA}║              Opening Monitoring Dashboards                   ║${NC}"
            echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
            echo ""
            
            echo -e "${CYAN}Opening dashboards...${NC}"
            open http://localhost:3000 2>/dev/null || true  # Grafana
            sleep 1
            open http://localhost:9090 2>/dev/null || true  # Prometheus
            sleep 1
            open http://localhost:8500 2>/dev/null || true  # Consul
            
            echo -e "${GREEN}✅ Dashboards opened in browser${NC}"
            echo ""
            echo -e "${BLUE}Dashboard URLs:${NC}"
            echo -e "  Grafana:    ${GREEN}http://localhost:3000${NC} (admin/admin)"
            echo -e "  Prometheus: ${GREEN}http://localhost:9090${NC}"
            echo -e "  Consul:     ${GREEN}http://localhost:8500${NC}"
            echo ""
            read -p "Press ENTER to return to menu..."
            ;;
            
        6)
            clear
            "$PROJECT_ROOT/scripts/demo-rental-system.sh"
            ;;
            
        7)
            clear
            echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${MAGENTA}║                    Documentation                             ║${NC}"
            echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
            echo ""
            echo -e "${BLUE}📚 Available Documentation:${NC}"
            echo ""
            echo -e "  ${GREEN}1.${NC} START_PROJECT.md - Complete startup guide"
            echo -e "  ${GREEN}2.${NC} README.md - Project overview"
            echo -e "  ${GREEN}3.${NC} docs/API_DOCUMENTATION.md - API reference"
            echo -e "  ${GREEN}4.${NC} docs/ARCHITECTURE.md - System architecture"
            echo -e "  ${GREEN}5.${NC} docs/RUNBOOK.md - Operations guide"
            echo ""
            echo -e "${YELLOW}Opening START_PROJECT.md...${NC}"
            cat "$PROJECT_ROOT/START_PROJECT.md" | less
            ;;
            
        8)
            clear
            echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${MAGENTA}║                Stopping All Services                         ║${NC}"
            echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════╝${NC}"
            echo ""
            
            cd "$PROJECT_ROOT"
            echo -e "${YELLOW}🛑 Stopping Docker services...${NC}"
            docker-compose down
            
            echo -e "${YELLOW}🛑 Stopping Node processes...${NC}"
            pkill -f "vite" 2>/dev/null || true
            pkill -f "npm run dev" 2>/dev/null || true
            
            echo -e "${GREEN}✅ All services stopped${NC}"
            echo ""
            read -p "Press ENTER to return to menu..."
            ;;
            
        9)
            clear
            echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${CYAN}║                                                              ║${NC}"
            echo -e "${CYAN}║                  Thank you for using                         ║${NC}"
            echo -e "${CYAN}║                    DanteGPU Platform!                        ║${NC}"
            echo -e "${CYAN}║                                                              ║${NC}"
            echo -e "${CYAN}║  {🙏 Don't worry about what the f😳ck I be doing,          ║${NC}"
            echo -e "${CYAN}║                                                              ║${NC}"
            echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
            echo ""
            exit 0
            ;;
            
        *)
            echo -e "${RED}Invalid choice. Please enter 1-9.${NC}"
            sleep 2
            ;;
    esac
done

