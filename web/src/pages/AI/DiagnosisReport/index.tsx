import React, { useEffect, useState } from 'react';
import { Alert, Spin } from 'antd';
import { useParams } from 'react-router-dom';
import { aiApi, type AIDiagnosisReport } from '../../../api/modules/ai';
import DiagnosisReportView from '../../../components/AIAssistant/DiagnosisReportView';

const DiagnosisReportPage: React.FC = () => {
  const { reportId = '' } = useParams();
  const [report, setReport] = useState<AIDiagnosisReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    void (async () => {
      try {
        const response = await aiApi.getDiagnosisReport(reportId);
        setReport(response.data);
      } catch {
        setFailed(true);
      } finally {
        setLoading(false);
      }
    })();
  }, [reportId]);

  if (loading) {
    return <Spin />;
  }

  if (failed || !report) {
    return <Alert type="error" message="诊断报告不可用" />;
  }

  return <DiagnosisReportView report={report} />;
};

export default DiagnosisReportPage;
