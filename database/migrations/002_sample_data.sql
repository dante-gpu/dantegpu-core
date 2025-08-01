-- Sample data for DanteGPU Platform

-- Insert sample GPU providers
INSERT INTO gpu_providers (id, name, location, contact_email, status) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'CloudGPU Pro', 'US-East', 'support@cloudgpupro.com', 'active'),
('550e8400-e29b-41d4-a716-446655440002', 'Enterprise Cloud', 'EU-West', 'contact@enterprisecloud.eu', 'active'),
('550e8400-e29b-41d4-a716-446655440003', 'GPU Farm', 'US-West', 'hello@gpufarm.com', 'active'),
('550e8400-e29b-41d4-a716-446655440004', 'AI Compute', 'US-Central', 'info@aicompute.io', 'active');

-- Insert sample GPU models
INSERT INTO gpu_models (id, name, manufacturer, architecture, memory_gb, memory_type, memory_bandwidth_gbps, cuda_cores, tensor_cores, base_clock_mhz, boost_clock_mhz, power_consumption_w, pcie_version, features, benchmarks) VALUES
('660e8400-e29b-41d4-a716-446655440001', 'RTX 4090', 'NVIDIA', 'Ada Lovelace', 24, 'GDDR6X', 1008.0, 16384, 512, 2230, 2520, 450, 'PCIe 4.0', 
 '{"dlss": "3.0", "ray_tracing": "3rd Gen RT Cores", "nvenc": true, "nvdec": true}',
 '{"gaming": 98, "compute": 95, "ai_training": 92, "rendering": 96}'),

('660e8400-e29b-41d4-a716-446655440002', 'A100', 'NVIDIA', 'Ampere', 80, 'HBM2e', 2039.0, 6912, 432, 1410, 1410, 400, 'PCIe 4.0',
 '{"multi_instance": true, "nvlink": true, "tensor_cores": "3rd Gen", "compute_capability": "8.0"}',
 '{"gaming": 75, "compute": 100, "ai_training": 100, "rendering": 85}'),

('660e8400-e29b-41d4-a716-446655440003', 'RTX 3080', 'NVIDIA', 'Ampere', 10, 'GDDR6X', 760.0, 8704, 272, 1440, 1710, 320, 'PCIe 4.0',
 '{"dlss": "2.0", "ray_tracing": "2nd Gen RT Cores", "nvenc": true, "nvdec": true}',
 '{"gaming": 85, "compute": 80, "ai_training": 75, "rendering": 82}'),

('660e8400-e29b-41d4-a716-446655440004', 'H100', 'NVIDIA', 'Hopper', 80, 'HBM3', 3350.0, 14592, 456, 1830, 1980, 700, 'PCIe 5.0',
 '{"transformer_engine": true, "nvlink": "4th Gen", "tensor_cores": "4th Gen", "compute_capability": "9.0"}',
 '{"gaming": 70, "compute": 100, "ai_training": 100, "rendering": 80}');

-- Insert sample GPU instances
INSERT INTO gpu_instances (id, provider_id, model_id, instance_id, price_per_hour, status, location, specs) VALUES
-- CloudGPU Pro instances
('770e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', 'gpu-4090-001', 2.50, 'available', 'US-East-1a', '{"cpu": "Intel Xeon", "ram_gb": 64, "storage_gb": 1000}'),
('770e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', 'gpu-4090-002', 2.50, 'busy', 'US-East-1b', '{"cpu": "Intel Xeon", "ram_gb": 64, "storage_gb": 1000}'),

-- Enterprise Cloud instances
('770e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440002', 'gpu-a100-007', 8.75, 'available', 'EU-West-1a', '{"cpu": "AMD EPYC", "ram_gb": 128, "storage_gb": 2000}'),
('770e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440002', 'gpu-a100-008', 8.75, 'maintenance', 'EU-West-1b', '{"cpu": "AMD EPYC", "ram_gb": 128, "storage_gb": 2000}'),

-- GPU Farm instances
('770e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440003', 'gpu-3080-042', 1.80, 'available', 'US-West-1a', '{"cpu": "Intel Core i9", "ram_gb": 32, "storage_gb": 500}'),
('770e8400-e29b-41d4-a716-446655440006', '550e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440003', 'gpu-3080-043', 1.80, 'available', 'US-West-1b', '{"cpu": "Intel Core i9", "ram_gb": 32, "storage_gb": 500}'),

