# DanteGPU Platform Terminal Restoration Guide

## Mevcut Durum Analizi
Platform şu anda şu sorunları yaşıyor:
1. **Auth Service**: base_class.py dosyası bozuk (syntax error)
2. **Consul**: data-dir parametresi eksik
3. **Provider Daemon**: gopsutil bağımlılıkları eksik
4. **Frontend**: Yanlış dizin referansı
5. **Log Streaming**: WebSocket bağlantı sorunları

## Adım Adım Terminal Çözümü

### 1. Workspace'e Git
```bash
cd /Users/baturalpguvenc/Documents/GitHub/dantegpu-core
```

### 2. Auth Service base_class.py Dosyasını Düzelt
```bash
# Bozuk dosyayı yedekle
cp auth-service/app/db/base_class.py auth-service/app/db/base_class.py.backup

# Düzeltilmiş içeriği yaz
cat > auth-service/app/db/base_class.py << 'EOF'
from sqlalchemy.orm import DeclarativeBase
from sqlalchemy import MetaData
from typing import Any

# I should define a naming convention for constraints for Alembic migrations.
# This helps keep migration files consistent and avoids unnamed constraints.
convention = {
    "ix": "ix_%(column_0_label)s",
    "uq": "uq_%(table_name)s_%(column_0_name)s",
    "ck": "ck_%(table_name)s_%(constraint_name)s",
    "fk": "fk_%(table_name)s_%(column_0_name)s_%(referred_table_name)s",
    "pk": "pk_%(table_name)s",
}

metadata = MetaData(naming_convention=convention)

# I need to create the base class for my SQLAlchemy models.
class Base(DeclarativeBase):
    metadata = metadata
    # Optionally, define type annotation map or other base configurations
    # type_annotation_map = {dict[str, Any]: JSON}
    pass
EOF
```

### 3. Provider Daemon Bağımlılıklarını Ekle
```bash
# Provider daemon dizinine git
cd provider-daemon

# Eksik gopsutil paketlerini ekle
go get github.com/shirou/gopsutil/v3/cpu
go get github.com/shirou/gopsutil/v3/disk
go get github.com/shirou/gopsutil/v3/host
go get github.com/shirou/gopsutil/v3/mem

# Bağımlılıkları temizle ve güncelle
go mod tidy

# Ana dizine dön
cd ..
```

### 4. Scheduler Orchestrator Service go.sum Eksikliğini Düzelt
```bash
# Scheduler service dizinine git
cd scheduler-orchestrator-service

# go.sum dosyasını oluştur
go mod tidy

# Ana dizine dön
cd ..
```

### 5. Docker Compose Dosyasındaki Consul Ayarını Düzelt
```bash
# Consul command'ını düzelt (data-dir parametresi ekle)
sed -i '' 's/-log-level=INFO/-data-dir=\/consul\/data\n      -log-level=INFO/' docker-compose.yml
```

### 6. Frontend Service Ayarını Düzelt (provider-web-app'e yönlendir)
```bash
# Frontend service'i aktifleştir ve doğru dizine yönlendir
sed -i '' 's/# *frontend:/frontend:/' docker-compose.yml
sed -i '' 's/# *build:/    build:/' docker-compose.yml
sed -i '' 's/# *context: .\/frontend\/web-app/      context: .\/provider-web-app/' docker-compose.yml
```

### 7. API Gateway Environment Variables Ekle
```bash
# API Gateway'e LOKI_URL ve PORT environment variables ekle
# Bu zaten docker-compose.yml'de mevcut, kontrol et:
grep -A 20 "api-gateway:" docker-compose.yml | grep -E "(LOKI_URL|PORT)"
```

### 8. Tüm Docker Images'ları Temiz Bir Şekilde Build Et
```bash
# Eski container'ları durdur ve temizle
docker-compose down -v

# Eski images'ları temizle
docker system prune -f

# Tüm servisleri cache olmadan build et
docker-compose build --no-cache
```

### 9. Platform'u Başlat
```bash
# Servisleri detached modda başlat
docker-compose up -d
```

### 10. Service Durumlarını Kontrol Et
```bash
# Tüm container'ların durumunu kontrol et
docker-compose ps

# Başarısız olan servislerin loglarını kontrol et
docker logs dante-auth-service
docker logs dante-consul
docker logs dante-provider-registry-service
```

### 11. WebSocket Log Streaming Test Et
```bash
# API Gateway'in çalıştığını kontrol et
curl -f http://localhost:8080/health

# Frontend'in çalıştığını kontrol et
curl -f http://localhost:5174

# Loki'nin çalıştığını kontrol et
curl -f http://localhost:3100/ready
```

