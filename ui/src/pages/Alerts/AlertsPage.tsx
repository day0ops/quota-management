// extras/quota-management/ui/src/pages/Alerts/AlertsPage.tsx
import { useState } from 'react';
import styled from '@emotion/styled';
import toast from 'react-hot-toast';
import { useSWRConfig } from 'swr';
import { spacing, colors, fontSize, radius } from '../../styles';
import { PageHeader } from '../../components/layout/PageHeader';
import { Button } from '../../components/common/Button';
import { BudgetAlert, AlertType, AlertStatus } from '../../api/types';
import { useAlerts, alertsApi } from '../../api/alerts';
import { CacheKeys } from '../../hooks/useSWR';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${spacing[4]};
`;

const TabRow = styled.div`
  display: flex;
  gap: 0;
  border-bottom: 1px solid ${colors.border};
`;

const Tab = styled.button<{ $active: boolean }>`
  padding: ${spacing[2]} ${spacing[4]};
  font-size: ${fontSize.sm};
  font-weight: ${({ $active }) => ($active ? '500' : '400')};
  color: ${({ $active }) => ($active ? colors.foreground : colors.mutedForeground)};
  background: none;
  border: none;
  border-bottom: 2px solid ${({ $active }) => ($active ? colors.primary : 'transparent')};
  cursor: pointer;
  margin-bottom: -1px;
  transition:
    color 0.15s ease,
    border-color 0.15s ease;
  &:hover {
    color: ${colors.foreground};
  }
`;

const AlertList = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${spacing[2]};
`;

const AlertRow = styled.div<{ $type: AlertType }>`
  display: flex;
  align-items: center;
  gap: ${spacing[3]};
  padding: ${spacing[3]} ${spacing[4]};
  background: ${colors.cardBg};
  border: 1px solid ${colors.border};
  border-left: 3px solid ${({ $type }) => alertColor($type)};
  border-radius: ${radius.md};
`;

const Dot = styled.div<{ $type: AlertType }>`
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  background: ${({ $type }) => alertColor($type)};
`;

const TypeBadge = styled.span<{ $type: AlertType }>`
  display: inline-block;
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: ${radius.full};
  background: ${({ $type }) => alertBgColor($type)};
  color: ${({ $type }) => alertColor($type)};
  white-space: nowrap;
  margin-bottom: ${spacing[1]};
`;

const Content = styled.div`
  flex: 1;
  min-width: 0;
`;

const Message = styled.div`
  font-size: ${fontSize.sm};
  color: ${colors.foreground};
`;

const Meta = styled.div`
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
  margin-top: 2px;
`;

const TimeAgo = styled.div`
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
  white-space: nowrap;
  flex-shrink: 0;
`;

const DismissBtn = styled.button`
  background: none;
  border: none;
  cursor: pointer;
  color: ${colors.mutedForeground};
  font-size: 14px;
  line-height: 1;
  padding: 4px 6px;
  border-radius: ${radius.sm};
  flex-shrink: 0;
  &:hover {
    color: ${colors.foreground};
    background: ${colors.border};
  }
`;

const EmptyState = styled.div`
  text-align: center;
  padding: ${spacing[12]} ${spacing[4]};
  color: ${colors.mutedForeground};
  font-size: ${fontSize.sm};
`;

function alertColor(type: AlertType): string {
  switch (type) {
    case 'budget_exhausted':
      return colors.error;
    case 'threshold_warning':
      return colors.warning;
    default:
      return colors.info;
  }
}

function alertBgColor(type: AlertType): string {
  switch (type) {
    case 'budget_exhausted':
      return colors.errorBg;
    case 'threshold_warning':
      return colors.warningBg;
    default:
      return colors.infoBg;
  }
}

function typeLabel(type: AlertType): string {
  switch (type) {
    case 'threshold_warning':
      return 'Warning';
    case 'budget_exhausted':
      return 'Exhausted';
    case 'forecast_overrun':
      return 'Forecast Overrun';
    case 'forecast_exhaustion':
      return 'Forecast';
    default:
      return '';
  }
}

function alertMessage(a: BudgetAlert): string {
  const usage = a.triggered_usage_usd.toFixed(2);
  const budget = a.budget_amount_usd.toFixed(2);
  switch (a.alert_type) {
    case 'threshold_warning':
      return `Usage reached ${a.threshold_pct ?? '?'}% ($${usage} of $${budget})`;
    case 'budget_exhausted':
      return `Budget exhausted ($${usage} of $${budget})`;
    case 'forecast_overrun':
      return `Projected spend $${a.projected_spend_usd?.toFixed(2) ?? '—'} exceeds budget $${budget}`;
    case 'forecast_exhaustion':
      return `Budget exhausts in ${a.days_until_exhausted?.toFixed(1) ?? '—'} days`;
    default:
      return '';
  }
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const s = Math.floor(diff / 1000);
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

const TABS: { label: string; value: AlertStatus | 'all' }[] = [
  { label: 'Active', value: 'active' },
  { label: 'Dismissed', value: 'dismissed' },
  { label: 'All', value: 'all' },
];

export function AlertsPage() {
  const [tab, setTab] = useState<AlertStatus | 'all'>('active');
  const { mutate } = useSWRConfig();

  const { data, loading } = useAlerts({ status: tab });
  const alerts = data?.alerts ?? [];

  const invalidate = () => {
    mutate((key: unknown) => typeof key === 'string' && key.startsWith(CacheKeys.alerts));
    mutate(CacheKeys.alertCount);
    mutate(CacheKeys.sidebarStats);
  };

  const handleDismiss = async (id: string) => {
    try {
      await alertsApi.dismiss(id);
      invalidate();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to dismiss alert';
      toast.error(message);
    }
  };

  const handleDismissAll = async () => {
    try {
      await alertsApi.dismissAll();
      invalidate();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to dismiss alerts';
      toast.error(message);
    }
  };

  return (
    <Container>
      <PageHeader description="Monitor budget threshold crossings, exhaustions, and forecast overruns">
        {tab === 'active' && alerts.length > 0 && (
          <Button variant="secondary" onClick={handleDismissAll}>
            Dismiss All
          </Button>
        )}
      </PageHeader>

      <TabRow>
        {TABS.map(t => (
          <Tab key={t.value} $active={tab === t.value} onClick={() => setTab(t.value)}>
            {t.label}
          </Tab>
        ))}
      </TabRow>

      {loading ? (
        <EmptyState>Loading...</EmptyState>
      ) : alerts.length === 0 ? (
        <EmptyState>No {tab === 'all' ? '' : tab + ' '}alerts</EmptyState>
      ) : (
        <AlertList>
          {alerts.map(a => (
            <AlertRow key={a.id} $type={a.alert_type}>
              <Dot $type={a.alert_type} />
              <Content>
                <TypeBadge $type={a.alert_type}>{typeLabel(a.alert_type)}</TypeBadge>
                <Message>{alertMessage(a)}</Message>
                <Meta>
                  {a.budget_name} · {a.entity_type}
                </Meta>
              </Content>
              <TimeAgo>{timeAgo(a.created_at)}</TimeAgo>
              {a.status === 'active' && (
                <DismissBtn onClick={() => handleDismiss(a.id)} title="Dismiss">
                  ✕
                </DismissBtn>
              )}
            </AlertRow>
          ))}
        </AlertList>
      )}
    </Container>
  );
}
