/**
 * k6 Performance Test: Cluster Import and Management
 *
 * Run with:
 *   k6 run e2e/performance/cluster_import_test.js
 *
 * With environment variables:
 *   BASE_URL=http://localhost:8080 API_TOKEN=your-token k6 run e2e/performance/cluster_import_test.js
 *
 * For more details see: https://k6.io/docs/
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// ============================================================
// Custom Metrics
// ============================================================

const clusterImportSuccess = new Rate('cluster_import_success');
const clusterImportLatency = new Trend('cluster_import_latency');
const clusterListLatency = new Trend('cluster_list_latency');
const clusterDetailLatency = new Trend('cluster_detail_latency');
const clusterDeleteLatency = new Trend('cluster_delete_latency');
const errorCount = new Counter('errors');

// ============================================================
// Test Configuration
// ============================================================

export const options = {
  // Test scenarios
  scenarios: {
    // Smoke test - basic functionality
    smoke: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
      exec: 'smokeTest',
    },
    // Load test - normal load
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 5 },   // Ramp up
        { duration: '1m', target: 10 },   // Stay at 10 users
        { duration: '30s', target: 20 },  // Peak load
        { duration: '30s', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '30s',
      exec: 'loadTest',
    },
    // Stress test - find breaking point
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 50 },
        { duration: '2m', target: 100 },
        { duration: '1m', target: 0 },
      ],
      gracefulRampDown: '30s',
      exec: 'loadTest',
    },
  },

  // Thresholds
  thresholds: {
    http_req_duration: ['p(95)<3000'],              // 95% of requests < 3s
    cluster_import_success: ['rate>0.90'],          // > 90% success rate
    cluster_list_latency: ['p(95)<500'],            // List should be fast
    errors: ['count<100'],                          // Less than 100 errors total
  },
};

// ============================================================
// Configuration
// ============================================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_TOKEN = __ENV.API_TOKEN || 'test-token';

// Default headers
const headers = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${API_TOKEN}`,
};

// ============================================================
// Test Data
// ============================================================

// Generate a valid kubeconfig for testing
// Note: This kubeconfig points to localhost, actual connection will fail
// but the validation and API flow will work
function generateKubeconfig(clusterName, serverUrl) {
  return `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: ${serverUrl}
    certificate-authority-data: YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=
  name: ${clusterName}
contexts:
- context:
    cluster: ${clusterName}
    user: test-user
  name: default
current-context: default
users:
- name: test-user
  token: test-token-for-performance-testing
`.trim();
}

// Generate unique cluster name
function generateClusterName() {
  return `perf-test-cluster-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

// ============================================================
// API Helper Functions
// ============================================================

/**
 * Get list of clusters
 */
function getClusters() {
  const res = http.get(`${BASE_URL}/api/v1/clusters`, { headers });

  clusterListLatency.add(res.timings.duration);

  check(res, {
    'GET /clusters status 200': (r) => r.status === 200,
    'GET /clusters has valid response': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success === true || Array.isArray(body.data);
      } catch {
        return false;
      }
    },
  });

  return res;
}

/**
 * Import a cluster
 */
function importCluster(name, kubeconfig) {
  const payload = JSON.stringify({
    name: name,
    description: 'Performance test cluster',
    kubeconfig: kubeconfig,
  });

  const startTime = new Date();
  const res = http.post(`${BASE_URL}/api/v1/clusters/import`, payload, {
    headers,
    timeout: '30s',
  });
  const duration = new Date() - startTime;

  clusterImportLatency.add(duration);

  const success = check(res, {
    'POST /clusters/import has response': (r) => r.status === 200 || r.status === 400 || r.status === 500,
    'POST /clusters/import has valid JSON': (r) => {
      try {
        JSON.parse(r.body);
        return true;
      } catch {
        return false;
      }
    },
  });

  // Success is defined as getting a valid response (connection may fail, which is expected)
  clusterImportSuccess.add(success);

  if (!success) {
    errorCount.add(1);
  }

  return res;
}

/**
 * Get cluster details
 */
function getClusterDetail(clusterId) {
  const res = http.get(`${BASE_URL}/api/v1/clusters/${clusterId}`, { headers });

  clusterDetailLatency.add(res.timings.duration);

  check(res, {
    'GET /clusters/:id status 200 or 404': (r) => r.status === 200 || r.status === 404,
  });

  return res;
}

/**
 * Test cluster connectivity
 */
function testClusterConnectivity(clusterId) {
  const res = http.post(`${BASE_URL}/api/v1/clusters/${clusterId}/test`, null, {
    headers,
    timeout: '30s',
  });

  check(res, {
    'POST /clusters/:id/test has response': (r) => r.status === 200 || r.status === 404,
  });

  return res;
}

