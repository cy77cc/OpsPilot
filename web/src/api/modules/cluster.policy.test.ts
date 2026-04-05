import { afterEach, describe, expect, it, vi } from 'vitest';
import apiService from '../api';
import { clusterApi } from './cluster';

describe('clusterApi policy contract', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('normalizes policy simulation payloads and errors into blocking issues', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: {
        simulation_result: {
          passed: false,
          errors: [
            {
              error_code: 'SIMULATION_BLOCKING_CONFLICT',
              message: 'critical namespace would be blocked',
              level: 'blocking',
              remediation: 'add an allow rule',
            },
          ],
          warnings: ['adapter downgraded one L7 rule'],
          impactSummary: {
            affectedPods: 12,
            affectedNamespaces: ['prod', 'payments'],
            newDeniedFlows: ['api -> kube-dns'],
          },
          riskScore: 72,
          riskLevel: 'CRITICAL',
        },
      },
    });

    const simulatePolicy = (clusterApi as any).simulatePolicy;
    expect(typeof simulatePolicy).toBe('function');

    const response = await simulatePolicy(42, 'prod', 'allow-api', {
      base_version: 'stable-v1',
      candidate_version: 'candidate-v2',
    });

    expect(postMock).toHaveBeenCalledWith(
      '/clusters/42/policies/prod/allow-api/simulate',
      {
        base_version: 'stable-v1',
        candidate_version: 'candidate-v2',
      },
    );
    expect(response.data).toEqual({
      passed: false,
      blocking_issues: [
        {
          code: 'SIMULATION_BLOCKING_CONFLICT',
          message: 'critical namespace would be blocked',
          severity: 'BLOCKING',
          suggestion: 'add an allow rule',
        },
      ],
      warnings: [
        {
          code: undefined,
          message: 'adapter downgraded one L7 rule',
        },
      ],
      impact_summary: {
        affected_pods: 12,
        affected_namespaces: ['prod', 'payments'],
        new_denied_flows: ['api -> kube-dns'],
      },
      risk_score: 72,
      risk_level: 'CRITICAL',
    });
  });

  it('unwraps and normalizes policy release detail payloads', async () => {
    const getMock = vi.spyOn(apiService, 'get').mockResolvedValue({
      success: true,
      data: {
        release: {
          release_id: 501,
          version: 'candidate-v2',
          previous_stable_version: 'stable-v1',
          policy: {
            apiVersion: 'ops/v1',
            kind: 'NetworkPolicyDefinition',
            name: 'allow-api',
            namespace: 'prod',
          },
          target_cluster: {
            clusterId: 42,
            cniType: 'cilium',
            cniVersion: '1.17.0',
          },
          status: {
            phase: 'simulation_passed',
            riskScore: 35,
            riskLevel: 'MEDIUM',
          },
          Simulation: {
            jobId: 'sim-42',
            blockingIssues: [],
            warnings: [{ code: 'L7_SIMPLIFIED', message: 'translated with downgrade' }],
            impactSummary: {
              affectedPods: 8,
              affectedNamespaces: ['prod'],
              newDeniedFlows: ['api -> db'],
            },
          },
          approval: {
            required: true,
            approvalToken: 'ticket-1',
          },
          audit: {
            createdAt: '2026-04-05T12:00:00Z',
            createdBy: 1001,
          },
          lastErrorCode: 'APPLY_FAILED',
          last_error_message: 'adapter refused the rule',
        },
      },
    });

    const getPolicyRelease = (clusterApi as any).getPolicyRelease;
    expect(typeof getPolicyRelease).toBe('function');

    const response = await getPolicyRelease(42, 501);

    expect(getMock).toHaveBeenCalledWith('/clusters/42/releases/501');
    expect(response.data).toEqual({
      release_id: 501,
      version: 'candidate-v2',
      previous_stable_version: 'stable-v1',
      policy: {
        api_version: 'ops/v1',
        kind: 'NetworkPolicyDefinition',
        name: 'allow-api',
        namespace: 'prod',
      },
      target_cluster: {
        cluster_id: 42,
        cni_type: 'cilium',
        cni_version: '1.17.0',
      },
      status: {
        phase: 'simulation_passed',
        risk_score: 35,
        risk_level: 'MEDIUM',
      },
      simulation: {
        job_id: 'sim-42',
        blocking_issues: [],
        warnings: [{ code: 'L7_SIMPLIFIED', message: 'translated with downgrade' }],
        impact_summary: {
          affected_pods: 8,
          affected_namespaces: ['prod'],
          new_denied_flows: ['api -> db'],
        },
      },
      approval: {
        required: true,
        approval_token: 'ticket-1',
      },
      audit: {
        created_at: '2026-04-05T12:00:00Z',
        created_by: 1001,
      },
      last_error_code: 'APPLY_FAILED',
      last_error_message: 'adapter refused the rule',
    });
  });

  it('keeps the unified operation envelope while normalizing nested policy release results', async () => {
    const postMock = vi.spyOn(apiService, 'post').mockResolvedValue({
      success: true,
      data: {
        state: 'approval_required',
        code: 'approval_required',
        message: 'needs approval',
        approval: {
          required: true,
          ticket: 'k8s-appr-1',
        },
        data: {
          release: {
            release_id: 501,
            version: 'candidate-v2',
            previous_stable_version: 'stable-v1',
            policy: {
              apiVersion: 'ops/v1',
              kind: 'NetworkPolicyDefinition',
              name: 'allow-api',
              namespace: 'prod',
            },
            target_cluster: {
              clusterId: 42,
              cniType: 'cilium',
            },
            status: {
              phase: 'approval_required',
              riskScore: 35,
              riskLevel: 'MEDIUM',
            },
            Simulation: {
              warnings: ['adapter downgraded one L7 rule'],
              errors: [
                {
                  code: 'SIMULATION_BLOCKING_CONFLICT',
                  message: 'critical namespace would be blocked',
                  severity: 'BLOCKING',
                },
              ],
              impact_summary: {
                affected_pods: 12,
                affected_namespaces: ['prod'],
                new_denied_flows: ['api -> kube-dns'],
              },
            },
          },
        },
      },
    });

    const createPolicyRelease = (clusterApi as any).createPolicyRelease;
    expect(typeof createPolicyRelease).toBe('function');

    const response = await createPolicyRelease(42, 'prod', 'allow-api', {
      version: 'candidate-v2',
      previous_stable_version: 'stable-v1',
    });

    expect(postMock).toHaveBeenCalledWith(
      '/clusters/42/policies/prod/allow-api/releases',
      {
        version: 'candidate-v2',
        previous_stable_version: 'stable-v1',
      },
    );
    expect(response.data.state).toBe('approval_required');
    expect(response.data.approval?.ticket).toBe('k8s-appr-1');
    expect(response.data.result).toEqual({
      release: {
        release_id: 501,
        version: 'candidate-v2',
        previous_stable_version: 'stable-v1',
        policy: {
          api_version: 'ops/v1',
          kind: 'NetworkPolicyDefinition',
          name: 'allow-api',
          namespace: 'prod',
        },
        target_cluster: {
          cluster_id: 42,
          cni_type: 'cilium',
        },
        status: {
          phase: 'approval_required',
          risk_score: 35,
          risk_level: 'MEDIUM',
        },
        simulation: {
          blocking_issues: [
            {
              code: 'SIMULATION_BLOCKING_CONFLICT',
              message: 'critical namespace would be blocked',
              severity: 'BLOCKING',
            },
          ],
          warnings: [
            {
              code: undefined,
              message: 'adapter downgraded one L7 rule',
            },
          ],
          impact_summary: {
            affected_pods: 12,
            affected_namespaces: ['prod'],
            new_denied_flows: ['api -> kube-dns'],
          },
        },
      },
    });
  });

  it('loads normalized cluster cni info for policy capability gates', async () => {
    const getMock = vi.spyOn(apiService, 'get').mockResolvedValue({
      success: true,
      data: {
        cluster_id: 42,
        cni_type: 'flannel',
        cni_version: '0.25.5',
        capabilities: {
          standard_np: true,
          publishable: false,
        },
        constraints: {
          netpol_enabled: false,
          publish_block_reason: 'FLANNEL_NETPOL_DISABLED',
        },
      },
    });

    const getClusterCNIInfo = (clusterApi as any).getClusterCNIInfo;
    expect(typeof getClusterCNIInfo).toBe('function');

    const response = await getClusterCNIInfo(42);

    expect(getMock).toHaveBeenCalledWith('/clusters/42/cni-info');
    expect(response.data).toEqual({
      cluster_id: 42,
      cni_type: 'flannel',
      cni_version: '0.25.5',
      capabilities: {
        standard_np: true,
        publishable: false,
      },
      constraints: {
        netpol_enabled: false,
        publish_block_reason: 'FLANNEL_NETPOL_DISABLED',
      },
    });
  });
});
