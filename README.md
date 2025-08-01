# DanteGPU - Decentralized GPU Rental Platform

## 🚀 Genel Bakış

DanteGPU, blockchain tabanlı merkezi olmayan bir GPU kiralama platformudur. Bu platform, GPU sahiplerinin boşta olan kaynaklarını monetize etmelerini ve kullanıcıların ihtiyaç duydukları GPU gücünü esnek bir şekilde kiralamalarını sağlar.

### ✨ Ana Özellikler

- **🔗 Blockchain Entegrasyonu**: Solana blockchain üzerinde dGPU token ile ödemeler
- **⚡ Gerçek Zamanlı Faturalandırma**: Dakika bazında kullanım takibi
- **🎯 Dinamik Fiyatlandırma**: Talep ve GPU özelliklerine göre otomatik fiyat ayarı
- **🖥️ Çoklu GPU Desteği**: NVIDIA, AMD ve Apple Silicon GPU'ları
- **📊 Kapsamlı Monitoring**: Prometheus, Grafana ve Loki ile izleme
- **🔒 Güvenlik**: JWT tabanlı kimlik doğrulama ve yetkilendirme
- **🌐 Modern UI**: React/Next.js tabanlı kullanıcı dostu arayüz

## 🏗️ Mimari Genel Bakış

Platform, mikroservis mimarisi kullanarak geliştirilmiştir ve aşağıdaki ana katmanlardan oluşur:

### 1. **Altyapı Katmanı**
- **PostgreSQL**: Ana veritabanı
- **Redis**: Önbellek ve rate limiting
- **NATS JetStream**: Mesaj kuyruğu ve event streaming
- **Consul**: Servis keşfi ve konfigürasyon
- **MinIO**: S3 uyumlu dosya depolama

### 2. **Mikroservisler**
- **API Gateway**: Merkezi giriş noktası ve yönlendirme
- **Auth Service**: Kimlik doğrulama ve kullanıcı yönetimi
- **Billing Service**: Blockchain ödemeleri ve faturalandırma
- **Provider Registry**: GPU sağlayıcı kayıt ve yönetimi
- **Scheduler Orchestrator**: İş yükü planlama ve yürütme
- **Storage Service**: Dosya yönetimi ve depolama
- **Monitoring Services**: Sistem izleme ve alerting

### 3. **Kullanıcı Arayüzleri**
- **Web Dashboard**: React tabanlı ana platform arayüzü
- **Provider GUI**: Tauri tabanlı masaüstü uygulaması
- **Provider Daemon**: Arka plan GPU monitoring servisi

## 🚀 Hızlı Başlangıç

### Gereksinimler

- Docker & Docker Compose
- Git
- 8GB+ RAM
- 20GB+ disk alanı

### Kurulum

1. **Depoyu klonlayın:**
```bash
git clone https://github.com/dante-gpu/dantegpu-core.git
cd dantegpu-core
```

2. **Ortam değişkenlerini ayarlayın:**
```bash
cp env.production.example .env
# .env dosyasını düzenleyin
```

3. **Platformu başlatın:**
```bash
./deploy-production.sh
```

### Servis URL'leri

- **Ana Platform**: http://localhost:3000
- **API Gateway**: http://localhost:8080
- **Grafana**: http://localhost:3001 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Consul UI**: http://localhost:8500
- **MinIO Console**: http://localhost:9001

## 📋 Servis Detayları

### API Gateway (Port: 8080)
**Teknoloji**: Go, Chi Router
**Rol**: Merkezi API yönlendirme, kimlik doğrulama, rate limiting

**Ana Endpoint'ler:**
- `GET /health` - Sistem sağlık kontrolü
- `POST /api/v1/auth/*` - Kimlik doğrulama işlemleri
- `GET /api/v1/providers/*` - GPU sağlayıcı listesi
- `POST /api/v1/jobs/*` - İş yükü yönetimi
- `GET /api/v1/billing/*` - Faturalandırma sorguları

### Auth Service (Port: 8090)
**Teknoloji**: Python, FastAPI
**Rol**: Kullanıcı kayıt, giriş, JWT token yönetimi

**Ana Endpoint'ler:**
- `POST /api/v1/auth/register` - Kullanıcı kaydı
- `POST /api/v1/auth/login` - Kullanıcı girişi
- `POST /api/v1/auth/refresh` - Token yenileme
- `GET /api/v1/auth/me` - Kullanıcı profili

### Billing Service (Port: 8082)
**Teknoloji**: Go, Solana SDK
**Rol**: Blockchain ödemeleri, dGPU token işlemleri, faturalandırma

**Ana Endpoint'ler:**
- `POST /api/v1/billing/create-wallet` - Cüzdan oluşturma
- `POST /api/v1/billing/transfer` - Token transferi
- `GET /api/v1/billing/balance` - Bakiye sorgulama
- `POST /api/v1/billing/estimate-cost` - Maliyet tahmini

