import React, { useState, useEffect } from 'react';
import { invoke } from '@tauri-apps/api/tauri';

interface DynamicPricingProps {
  gpus: GpuInfo[];
  onPriceUpdate: (gpuId: string, newPrice: number) => void;
}

interface GpuInfo {
  id: string;
  name: string;
  model: string;
  vram_total_mb: number;
  vram_free_mb: number;
  utilization_gpu_percent?: number;
  temperature_c?: number;
  power_draw_w?: number;
  is_available_for_rent: boolean;
  current_hourly_rate_dgpu: number | null;
}

interface MarketData {
  average_market_price: number;
  demand_factor: number;
  supply_factor: number;
  peak_hours_multiplier: number;
  competitor_prices: CompetitorPrice[];
  trending_direction: 'up' | 'down' | 'stable';
}

interface CompetitorPrice {
  provider: string;
  gpu_model: string;
  price_usd: number;
  price_dgpu: number;
  availability: number;
}

interface PricingStrategy {
  strategy_type: 'aggressive' | 'competitive' | 'premium' | 'custom';
  base_multiplier: number;
  peak_hour_boost: number;
  performance_bonus: number;
  demand_sensitivity: number;
  minimum_price: number;
  maximum_price: number;
}

interface PricingRecommendation {
  gpu_id: string;
  current_price: number;
  recommended_price: number;
  reason: string;
  expected_impact: string;
  confidence: number;
  market_position: 'below' | 'competitive' | 'above';
}