/**
 * Delete a cluster
 */
function deleteCluster(clusterId) {
  const res = http.del(`${BASE_URL}/api/v1/clusters/${clusterId}`, null, {
    headers,
    timeout: '30s',
  });

  clusterDeleteLatency.add(res.timings.duration);

  check(res, {
    'DELETE /clusters/:id status 200 or 404': (r) => r.status === 200 || r.status === 404,
  });

  return res;
}

/**
 * Validate import parameters (without actually importing)
 */
function validateImport(name, kubeconfig) {
  const payload = JSON.stringify({
    name: name,
    kubeconfig: kubeconfig,
  });

  const res = http.post(`${BASE_URL}/api/v1/clusters/import/validate`, payload, {
    headers,
    timeout: '10s',
  });

  check(res, {
    'POST /clusters/import/validate has response': (r) => r.status === 200 || r.status === 400,
  });

  return res;
}

// ============================================================
// Test Scenarios
// ============================================================

/**
 * Smoke test - basic functionality check
 */
export function smokeTest() {
  group('Smoke Test', () => {
    console.log('Running smoke test...');

    // 1. List clusters
    group('List Clusters', () => {
      getClusters();
    });

    sleep(1);

    // 2. Validate import (no actual K8s connection)
    group('Validate Import', () => {
      const name = generateClusterName();
      const kubeconfig = generateKubeconfig(name, 'https://localhost:6443');
      validateImport(name, kubeconfig);
    });

    sleep(1);

    // 3. Attempt import (will fail connection, but tests the flow)
    group('Import Cluster', () => {
      const name = generateClusterName();
      const kubeconfig = generateKubeconfig(name, 'https://192.168.255.255:6443');
      const res = importCluster(name, kubeconfig);

      // Clean up if by chance it succeeded
      if (res.status === 200) {
        try {
          const body = JSON.parse(res.body);
          if (body.data && body.data.id) {
            deleteCluster(body.data.id);
          }
        } catch {}
      }
    });
  });
}

/**
 * Load test - normal and peak load simulation
 */
export function loadTest() {
  // Random operation selection
  const operation = Math.random();

  if (operation < 0.5) {
    // 50% - List clusters (most common operation)
    group('List Clusters', () => {
      getClusters();
    });
  } else if (operation < 0.75) {
    // 25% - Get cluster detail
    group('Get Cluster Detail', () => {
      // Try to get a random cluster ID (1-100 range for testing)
      const clusterId = Math.floor(Math.random() * 100) + 1;
      getClusterDetail(clusterId);
    });
  } else if (operation < 0.9) {
    // 15% - Validate import
    group('Validate Import', () => {
      const name = generateClusterName();
      const kubeconfig = generateKubeconfig(name, 'https://localhost:6443');
      validateImport(name, kubeconfig);
    });
  } else {
    // 10% - Import cluster (expensive operation)
    group('Import Cluster', () => {
      const name = generateClusterName();
      // Use unreachable IP to avoid actual K8s connection
      const kubeconfig = generateKubeconfig(name, 'https://192.168.255.255:6443');
      const res = importCluster(name, kubeconfig);

      // Clean up if succeeded
      if (res.status === 200) {
        try {
          const body = JSON.parse(res.body);
          if (body.data && body.data.id) {
            sleep(1);
            deleteCluster(body.data.id);
          }
        } catch {}
      }
    });
  }

  sleep(Math.random() * 2 + 0.5); // Random sleep 0.5-2.5s
}

// ============================================================
// Setup and Teardown
// ============================================================

export function setup() {
  console.log(`Starting performance test against ${BASE_URL}`);

  // Verify API is accessible
  const res = http.get(`${BASE_URL}/api/v1/clusters`, { headers });

  if (res.status !== 200 && res.status !== 401) {
    console.log(`Warning: API returned status ${res.status}`);
  }

  return { startTime: Date.now() };
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`Performance test completed in ${duration.toFixed(2)}s`);

  // Optionally: Clean up any remaining test clusters
  // This would require listing and deleting clusters with name prefix "perf-test-"
}

// ============================================================
// Helper: Handle rate limiting
// ============================================================

// Automatically retry on 429 (rate limited)
http.setResponseCallback({
  '429': (res, retry) => {
    const retryAfter = res.headers['Retry-After'] || 5;
    console.log(`Rate limited, waiting ${retryAfter}s...`);
    sleep(parseInt(retryAfter));
    retry();
  },
});