-- AI Compute instances
('770e8400-e29b-41d4-a716-446655440007', '550e8400-e29b-41d4-a716-446655440004', '660e8400-e29b-41d4-a716-446655440004', 'gpu-h100-001', 12.00, 'available', 'US-Central-1a', '{"cpu": "Intel Xeon Platinum", "ram_gb": 256, "storage_gb": 4000}'),
('770e8400-e29b-41d4-a716-446655440008', '550e8400-e29b-41d4-a716-446655440004', '660e8400-e29b-41d4-a716-446655440004', 'gpu-h100-002', 12.00, 'available', 'US-Central-1b', '{"cpu": "Intel Xeon Platinum", "ram_gb": 256, "storage_gb": 4000}');

-- Insert sample users
INSERT INTO users (id, email, password_hash, name, balance, verified) VALUES
('880e8400-e29b-41d4-a716-446655440001', 'demo@dantegpu.com', '$2b$10$rOzJqQqQqQqQqQqQqQqQqO', 'Demo User', 100.00, true),
('880e8400-e29b-41d4-a716-446655440002', 'john@example.com', '$2b$10$rOzJqQqQqQqQqQqQqQqQqO', 'John Doe', 250.50, true),
('880e8400-e29b-41d4-a716-446655440003', 'jane@example.com', '$2b$10$rOzJqQqQqQqQqQqQqQqQqO', 'Jane Smith', 75.25, false);

-- Insert sample rentals
INSERT INTO gpu_rentals (id, user_id, gpu_instance_id, status, start_time, duration_hours, price_per_hour, total_cost, payment_status) VALUES
('990e8400-e29b-41d4-a716-446655440001', '880e8400-e29b-41d4-a716-446655440001', '770e8400-e29b-41d4-a716-446655440002', 'running', NOW() - INTERVAL '4 hours', 4.2, 2.50, 10.50, 'paid'),
('990e8400-e29b-41d4-a716-446655440002', '880e8400-e29b-41d4-a716-446655440002', '770e8400-e29b-41d4-a716-446655440005', 'paused', NOW() - INTERVAL '6 hours', 6.8, 1.80, 12.24, 'paid'),
('990e8400-e29b-41d4-a716-446655440003', '880e8400-e29b-41d4-a716-446655440001', '770e8400-e29b-41d4-a716-446655440004', 'completed', NOW() - INTERVAL '1 day', 2.5, 8.75, 21.88, 'paid');

-- Insert sample payment transactions
INSERT INTO payment_transactions (id, user_id, rental_id, type, amount, status, payment_method, external_transaction_id) VALUES
('aa0e8400-e29b-41d4-a716-446655440001', '880e8400-e29b-41d4-a716-446655440001', '990e8400-e29b-41d4-a716-446655440001', 'rental', 10.50, 'completed', 'stripe', 'pi_1234567890'),
('aa0e8400-e29b-41d4-a716-446655440002', '880e8400-e29b-41d4-a716-446655440002', '990e8400-e29b-41d4-a716-446655440002', 'rental', 12.24, 'completed', 'paypal', 'PAYID-1234567890'),
('aa0e8400-e29b-41d4-a716-446655440003', '880e8400-e29b-41d4-a716-446655440001', '990e8400-e29b-41d4-a716-446655440003', 'rental', 21.88, 'completed', 'stripe', 'pi_0987654321'),
('aa0e8400-e29b-41d4-a716-446655440004', '880e8400-e29b-41d4-a716-446655440001', NULL, 'topup', 100.00, 'completed', 'stripe', 'pi_topup123456');

-- Insert sample notifications
INSERT INTO notifications (id, user_id, type, title, message, read) VALUES
('bb0e8400-e29b-41d4-a716-446655440001', '880e8400-e29b-41d4-a716-446655440001', 'rental_started', 'GPU Rental Started', 'Your RTX 4090 rental has started successfully.', false),
('bb0e8400-e29b-41d4-a716-446655440002', '880e8400-e29b-41d4-a716-446655440001', 'payment_success', 'Payment Successful', 'Your payment of $100.00 has been processed successfully.', true),
('bb0e8400-e29b-41d4-a716-446655440003', '880e8400-e29b-41d4-a716-446655440002', 'rental_paused', 'GPU Rental Paused', 'Your RTX 3080 rental has been paused.', false);