export const DynamicPricing: React.FC<DynamicPricingProps> = ({ gpus, onPriceUpdate }) => {
  const [marketData, setMarketData] = useState<MarketData | null>(null);
  const [pricingStrategy, setPricingStrategy] = useState<PricingStrategy>({
    strategy_type: 'competitive',
    base_multiplier: 1.0,
    peak_hour_boost: 1.3,
    performance_bonus: 1.2,
    demand_sensitivity: 0.8,
    minimum_price: 0.5,
    maximum_price: 15.0
  });
  const [recommendations, setRecommendations] = useState<PricingRecommendation[]>([]);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [autoAdjustEnabled, setAutoAdjustEnabled] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<string | null>(null);

  // Fetch market data
  const fetchMarketData = async () => {
    try {
      const data = await invoke<MarketData>('get_gpu_market_data');
      setMarketData(data);
    } catch (error) {
      console.error('Failed to fetch market data:', error);
      // Mock data for development
      setMarketData({
        average_market_price: 2.50,
        demand_factor: 1.2,
        supply_factor: 0.8,
        peak_hours_multiplier: 1.4,
        competitor_prices: [
          { provider: 'Vast.ai', gpu_model: 'RTX 4090', price_usd: 0.79, price_dgpu: 2.1, availability: 85 },
          { provider: 'RunPod', gpu_model: 'RTX 4090', price_usd: 0.89, price_dgpu: 2.3, availability: 92 },
          { provider: 'Lambda Labs', gpu_model: 'RTX 4090', price_usd: 1.20, price_dgpu: 3.2, availability: 78 }
        ],
        trending_direction: 'up'
      });
    }
  };

  // Calculate dynamic pricing
  const calculateDynamicPricing = async () => {
    if (!marketData) return;
    
    setIsAnalyzing(true);
    
    try {
      const newRecommendations: PricingRecommendation[] = [];
      
      for (const gpu of gpus) {
        if (!gpu.is_available_for_rent) continue;
        
        const currentPrice = gpu.current_hourly_rate_dgpu || 2.0;
        
        // Performance-based pricing
        const performanceScore = calculateGpuPerformanceScore(gpu);
        const performanceMultiplier = 1 + (performanceScore - 0.5) * pricingStrategy.performance_bonus;
        
        // Market demand adjustments
        const demandMultiplier = marketData.demand_factor * pricingStrategy.demand_sensitivity;
        
        // Peak hours detection
        const isPeakHour = isCurrentlyPeakHour();
        const peakMultiplier = isPeakHour ? pricingStrategy.peak_hour_boost : 1.0;
        
        // Utilization-based pricing
        const utilizationBonus = gpu.utilization_gpu_percent ? 
          1 + ((100 - gpu.utilization_gpu_percent) / 100) * 0.1 : 1.0;
        
        // Calculate base recommended price
        let recommendedPrice = marketData.average_market_price * 
          pricingStrategy.base_multiplier * 
          performanceMultiplier * 
          demandMultiplier * 
          peakMultiplier * 
          utilizationBonus;
        
        // Apply constraints
        recommendedPrice = Math.max(pricingStrategy.minimum_price, 
          Math.min(pricingStrategy.maximum_price, recommendedPrice));
        
        // Determine market position
        const marketPosition = getMarketPosition(recommendedPrice, gpu.model);
        
        // Generate recommendation reason
        const reason = generatePricingReason(
          currentPrice, 
          recommendedPrice, 
          isPeakHour, 
          performanceScore, 
          marketData.trending_direction
        );
        
        // Calculate expected impact
        const _expectedImpact = calculateExpectedImpact(currentPrice, recommendedPrice);
        
        // Calculate confidence based on market data quality
        const confidence = calculateConfidence(marketData, gpu);
        
        newRecommendations.push({
          gpu_id: gpu.id,
          current_price: currentPrice,
          recommended_price: Math.round(recommendedPrice * 100) / 100,
          reason,
          expected_impact: _expectedImpact,
          confidence,
          market_position: marketPosition
        });
      }
      
      setRecommendations(newRecommendations);
      setLastUpdate(new Date().toISOString());
      
    } catch (error) {
      console.error('Failed to calculate dynamic pricing:', error);
    } finally {
      setIsAnalyzing(false);
    }
  };

  // Helper functions
  const calculateGpuPerformanceScore = (gpu: GpuInfo): number => {
    // Simple performance scoring based on VRAM and utilization
    const vramScore = Math.min(gpu.vram_total_mb / 24000, 1.0); // Normalize to 24GB
    const utilizationScore = (gpu.utilization_gpu_percent || 50) / 100;
    const tempScore = gpu.temperature_c ? Math.max(0, (85 - gpu.temperature_c) / 85) : 0.7;
    
    return (vramScore * 0.5) + (utilizationScore * 0.3) + (tempScore * 0.2);
  };

  const isCurrentlyPeakHour = (): boolean => {
    const hour = new Date().getUTCHours();
    // Peak hours: 6PM-11PM UTC (typical US evening hours)
    return hour >= 18 && hour <= 23;
  };

  const getMarketPosition = (price: number, model: string): 'below' | 'competitive' | 'above' => {
    if (!marketData) return 'competitive';
    
    const avgCompetitorPrice = marketData.competitor_prices
      .filter(cp => cp.gpu_model.includes(model.split(' ')[1] || model))
      .reduce((sum, cp) => sum + cp.price_dgpu, 0) / 
      Math.max(marketData.competitor_prices.length, 1);
    
    if (price < avgCompetitorPrice * 0.9) return 'below';
    if (price > avgCompetitorPrice * 1.1) return 'above';
    return 'competitive';
  };

  const generatePricingReason = (
    current: number, 
    recommended: number, 
    isPeak: boolean, 
    perfScore: number,
    trend: string
  ): string => {
    const change = recommended - current;
    const changePercent = Math.abs(change / current * 100);
    
    if (Math.abs(change) < 0.05) {
      return 'Price is optimal - no adjustment needed';
    }
    
    let reason = change > 0 ? 'Increase recommended: ' : 'Decrease recommended: ';
    
    if (isPeak) reason += 'Peak demand hours, ';
    if (perfScore > 0.8) reason += 'High performance GPU, ';
    if (trend === 'up') reason += 'Market trending upward, ';
    if (trend === 'down') reason += 'Market softening, ';
    
    reason += `${changePercent.toFixed(0)}% adjustment for competitiveness`;
    
    return reason;
  };

  const calculateExpectedImpact = (current: number, recommended: number): string => {
    const change = (recommended - current) / current;
    
    if (Math.abs(change) < 0.05) {
      return 'Minimal impact expected';
    }
    
    if (change > 0) {
      return `+${(change * 100).toFixed(0)}% revenue potential, may reduce bookings`;
    } else {
      return `${(change * 100).toFixed(0)}% revenue, likely increased demand`;
    }
  };

  const calculateConfidence = (market: MarketData, gpu: GpuInfo): number => {
    let confidence = 0.5; // Base confidence
    
    // More data = higher confidence
    if (market.competitor_prices.length > 2) confidence += 0.2;
    if (gpu.utilization_gpu_percent !== undefined) confidence += 0.1;
    if (gpu.temperature_c !== undefined) confidence += 0.1;
    
    // Market stability increases confidence
    if (market.trending_direction === 'stable') confidence += 0.1;
    
    return Math.min(confidence, 1.0);
  };

  // Apply recommendations automatically
  const applyRecommendation = (recommendation: PricingRecommendation) => {
    onPriceUpdate(recommendation.gpu_id, recommendation.recommended_price);
  };

  // Apply all recommendations
  const applyAllRecommendations = () => {
    recommendations.forEach(rec => {
      if (rec.confidence > 0.7) { // Only apply high-confidence recommendations
        applyRecommendation(rec);
      }
    });
  };

  // Auto-adjustment logic
  useEffect(() => {
    if (autoAdjustEnabled) {
      const interval = setInterval(() => {
        calculateDynamicPricing().then(() => {
          // Auto-apply high-confidence recommendations
          recommendations.forEach(rec => {
            if (rec.confidence > 0.8 && Math.abs(rec.recommended_price - rec.current_price) > 0.1) {
              applyRecommendation(rec);
            }
          });
        });
      }, 10 * 60 * 1000); // Every 10 minutes
      
      return () => clearInterval(interval);
    }
  }, [autoAdjustEnabled, recommendations]);

  // Initial data fetch
  useEffect(() => {
    fetchMarketData();
    const interval = setInterval(fetchMarketData, 5 * 60 * 1000); // Every 5 minutes
    return () => clearInterval(interval);
  }, []);

  // Calculate pricing when data changes
  useEffect(() => {
    if (marketData && gpus.length > 0) {
      calculateDynamicPricing();
    }
  }, [marketData, gpus, pricingStrategy]);

  const getPositionColor = (position: string) => {
    switch (position) {
      case 'below': return '#ff9800';
      case 'competitive': return '#4caf50';
      case 'above': return '#f44336';
      default: return '#757575';
    }
  };

  const getConfidenceColor = (confidence: number) => {
    if (confidence > 0.8) return '#4caf50';
    if (confidence > 0.6) return '#ff9800';
    return '#f44336';
  };

  return (
    <div className="dynamic-pricing-container">
      <div className="pricing-header">
        <h3>🎯 Dynamic Pricing Engine</h3>
        <div className="pricing-controls">
          <button 
            onClick={calculateDynamicPricing}
            disabled={isAnalyzing}
            className="refresh-button"
          >
            {isAnalyzing ? '⏳ Analyzing...' : '🔄 Refresh Analysis'}
          </button>
          <label className="auto-adjust-toggle">
            <input
              type="checkbox"
              checked={autoAdjustEnabled}
              onChange={(e) => setAutoAdjustEnabled(e.target.checked)}
            />
            Auto-adjust pricing
          </label>
        </div>
      </div>

      {/* Market Overview */}
      {marketData && (
        <div className="market-overview">
          <h4>📊 Market Overview</h4>
          <div className="market-stats">
            <div className="market-stat">
              <span className="stat-label">Average Market Price:</span>
              <span className="stat-value">{marketData.average_market_price.toFixed(2)} DGPU/hr</span>
            </div>
            <div className="market-stat">
              <span className="stat-label">Demand Factor:</span>
              <span className="stat-value">{marketData.demand_factor.toFixed(2)}x</span>
            </div>
            <div className="market-stat">
              <span className="stat-label">Market Trend:</span>
              <span className={`stat-value ${marketData.trending_direction}`}>
                {marketData.trending_direction === 'up' ? '📈' : 
                 marketData.trending_direction === 'down' ? '📉' : '➡️'} 
                {marketData.trending_direction}
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Pricing Strategy Configuration */}
      <div className="pricing-strategy">
        <h4>⚙️ Pricing Strategy</h4>
        <div className="strategy-controls">
          <div className="strategy-row">
            <label>Strategy Type:</label>
            <select
              value={pricingStrategy.strategy_type}
              onChange={(e) => setPricingStrategy(prev => ({
                ...prev,
                strategy_type: e.target.value as PricingStrategy['strategy_type']
              }))}
            >
              <option value="aggressive">Aggressive (Higher prices)</option>
              <option value="competitive">Competitive (Market-based)</option>
              <option value="premium">Premium (Quality-focused)</option>
              <option value="custom">Custom</option>
            </select>
          </div>
          
          <div className="strategy-row">
            <label>Peak Hour Boost:</label>
            <input
              type="range"
              min="1.0"
              max="2.0"
              step="0.1"
              value={pricingStrategy.peak_hour_boost}
              onChange={(e) => setPricingStrategy(prev => ({
                ...prev,
                peak_hour_boost: parseFloat(e.target.value)
              }))}
            />
            <span>{pricingStrategy.peak_hour_boost.toFixed(1)}x</span>
          </div>
          
          <div className="strategy-row">
            <label>Performance Bonus:</label>
            <input
              type="range"
              min="1.0"
              max="2.0"
              step="0.1"
              value={pricingStrategy.performance_bonus}
              onChange={(e) => setPricingStrategy(prev => ({
                ...prev,
                performance_bonus: parseFloat(e.target.value)
              }))}
            />
            <span>{pricingStrategy.performance_bonus.toFixed(1)}x</span>
          </div>
          
          <div className="strategy-row">
            <label>Price Range:</label>
            <input
              type="number"
              value={pricingStrategy.minimum_price}
              onChange={(e) => setPricingStrategy(prev => ({
                ...prev,
                minimum_price: parseFloat(e.target.value)
              }))}
              min="0.1"
              step="0.1"
              style={{ width: '80px' }}
            />
            <span> - </span>
            <input
              type="number"
              value={pricingStrategy.maximum_price}
              onChange={(e) => setPricingStrategy(prev => ({
                ...prev,
                maximum_price: parseFloat(e.target.value)
              }))}
              min="1"
              step="0.5"
              style={{ width: '80px' }}
            />
            <span> DGPU/hr</span>
          </div>
        </div>
      </div>

      {/* Pricing Recommendations */}
      {recommendations.length > 0 && (
        <div className="pricing-recommendations">
          <div className="recommendations-header">
            <h4>💡 Pricing Recommendations</h4>
            <div className="header-actions">
              {lastUpdate && (
                <span className="last-update">
                  Last updated: {new Date(lastUpdate).toLocaleTimeString()}
                </span>
              )}
              <button 
                onClick={applyAllRecommendations}
                className="apply-all-button"
                disabled={recommendations.filter(r => r.confidence > 0.7).length === 0}
              >
                Apply All High-Confidence
              </button>
            </div>
          </div>
          
          <div className="recommendations-list">
            {recommendations.map((rec) => (
              <div key={rec.gpu_id} className="recommendation-item">
                <div className="recommendation-header">
                  <div className="gpu-info">
                    <span className="gpu-name">
                      {gpus.find(g => g.id === rec.gpu_id)?.name || 'Unknown GPU'}
                    </span>
                    <span 
                      className="market-position"
                      style={{ color: getPositionColor(rec.market_position) }}
                    >
                      {rec.market_position} market
                    </span>
                  </div>
                  <div className="confidence-indicator">
                    <span 
                      className="confidence-score"
                      style={{ color: getConfidenceColor(rec.confidence) }}
                    >
                      {(rec.confidence * 100).toFixed(0)}% confidence
                    </span>
                  </div>
                </div>
                
                <div className="recommendation-details">
                  <div className="price-comparison">
                    <div className="price-current">
                      <span className="price-label">Current:</span>
                      <span className="price-value">{rec.current_price.toFixed(2)} DGPU/hr</span>
                    </div>
                    <div className="price-arrow">→</div>
                    <div className="price-recommended">
                      <span className="price-label">Recommended:</span>
                      <span className="price-value recommended">{rec.recommended_price.toFixed(2)} DGPU/hr</span>
                    </div>
                  </div>
                  
                  <div className="recommendation-reason">
                    <span className="reason-text">{rec.reason}</span>
                  </div>
                  
                  <div className="expected-impact">
                    <span className="impact-text">{rec.expected_impact}</span>
                  </div>
                </div>
                
                <div className="recommendation-actions">
                  <button 
                    onClick={() => applyRecommendation(rec)}
                    className="apply-button"
                    disabled={Math.abs(rec.recommended_price - rec.current_price) < 0.05}
                  >
                    Apply
                  </button>
                  <button 
                    onClick={() => {
                      const customPrice = prompt('Enter custom price (DGPU/hr):', rec.current_price.toString());
                      if (customPrice && !isNaN(parseFloat(customPrice))) {
                        onPriceUpdate(rec.gpu_id, parseFloat(customPrice));
                      }
                    }}
                    className="custom-button"
                  >
                    Custom
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Competitor Analysis */}
      {marketData?.competitor_prices && (
        <div className="competitor-analysis">
          <h4>🏆 Competitor Analysis</h4>
          <div className="competitor-list">
            {marketData.competitor_prices.map((comp, index) => (
              <div key={index} className="competitor-item">
                <div className="competitor-name">{comp.provider}</div>
                <div className="competitor-model">{comp.gpu_model}</div>
                <div className="competitor-price">{comp.price_dgpu.toFixed(2)} DGPU/hr</div>
                <div className="competitor-availability">{comp.availability}% available</div>
              </div>
            ))}
          </div>
        </div>
      )}

      <style>
        {`
          .dynamic-pricing-container {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
          }

          .pricing-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            border-bottom: 2px solid #dee2e6;
            padding-bottom: 10px;
          }

          .pricing-header h3 {
            margin: 0;
            color: #333;
          }

          .pricing-controls {
            display: flex;
            gap: 15px;
            align-items: center;
          }

          .refresh-button {
            background: #007bff;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            cursor: pointer;
            transition: background-color 0.3s;
          }

          .refresh-button:hover:not(:disabled) {
            background: #0056b3;
          }

          .refresh-button:disabled {
            background: #6c757d;
            cursor: not-allowed;
          }

          .auto-adjust-toggle {
            display: flex;
            align-items: center;
            gap: 5px;
            font-size: 14px;
          }

          .market-overview {
            background: white;
            border-radius: 6px;
            padding: 15px;
            margin-bottom: 20px;
            border: 1px solid #dee2e6;
          }

          .market-overview h4 {
            margin: 0 0 15px 0;
            color: #333;
          }

          .market-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
          }

          .market-stat {
            display: flex;
            justify-content: space-between;
            padding: 8px;
            background: #f8f9fa;
            border-radius: 4px;
          }

          .stat-label {
            font-weight: bold;
            color: #666;
          }

          .stat-value {
            color: #333;
          }

          .stat-value.up {
            color: #28a745;
          }

          .stat-value.down {
            color: #dc3545;
          }

          .stat-value.stable {
            color: #17a2b8;
          }

          .pricing-strategy {
            background: white;
            border-radius: 6px;
            padding: 15px;
            margin-bottom: 20px;
            border: 1px solid #dee2e6;
          }

          .pricing-strategy h4 {
            margin: 0 0 15px 0;
            color: #333;
          }

          .strategy-controls {
            display: flex;
            flex-direction: column;
            gap: 12px;
          }

          .strategy-row {
            display: flex;
            align-items: center;
            gap: 10px;
          }

          .strategy-row label {
            min-width: 120px;
            font-weight: bold;
            color: #666;
          }

          .strategy-row select,
          .strategy-row input[type="range"],
          .strategy-row input[type="number"] {
            flex: 1;
          }

          .pricing-recommendations {
            background: white;
            border-radius: 6px;
            padding: 15px;
            margin-bottom: 20px;
            border: 1px solid #dee2e6;
          }

          .recommendations-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
          }

          .recommendations-header h4 {
            margin: 0;
            color: #333;
          }

          .header-actions {
            display: flex;
            align-items: center;
            gap: 15px;
          }

          .last-update {
            font-size: 12px;
            color: #666;
          }

          .apply-all-button {
            background: #28a745;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            cursor: pointer;
          }

          .apply-all-button:hover:not(:disabled) {
            background: #218838;
          }

          .apply-all-button:disabled {
            background: #6c757d;
            cursor: not-allowed;
          }

          .recommendations-list {
            display: flex;
            flex-direction: column;
            gap: 15px;
          }

          .recommendation-item {
            border: 1px solid #dee2e6;
            border-radius: 6px;
            padding: 15px;
            background: #f8f9fa;
          }

          .recommendation-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
          }

          .gpu-info {
            display: flex;
            flex-direction: column;
            gap: 4px;
          }

          .gpu-name {
            font-weight: bold;
            color: #333;
          }

          .market-position {
            font-size: 12px;
            font-weight: bold;
          }

          .confidence-indicator {
            text-align: right;
          }

          .confidence-score {
            font-size: 12px;
            font-weight: bold;
          }

          .recommendation-details {
            margin-bottom: 15px;
          }

          .price-comparison {
            display: flex;
            align-items: center;
            gap: 15px;
            margin-bottom: 10px;
          }

          .price-current,
          .price-recommended {
            display: flex;
            flex-direction: column;
            align-items: center;
          }

          .price-label {
            font-size: 12px;
            color: #666;
            margin-bottom: 2px;
          }

          .price-value {
            font-weight: bold;
            color: #333;
          }

          .price-value.recommended {
            color: #007bff;
          }

          .price-arrow {
            font-size: 18px;
            color: #666;
          }

          .recommendation-reason,
          .expected-impact {
            font-size: 14px;
            color: #666;
            margin-bottom: 5px;
          }

          .recommendation-actions {
            display: flex;
            gap: 10px;
          }

          .apply-button {
            background: #007bff;
            color: white;
            border: none;
            padding: 6px 12px;
            border-radius: 4px;
            cursor: pointer;
          }

          .apply-button:hover:not(:disabled) {
            background: #0056b3;
          }

          .apply-button:disabled {
            background: #6c757d;
            cursor: not-allowed;
          }

          .custom-button {
            background: #6c757d;
            color: white;
            border: none;
            padding: 6px 12px;
            border-radius: 4px;
            cursor: pointer;
          }

          .custom-button:hover {
            background: #5a6268;
          }

          .competitor-analysis {
            background: white;
            border-radius: 6px;
            padding: 15px;
            border: 1px solid #dee2e6;
          }

          .competitor-analysis h4 {
            margin: 0 0 15px 0;
            color: #333;
          }

          .competitor-list {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 10px;
          }

          .competitor-item {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 8px;
            padding: 10px;
            background: #f8f9fa;
            border-radius: 4px;
            font-size: 12px;
          }

          .competitor-name {
            font-weight: bold;
            color: #333;
          }

          .competitor-model {
            color: #666;
          }

          .competitor-price {
            font-weight: bold;
            color: #007bff;
          }

          .competitor-availability {
            color: #28a745;
          }
        `}
      </style>
    </div>
  );
};

export type { DynamicPricingProps, PricingStrategy, PricingRecommendation, MarketData }; 