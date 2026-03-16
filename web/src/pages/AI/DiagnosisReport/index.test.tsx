import React from 'react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DiagnosisReportPage from './index';

const aiMocks = vi.hoisted(() => ({
  getDiagnosisReport: vi.fn(),
}));

vi.mock('../../../api/modules/ai', () => ({
  aiApi: {
    getDiagnosisReport: aiMocks.getDiagnosisReport,
  },
}));

describe('DiagnosisReportPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders summary, evidence, root causes, and recommendations', async () => {
    aiMocks.getDiagnosisReport.mockResolvedValue({
      data: {
        report_id: 'report-1',
        summary: 'Quota exhausted',
        evidence: ['events show quota exceeded'],
        root_causes: ['namespace quota exhausted'],
        recommendations: ['increase quota'],
      },
    });

    render(
      <MemoryRouter initialEntries={['/ai/diagnosis/report-1']}>
        <Routes>
          <Route path="/ai/diagnosis/:reportId" element={<DiagnosisReportPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText('Quota exhausted')).toBeInTheDocument();
    expect(await screen.findByText('events show quota exceeded')).toBeInTheDocument();
    expect(await screen.findByText('namespace quota exhausted')).toBeInTheDocument();
    expect(await screen.findByText('increase quota')).toBeInTheDocument();
  });

  it('handles missing or failed report states', async () => {
    aiMocks.getDiagnosisReport.mockRejectedValue(new Error('missing'));

    render(
      <MemoryRouter initialEntries={['/ai/diagnosis/missing']}>
        <Routes>
          <Route path="/ai/diagnosis/:reportId" element={<DiagnosisReportPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText('诊断报告不可用')).toBeInTheDocument();
  });
});