### Provider Registry (Port: 8081)
**Teknoloji**: Go
**Rol**: GPU sağlayıcı kayıt, durum takibi, performans metrikleri

**Ana Endpoint'ler:**
- `POST /api/v1/providers/register` - Sağlayıcı kaydı
- `GET /api/v1/providers/list` - Mevcut sağlayıcılar
- `PUT /api/v1/providers/{id}/status` - Durum güncelleme
- `GET /api/v1/providers/{id}/metrics` - Performans metrikleri

### Scheduler Orchestrator (Port: 8084)
**Teknoloji**: Go
**Rol**: İş yükü planlama, Docker container yönetimi, kaynak tahsisi

**Ana Endpoint'ler:**
- `POST /api/v1/jobs/submit` - İş yükü gönderimi
- `GET /api/v1/jobs/{id}/status` - İş durumu
- `DELETE /api/v1/jobs/{id}` - İş iptali
- `GET /api/v1/jobs/{id}/logs` - İş logları

### Storage Service (Port: 8083)
**Teknoloji**: Go, MinIO SDK
**Rol**: Dosya yükleme, indirme, S3 uyumlu depolama

**Ana Endpoint'ler:**
- `POST /api/v1/storage/upload` - Dosya yükleme
- `GET /api/v1/storage/download/{id}` - Dosya indirme
- `DELETE /api/v1/storage/{id}` - Dosya silme
- `GET /api/v1/storage/list` - Dosya listesi

## 🔧 Geliştirme

### Yerel Geliştirme Ortamı

1. **Servisleri ayrı ayrı çalıştırma:**
```bash
# Altyapı servisleri
docker-compose up -d postgres redis nats consul minio

# Auth service
cd auth-service
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8090

# API Gateway
cd api-gateway
go run cmd/main.go
```

2. **Frontend geliştirme:**
```bash
cd provider-web-app
npm install
npm run dev
```

### Test Etme

```bash
# Sistem sağlık kontrolü
curl http://localhost:8080/health

# Demo kullanıcı oluşturma
cd auth-service
python create_test_user.py

# GPU rental testi
./run-gpu-rental-test.sh
```

## 📊 Monitoring ve Logging

### Grafana Dashboards
- **System Overview**: Genel sistem metrikleri
- **GPU Utilization**: GPU kullanım oranları
- **Transaction Monitoring**: Blockchain işlem takibi
- **Service Health**: Mikroservis sağlık durumu

### Log Aggregation
- **Loki**: Merkezi log toplama
- **Promtail**: Log shipping
- **Grafana**: Log görselleştirme

### Alerting
- **Prometheus AlertManager**: Otomatik uyarılar
- **Webhook entegrasyonu**: Slack, Discord, email

## 🔒 Güvenlik

### Kimlik Doğrulama
- JWT token tabanlı auth
- Refresh token mekanizması
- Role-based access control (RBAC)

### API Güvenliği
- Rate limiting (60 req/min)
- Request validation
- CORS policy
- HTTPS zorunluluğu

### Blockchain Güvenliği
- Private key şifreleme
- Transaction verification
- Escrow sistemi
- Multi-signature desteği

## 🌐 Deployment

### Production Deployment

```bash
# Production ortamı için
./deploy-production.sh

# Docker Compose ile
docker-compose -f docker-compose.prod.yml up -d

# Kubernetes için (opsiyonel)
kubectl apply -f k8s/
```

### Scaling

```bash
# Servisleri ölçeklendirme
docker-compose up -d --scale api-gateway=3
docker-compose up -d --scale billing-service=2
```

## 📈 Performans

### Benchmark Sonuçları
- **Concurrent Users**: 1000+ (test edildi)
- **GPU Providers**: Sınırsız
- **Jobs/Second**: 100+ (ölçeklendirme ile)
- **Transaction Throughput**: Solana network limitleri
- **Response Time**: <100ms (ortalama)

### Optimizasyon
- Container optimizasyonu
- Database connection pooling
- Redis caching
- Load balancing
- Horizontal scaling

## 🤝 Katkıda Bulunma

1. Fork edin
2. Feature branch oluşturun (`git checkout -b feature/amazing-feature`)
3. Commit edin (`git commit -m 'Add amazing feature'`)
4. Push edin (`git push origin feature/amazing-feature`)
5. Pull Request açın

## 📄 Lisans

Bu proje MIT lisansı altında lisanslanmıştır. Detaylar için [LICENSE](LICENSE) dosyasına bakın.

## 🆘 Destek

- **Documentation**: [Wiki](https://github.com/dante-gpu/dantegpu-core/wiki)
- **Issues**: [GitHub Issues](https://github.com/dante-gpu/dantegpu-core/issues)
- **Discord**: [Community Server](https://discord.gg/dantegpu)
- **Email**: support@dantegpu.com

---

**DanteGPU** - Decentralized GPU Computing for Everyone 🚀
