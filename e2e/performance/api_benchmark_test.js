/**
 * k6 API Benchmark Test
 *
 * Simple benchmark for key API endpoints
 *
 * Usage:
 *   k6 run e2e/performance/api_benchmark_test.js
 */

import http from 'k6/http';
import { check } from 'k6';
import { Trend } from 'k6/metrics';

// Custom metrics for each endpoint
const listClustersTrend = new Trend('api_list_clusters');
const getClusterTrend = new Trend('api_get_cluster');
const listNodesTrend = new Trend('api_list_nodes');
const listPodsTrend = new Trend('api_list_pods');
const aiChatTrend = new Trend('api_ai_chat');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_TOKEN = __ENV.API_TOKEN || 'test-token';

const headers = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${API_TOKEN}`,
};

// Test options
export const options = {
  iterations: 10,
  vus: 1,
  thresholds: {
    api_list_clusters: ['p(95)<500'],
    api_get_cluster: ['p(95)<300'],
  },
};

export default function () {
  // 1. List clusters
  let res = http.get(`${BASE_URL}/api/v1/clusters`, { headers });
  listClustersTrend.add(res.timings.duration);
  check(res, { 'list clusters': (r) => r.status === 200 || r.status === 401 });

  // 2. Get cluster detail (if we have any)
  res = http.get(`${BASE_URL}/api/v1/clusters/1`, { headers });
  getClusterTrend.add(res.timings.duration);
  check(res, { 'get cluster': (r) => r.status === 200 || r.status === 404 || r.status === 401 });

  // 3. List nodes
  res = http.get(`${BASE_URL}/api/v1/clusters/1/nodes`, { headers });
  listNodesTrend.add(res.timings.duration);
  check(res, { 'list nodes': (r) => r.status === 200 || r.status === 404 || r.status === 401 });

  // 4. List pods
  res = http.get(`${BASE_URL}/api/v1/clusters/1/namespaces/default/pods`, { headers });
  listPodsTrend.add(res.timings.duration);
  check(res, { 'list pods': (r) => r.status === 200 || r.status === 404 || r.status === 401 });

  // 5. AI chat (if enabled)
  const chatPayload = JSON.stringify({
    message: '你好',
    session_id: 'benchmark-test',
  });
  res = http.post(`${BASE_URL}/api/v1/ai/chat`, chatPayload, { headers, timeout: '30s' });
  aiChatTrend.add(res.timings.duration);
  check(res, { 'ai chat': (r) => r.status === 200 || r.status === 404 || r.status === 401 });
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const lines = [];

  lines.push(`${indent}API Benchmark Results`);
  lines.push(`${indent}======================`);
  lines.push('');

  for (const [metric, trend] of Object.entries(data.metrics)) {
    if (trend.values && trend.values.avg !== undefined) {
      lines.push(`${indent}${metric}:`);
      lines.push(`${indent}  avg: ${trend.values.avg.toFixed(2)}ms`);
      lines.push(`${indent}  p95: ${trend.values['p(95)']?.toFixed(2) || 'N/A'}ms`);
      lines.push(`${indent}  max: ${trend.values.max?.toFixed(2) || 'N/A'}ms`);
    }
  }

  return lines.join('\n');
}
