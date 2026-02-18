# 🚀 DanteGPU Projesini Başlatma Rehberi

{🙏 Don't worry about what the f😳ck I be doing, I'm Mock King AKINCI}

## ⚠️ ÖNEMLİ: Docker Gerekli

Projeyi çalıştırmak için **Docker Desktop**'ın çalışıyor olması gerekiyor.

### Docker'ı Başlatma

1. **Docker Desktop**'ı açın (Applications klasöründen)
2. Docker'ın başlamasını bekleyin (üst menü çubuğunda Docker ikonu yeşil olmalı)
3. Terminal'de kontrol edin:
   ```bash
   docker info
   ```

---

## 🎯 Hızlı Başlangıç (3 Adım)

### Adım 1: Docker Desktop'ı Başlat
- Docker Desktop uygulamasını açın
- Başlamasını bekleyin (~30 saniye)

### Adım 2: Projeyi Başlat
```bash
cd /Users/baturalpguvenc/Documents/GitHub/dantegpu-core
./scripts/start-rental-system.sh
```

### Adım 3: Frontend'i Başlat
Yeni bir terminal açın:
```bash
cd /Users/baturalpguvenc/Documents/GitHub/dantegpu-core/user-dashboard
npm install
npm run dev
```

---

## 📋 Detaylı Başlatma Adımları

### 1️⃣ Infrastructure Servisleri (Docker)

```bash
# .env dosyası oluştur
cp .env.example .env

# Docker servisleri başlat
docker-compose up -d postgres redis nats consul prometheus grafana loki

# Servislerin hazır olmasını bekle (~30 saniye)
docker-compose ps
```

**Çalışan Servisler:**
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- NATS: `localhost:4222`
- Consul: `http://localhost:8500`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

### 2️⃣ Backend Servisleri Başlat

#### API Gateway
```bash
cd api-gateway
go mod download
go run cmd/main.go
```
Çalışacak: `http://localhost:8080`

#### Auth Service (Python)
Yeni terminal:
```bash
cd auth-service
pip install -r requirements.txt
python main.py
```
Çalışacak: `http://localhost:8001`

#### Billing Service
Yeni terminal:
```bash
cd billing-service
go run internal/main.go
```
Çalışacak: `http://localhost:8002`

#### Provider Registry
Yeni terminal:
```bash
cd provider-registry
go run internal/main.go
```
Çalışacak: `http://localhost:8003`

#### Scheduler
Yeni terminal:
```bash
cd scheduler
go run internal/main.go
```
Çalışacak: `http://localhost:8004`

### 3️⃣ Frontend Uygulamaları

#### User Dashboard
```bash
cd user-dashboard
npm install
npm run dev
```
Açılacak: `http://localhost:5173`

#### Provider Web App
Yeni terminal:
```bash
cd provider-web-app
npm install
npm run dev
```
Açılacak: `http://localhost:5174`

---

## 🎮 Kullanıcı Olarak Test Etme

### 1. User Dashboard'a Git
Tarayıcıda: `http://localhost:5173`

### 2. Kayıt Ol
- Email: `test@dantegpu.com`
- Password: `Test123!@#`
- Kayıt ol butonuna tıkla

### 3. Wallet Oluştur
- Dashboard'da "Create Wallet" butonuna tıkla
- Solana wallet otomatik oluşturulacak

### 4. GPU'ları Gör
- "Browse GPUs" menüsüne git
- Mevcut GPU'ları listele

### 5. GPU Kirala
- Bir GPU seç
- "Rent Now" butonuna tıkla
- Escrow miktarını belirle
- Kiralama başlat

### 6. Job Gönder
- "Submit Job" menüsüne git
- Docker image seç (örn: `pytorch/pytorch:latest`)
- Komutu gir (örn: `python train.py`)
- Job'ı başlat

### 7. Monitoring
- "My Jobs" menüsünde job'ları gör
- Real-time GPU metrics
- Live logs

---

## 🔧 Provider Olarak Test Etme

### 1. Provider Web App'e Git
Tarayıcıda: `http://localhost:5174`

### 2. Provider Kaydı
- Email: `provider@dantegpu.com`
- Password: `Provider123!@#`
- Kayıt ol

### 3. GPU Ekle
- "Add GPU" butonuna tıkla
- GPU bilgilerini gir:
  - Model: RTX 4090
  - VRAM: 24GB
  - Price: 2.5 dGPU/hour

### 4. GPU'yu Aktif Et
- GPU listesinde "Activate" butonuna tıkla
- GPU marketplace'de görünür olacak