### 12. GPU Rental İşlemini Test Et

#### A. Test Kullanıcısı Oluştur
```bash
# Auth service container'ına gir ve test kullanıcısı oluştur
docker exec -it dante-auth-service python create_test_user.py
```

#### B. Provider Registry'ye Mock Provider Ekle
```bash
# Mock provider'ın çalıştığını kontrol et
docker logs dante-mock-provider

# Provider registry'deki provider'ları listele
curl -X GET http://localhost:8081/providers
```

#### C. GPU Rental Test Script'ini Çalıştır
```bash
# GPU rental test script'ini çalıştır
chmod +x run-gpu-rental-test.sh
./run-gpu-rental-test.sh
```

#### D. Real-time Log Monitoring
```bash
# Tüm servislerin loglarını real-time takip et
docker-compose logs -f
```

### 13. Frontend Terminal'de Log Streaming'i Test Et
```bash
# Browser'da frontend'i aç
open http://localhost:5174

# WebSocket bağlantısını test et (başka terminal'de)
wscat -c ws://localhost:8080/logs/stream
```

### 14. Sorun Giderme Komutları

#### Service Restart
```bash
# Belirli bir service'i restart et
docker-compose restart dante-auth-service
docker-compose restart dante-api-gateway
```

#### Database İşlemleri
```bash
# PostgreSQL'e bağlan
docker exec -it dante-postgres psql -U dante_user -d dante_auth

# Tabloları listele
\dt

# Kullanıcıları listele
SELECT * FROM users;
```

#### NATS İşlemleri
```bash
# NATS stream'lerini kontrol et
docker exec -it dante-nats nats stream list
docker exec -it dante-nats nats consumer list
```

#### Consul İşlemleri
```bash
# Consul UI'ye eriş
open http://localhost:8500

# Consul members'ı kontrol et
docker exec -it dante-consul consul members
```

### 15. GPU Rental İşlem Adımları

#### A. Authentication
```bash
# Login endpoint'ine POST request
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "testpassword"}'
```

#### B. Available Providers Listele
```bash
# Mevcut GPU provider'ları listele
curl -X GET http://localhost:8080/providers \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### C. Job Submit Et
```bash
# GPU job'ı submit et
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "provider_id": "mock-provider-id",
    "job_type": "training",
    "requirements": {
      "gpu_memory": 8192,
      "duration_minutes": 60
    },
    "payment": {
      "amount": 10.0,
      "currency": "USD"
    }
  }'
```

#### D. Job Status Takip Et
```bash
# Job status'unu kontrol et
curl -X GET http://localhost:8080/jobs/JOB_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 16. Monitoring ve Logs

#### Grafana Dashboard
```bash
# Grafana'ya eriş
open http://localhost:3000
# Username: admin, Password: admin
```

#### Prometheus Metrics
```bash
# Prometheus'a eriş
open http://localhost:9090
```

#### Real-time System Monitoring
```bash
# System resource kullanımını monitor et
docker stats

# Disk kullanımını kontrol et
docker system df
```

### 17. Cleanup ve Reset

#### Tam Reset
```bash
# Tüm container'ları durdur ve temizle
docker-compose down -v

# Tüm images'ları temizle
docker system prune -a -f

# Volumes'ları temizle
docker volume prune -f
```

#### Partial Reset
```bash
# Sadece belirli servisleri restart et
docker-compose restart dante-auth-service dante-api-gateway
```

## Önemli Notlar

1. **Port Çakışması**: Eğer portlarda çakışma varsa, docker-compose.yml'deki port mapping'lerini değiştirin
2. **Memory Kullanımı**: Platform yoğun memory kullanır, en az 8GB RAM önerilir
3. **Log Dosyaları**: Loglar `/var/lib/docker/containers/` altında saklanır
4. **Backup**: Önemli değişikliklerden önce backup alın
5. **Environment Variables**: Production'da environment variables'ları güvenli şekilde yönetin

## Başarı Kriterleri

✅ Tüm servisler healthy durumda
✅ WebSocket log streaming çalışıyor
✅ Frontend terminal'de loglar görünüyor
✅ GPU rental API'leri yanıt veriyor
✅ Mock provider job'ları kabul ediyor
✅ Payment processing çalışıyor
✅ Real-time monitoring aktif

Bu adımları sırasıyla takip ederek platform'u tamamen çalışır hale getirebilirsiniz. 