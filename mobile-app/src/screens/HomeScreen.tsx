import React, { useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  RefreshControl,
} from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { useNavigation } from '@react-navigation/native';
import Icon from 'react-native-vector-icons/MaterialCommunityIcons';
import { apiClient } from '../services/api';
import { useAuthStore } from '../stores/authStore';

interface GPU {
  id: string;
  gpu_model: string;
  gpu_memory_gb: number;
  price_per_minute: number;
  is_available: boolean;
  provider_name: string;
}

interface Stats {
  total_gpus: number;
  available_gpus: number;
  active_jobs: number;
  total_spent: number;
}

export const HomeScreen: React.FC = () => {
  const navigation = useNavigation();
  const { user } = useAuthStore();

  // Fetch user stats
  const { data: stats, refetch: refetchStats } = useQuery({
    queryKey: ['userStats'],
    queryFn: async () => {
      const response = await apiClient.get<Stats>('/api/v1/users/stats');
      return response.data;
    },
  });

  // Fetch available GPUs
  const {
    data: gpus,
    isLoading,
    refetch: refetchGPUs,
  } = useQuery({
    queryKey: ['availableGPUs'],
    queryFn: async () => {
      const response = await apiClient.get<{ gpus: GPU[] }>(
        '/api/v1/gpus?available=true&limit=5'
      );
      return response.data.gpus;
    },
  });

  const [refreshing, setRefreshing] = React.useState(false);

  const onRefresh = React.useCallback(async () => {
    setRefreshing(true);
    await Promise.all([refetchStats(), refetchGPUs()]);
    setRefreshing(false);
  }, []);

  return (
    <ScrollView
      style={styles.container}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
      }
    >
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.greeting}>Welcome back,</Text>
        <Text style={styles.userName}>{user?.first_name || 'User'}!</Text>
      </View>

      {/* Stats Cards */}
      <View style={styles.statsContainer}>
        <View style={styles.statCard}>
          <Icon name="gpu" size={32} color="#6366f1" />
          <Text style={styles.statValue}>{stats?.available_gpus || 0}</Text>
          <Text style={styles.statLabel}>Available GPUs</Text>
        </View>

        <View style={styles.statCard}>
          <Icon name="briefcase" size={32} color="#10b981" />
          <Text style={styles.statValue}>{stats?.active_jobs || 0}</Text>
          <Text style={styles.statLabel}>Active Jobs</Text>
        </View>

        <View style={styles.statCard}>
          <Icon name="currency-usd" size={32} color="#f59e0b" />
          <Text style={styles.statValue}>
            ${(stats?.total_spent || 0).toFixed(2)}
          </Text>
          <Text style={styles.statLabel}>Total Spent</Text>
        </View>
      </View>

      {/* Quick Actions */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Quick Actions</Text>
        <View style={styles.actionsContainer}>
          <TouchableOpacity
            style={styles.actionButton}
            onPress={() => navigation.navigate('Marketplace' as never)}
          >
            <Icon name="shopping" size={24} color="#fff" />
            <Text style={styles.actionText}>Browse GPUs</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.actionButton}
            onPress={() => navigation.navigate('Jobs' as never)}
          >
            <Icon name="briefcase-plus" size={24} color="#fff" />
            <Text style={styles.actionText}>Submit Job</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.actionButton}
            onPress={() => navigation.navigate('Wallet' as never)}
          >
            <Icon name="wallet" size={24} color="#fff" />
            <Text style={styles.actionText}>My Wallet</Text>
          </TouchableOpacity>
        </View>
      </View>

      {/* Featured GPUs */}
      <View style={styles.section}>
        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>Featured GPUs</Text>
          <TouchableOpacity
            onPress={() => navigation.navigate('Marketplace' as never)}
          >
            <Text style={styles.seeAll}>See All</Text>
          </TouchableOpacity>
        </View>

        {isLoading ? (
          <Text style={styles.loadingText}>Loading GPUs...</Text>
        ) : (
          gpus?.map((gpu) => (
            <TouchableOpacity
              key={gpu.id}
              style={styles.gpuCard}
              onPress={() =>
                navigation.navigate('GPUDetails' as never, { id: gpu.id } as never)
              }
            >
              <View style={styles.gpuHeader}>
                <Icon name="gpu" size={24} color="#6366f1" />
                <Text style={styles.gpuModel}>{gpu.gpu_model}</Text>
              </View>
              <View style={styles.gpuDetails}>
                <View style={styles.gpuDetail}>
                  <Icon name="memory" size={16} color="#6b7280" />
                  <Text style={styles.gpuDetailText}>
                    {gpu.gpu_memory_gb}GB VRAM
                  </Text>
                </View>
                <View style={styles.gpuDetail}>
                  <Icon name="currency-usd" size={16} color="#6b7280" />
                  <Text style={styles.gpuDetailText}>
                    ${gpu.price_per_minute.toFixed(4)}/min
                  </Text>
                </View>
              </View>
              <View style={styles.gpuFooter}>
                <Text style={styles.providerName}>{gpu.provider_name}</Text>
                <View style={styles.availableBadge}>
                  <Text style={styles.availableText}>Available</Text>
                </View>
              </View>
            </TouchableOpacity>
          ))
        )}
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f9fafb',
  },
  header: {
    padding: 20,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#e5e7eb',
  },
  greeting: {
    fontSize: 16,
    color: '#6b7280',
  },
  userName: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#111827',
    marginTop: 4,
  },
  statsContainer: {
    flexDirection: 'row',
    padding: 16,
    gap: 12,
  },
  statCard: {
    flex: 1,
    backgroundColor: '#fff',
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  statValue: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#111827',
    marginTop: 8,
  },
  statLabel: {
    fontSize: 12,
    color: '#6b7280',
    marginTop: 4,
    textAlign: 'center',
  },
  section: {
    padding: 16,
  },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 16,
  },
  sectionTitle: {
    fontSize: 20,
    fontWeight: 'bold',
    color: '#111827',
  },
  seeAll: {
    fontSize: 14,
    color: '#6366f1',
    fontWeight: '600',
  },
  actionsContainer: {
    flexDirection: 'row',
    gap: 12,
  },
  actionButton: {
    flex: 1,
    backgroundColor: '#6366f1',
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
    gap: 8,
  },
  actionText: {
    color: '#fff',
    fontSize: 12,
    fontWeight: '600',
  },
  loadingText: {
    textAlign: 'center',
    color: '#6b7280',
    padding: 20,
  },
  gpuCard: {
    backgroundColor: '#fff',
    padding: 16,
    borderRadius: 12,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  gpuHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginBottom: 12,
  },
  gpuModel: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#111827',
  },
  gpuDetails: {
    flexDirection: 'row',
    gap: 16,
    marginBottom: 12,
  },
  gpuDetail: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  gpuDetailText: {
    fontSize: 14,
    color: '#6b7280',
  },
  gpuFooter: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  providerName: {
    fontSize: 14,
    color: '#6b7280',
  },
  availableBadge: {
    backgroundColor: '#d1fae5',
    paddingHorizontal: 12,
    paddingVertical: 4,
    borderRadius: 12,
  },
  availableText: {
    fontSize: 12,
    color: '#065f46',
    fontWeight: '600',
  },
});

