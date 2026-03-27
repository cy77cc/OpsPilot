import apiService from '../api';
import type { ApiResponse, PaginatedResponse } from '../api';

export interface RiskFinding {
  id: number;
  type: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  title: string;
  description: string;
  service_id: number;
  service_name: string;
  created_at: string;
  resolved_at?: string;
}

export interface Anomaly {
  id: number;
  type: string;
  metric: string;
  value: number;
  threshold: number;
  service_id: number;
  service_name: string;
  detected_at: string;
  resolved_at?: string;
}

export interface Suggestion {
  id: number;
  type: string;
  title: string;
  description: string;
  impact: 'high' | 'medium' | 'low';
  service_id: number;
  service_name: string;
  created_at: string;
  applied_at?: string;
}

export const aiopsApi = {
  getRiskFindings(): Promise<ApiResponse<PaginatedResponse<RiskFinding>>> {
    return apiService.get('/aiops/risk-findings');
  },
  getAnomalies(): Promise<ApiResponse<PaginatedResponse<Anomaly>>> {
    return apiService.get('/aiops/anomalies');
  },
  getSuggestions(): Promise<ApiResponse<PaginatedResponse<Suggestion>>> {
    return apiService.get('/aiops/suggestions');
  },
};
