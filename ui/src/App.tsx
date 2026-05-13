import { Routes, Route, Navigate } from 'react-router-dom';
import { SWRConfig } from 'swr';
import { AppLayout } from './components/layout/AppLayout';
import { ModelCostsPage } from './pages/ModelCosts/ModelCostsPage';
import { BudgetsPage } from './pages/Budgets/BudgetsPage';
import { BudgetDetailPage } from './pages/Budgets/BudgetDetailPage';
import { RateLimitsPage } from './pages/RateLimits/RateLimitsPage';
import { ApprovalsPage } from './pages/Approvals/ApprovalsPage';
import { AlertsPage } from './pages/Alerts/AlertsPage';
import { AuditPage } from './pages/Audit/AuditPage';
import { NotFoundPage } from './pages/NotFound/NotFoundPage';
import { swrConfig } from './hooks/useSWR';
import { config } from './config';

function App() {
  return (
    <SWRConfig value={swrConfig}>
      <AppLayout>
        <Routes>
          <Route path="/" element={<Navigate to="/budgets" replace />} />
          <Route path="/model-costs" element={<ModelCostsPage />} />
          <Route path="/budgets" element={<BudgetsPage />} />
          <Route path="/budgets/:id" element={<BudgetDetailPage />} />
          {config.enableRateLimits && <Route path="/rate-limits" element={<RateLimitsPage />} />}
          <Route path="/approvals" element={<ApprovalsPage />} />
          <Route path="/alerts" element={<AlertsPage />} />
          <Route path="/audit" element={<AuditPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </AppLayout>
    </SWRConfig>
  );
}

export default App;