### 5. Kazançları Gör
- Dashboard'da earnings
- Rental history
- Payout requests

---

## 🧪 API'yi Test Etme

### Postman veya cURL ile

#### 1. Kullanıcı Kaydı
```bash
curl -X POST http://localhost:8001/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@dantegpu.com",
    "password": "Test123!@#",
    "username": "testuser"
  }'
```

#### 2. Login
```bash
curl -X POST http://localhost:8001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@dantegpu.com",
    "password": "Test123!@#"
  }'
```

#### 3. GPU Listesi
```bash
curl -X GET http://localhost:8003/api/v1/gpus \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 4. Wallet Oluştur
```bash
curl -X POST http://localhost:8002/api/v1/wallet/create \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 5. Job Gönder
```bash
curl -X POST http://localhost:8004/api/v1/jobs \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "gpu_id": "gpu_123",
    "docker_image": "pytorch/pytorch:latest",
    "command": "python train.py"
  }'
```

---

## 📊 Monitoring ve Logs

### Grafana Dashboard
1. `http://localhost:3000` adresine git
2. Login: `admin` / `admin`
3. Dashboards → Browse
4. DanteGPU dashboards'ları gör

### Prometheus Metrics
1. `http://localhost:9090` adresine git
2. Metrics'leri sorgula:
   - `api_requests_total`
   - `gpu_utilization`
   - `job_duration_seconds`

### Consul Service Discovery
1. `http://localhost:8500` adresine git
2. Services → Tüm servisleri gör
3. Health checks

### Docker Logs
```bash
# Tüm servislerin logları
docker-compose logs -f

# Sadece PostgreSQL
docker-compose logs -f postgres

# Sadece Redis
docker-compose logs -f redis
```

---

## 🛠️ Sorun Giderme

### Docker Çalışmıyor
```bash
# Docker'ı başlat
open -a Docker

# Durumu kontrol et
docker info
```

### Port Zaten Kullanımda
```bash
# Hangi process kullanıyor bul
lsof -i :5432  # PostgreSQL
lsof -i :6379  # Redis
lsof -i :8080  # API Gateway

# Process'i durdur
kill -9 <PID>
```

### Database Bağlantı Hatası
```bash
# PostgreSQL'in çalıştığını kontrol et
docker-compose ps postgres

# Logları kontrol et
docker-compose logs postgres

# Yeniden başlat
docker-compose restart postgres
```

### Frontend Başlamıyor
```bash
# Node modules'ları temizle
rm -rf node_modules package-lock.json
npm install

# Cache'i temizle
npm cache clean --force
npm install
```

---

## 🎯 Test Senaryoları

### Senaryo 1: Basit GPU Kiralama
1. Kullanıcı kaydı yap
2. Wallet oluştur
3. GPU listesini gör
4. Bir GPU kirala (1 saat)
5. Kiralama durumunu kontrol et

### Senaryo 2: Job Çalıştırma
1. GPU kirala
2. PyTorch job gönder
3. Real-time logs izle
4. Job tamamlanmasını bekle
5. Sonuçları indir

### Senaryo 3: Provider Kazancı
1. Provider olarak kayıt ol
2. GPU ekle
3. GPU'yu aktif et
4. Kullanıcı kiralama yapsın
5. Earnings'i kontrol et
6. Payout talep et

---

## 📞 Yardım

### Loglar
```bash
# Backend servislerin logları
tail -f api-gateway/logs/*.log
tail -f auth-service/logs/*.log

# Docker servislerin logları
docker-compose logs -f
```

### Durum Kontrolü
```bash
# Tüm servislerin durumu
docker-compose ps

# Health check
curl http://localhost:8080/health
curl http://localhost:8001/health
```

---

## ✅ Başarı Kriterleri

Proje başarıyla çalışıyorsa:

- ✅ Docker servisleri çalışıyor (postgres, redis, nats, consul)
- ✅ API Gateway yanıt veriyor (`http://localhost:8080/health`)
- ✅ Auth Service çalışıyor (`http://localhost:8001/health`)
- ✅ User Dashboard açılıyor (`http://localhost:5173`)
- ✅ Kullanıcı kaydı yapılabiliyor
- ✅ Wallet oluşturulabiliyor
- ✅ GPU'lar listeleniyor

---

**{🙏 Don't worry about what the f😳ck I be doing, I'm Mock King AKINCI}**

**Projeyi başlatmak için Docker Desktop'ı açın ve yukarıdaki adımları takip edin!**

