import { useState, useCallback } from 'react';
import styled from '@emotion/styled';
import toast from 'react-hot-toast';
import { spacing, colors, fontSize } from '../../styles';
import { PageHeader } from '../../components/layout/PageHeader';
import { Button } from '../../components/common/Button';
import { Badge } from '../../components/common/Badge';
import {
  TableContainer,
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell,
  EmptyState,
  EmptyStateText,
} from '../../components/common/Table';
import { Tooltip } from '../../components/common/Tooltip';
import { ConfirmDialog } from '../../components/common/ConfirmDialog';
import { Loading } from '../../components/common/Spinner';
import { Pagination } from '../../components/common/Pagination';
import { useMutation } from '../../hooks/useApi';
import { useSWRApi, CacheKeys, invalidateKey } from '../../hooks/useSWR';
import { rateLimitsApi } from '../../api/rate-limits';
import {
  RateLimitAllocation,
  CreateRateLimitRequest,
  UpdateRateLimitRequest,
} from '../../api/types';
import { ApiClientError } from '../../api/client';
import { RateLimitForm } from './RateLimitForm';
import { useAuth } from '../../contexts/AuthContext';

const Container = styled.div``;

const ActionButtons = styled.div`
  display: flex;
  gap: ${spacing[2]};
  justify-content: flex-end;
`;

const ActionButton = styled.button`
  padding: ${spacing[1]} ${spacing[2]};
  border-radius: 4px;
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
  transition: all 0.15s ease;

  &:hover {
    background: ${colors.hoverBg};
    color: ${colors.foreground};
  }
`;

const DisabledRow = styled(TableRow)<{ disabled?: boolean }>`
  opacity: ${({ disabled }) => (disabled ? 0.5 : 1)};
`;

const LimitCell = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${spacing[1]};
`;

const LimitValue = styled.span`
  font-weight: 500;
`;

const LimitUnit = styled.span`
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
`;

const PaginationWrapper = styled.div`
  display: flex;
  justify-content: flex-end;
  padding: ${spacing[4]} 0;
`;

function formatNumber(num: number): string {
  if (num >= 1000000) {
    return `${(num / 1000000).toFixed(1)}M`;
  }
  if (num >= 1000) {
    return `${(num / 1000).toFixed(1)}K`;
  }
  return num.toString();
}

function formatUnit(unit?: string): string {
  switch (unit) {
    case 'MINUTE':
      return '/min';
    case 'HOUR':
      return '/hr';
    case 'DAY':
      return '/day';
    default:
      return '';
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

function getStatusVariant(
  rateLimit: RateLimitAllocation
): 'success' | 'warning' | 'default' | 'info' | 'error' {
  if (!rateLimit.enabled) {
    return rateLimit.disabled_by_is_org ? 'error' : 'default';
  }
  if (rateLimit.approval_status === 'pending') return 'warning';
  if (rateLimit.approval_status === 'rejected') return 'default';
  if (rateLimit.enforcement === 'monitoring') return 'info';
  return 'success';
}

function getStatusLabel(rateLimit: RateLimitAllocation): string {
  if (!rateLimit.enabled) {
    return rateLimit.disabled_by_is_org ? 'Disabled (by org)' : 'Disabled';
  }
  if (rateLimit.approval_status === 'pending') return 'Pending';
  if (rateLimit.approval_status === 'rejected') return 'Rejected';
  if (rateLimit.enforcement === 'monitoring') return 'Monitoring';
  return 'Active';
}

export function RateLimitsPage() {
  const { permissions } = useAuth();
  const [formOpen, setFormOpen] = useState(false);
  const [editingRateLimit, setEditingRateLimit] = useState<RateLimitAllocation | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<RateLimitAllocation | null>(null);
  const [page, setPage] = useState(1);

  // Use SWR for rate limits list with background refresh
  const {
    data: pageData,
    loading,
    refresh,
  } = useSWRApi(`${CacheKeys.rateLimits}?page=${page}`, () => rateLimitsApi.list(page, 30), {
    refreshInterval: 15000,
  });

  const rateLimits = pageData?.data ?? null;
  const pagination = pageData?.pagination ?? null;

  const createMutation = useMutation(rateLimitsApi.create);
  const updateMutation = useMutation(
    useCallback((id: string, data: UpdateRateLimitRequest) => rateLimitsApi.update(id, data), [])
  );
  const deleteMutation = useMutation(rateLimitsApi.delete);

  const refreshAll = useCallback(() => {
    refresh();
    invalidateKey(CacheKeys.sidebarStats);
  }, [refresh]);

  const handleCreate = () => {
    setEditingRateLimit(null);
    setFormOpen(true);
  };

  const handleEdit = (rateLimit: RateLimitAllocation, e: React.MouseEvent) => {
    e.stopPropagation();
    setEditingRateLimit(rateLimit);
    setFormOpen(true);
  };

  const handleFormSubmit = async (data: CreateRateLimitRequest) => {
    try {
      if (editingRateLimit) {
        await updateMutation.execute(editingRateLimit.id, {
          ...data,
          version: editingRateLimit.version,
        });
        toast.success('Rate limit updated');
      } else {
        await createMutation.execute(data);
        toast.success('Rate limit created');
      }
      setFormOpen(false);
      refreshAll();
    } catch (error) {
      if (error instanceof ApiClientError && error.isDuplicate) {
        const msg =
          data.model_pattern === '*'
            ? 'A wildcard (*) rate limit already exists for this team. Edit the existing one or use a specific model pattern.'
            : 'A rate limit for this team and model pattern already exists. Edit the existing one or use a different model pattern.';
        toast.error(msg);
      } else if (error instanceof ApiClientError && error.isConflict) {
        toast.error('This record was modified. Please refresh and try again.');
        refresh();
      } else {
        const message = error instanceof Error ? error.message : 'An error occurred';
        toast.error(message);
      }
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await deleteMutation.execute(deleteTarget.id);
      toast.success('Rate limit deleted');
      setDeleteTarget(null);
      refreshAll();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'An error occurred';
      toast.error(message);
    }
  };

  const handleDeleteClick = (rateLimit: RateLimitAllocation, e: React.MouseEvent) => {
    e.stopPropagation();
    setDeleteTarget(rateLimit);
  };

  const handleToggleEnabled = async (rateLimit: RateLimitAllocation, e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await updateMutation.execute(rateLimit.id, {
        enabled: !rateLimit.enabled,
        version: rateLimit.version,
      });
      toast.success(rateLimit.enabled ? 'Rate limit disabled' : 'Rate limit enabled');
      refreshAll();
    } catch (error) {
      if (error instanceof ApiClientError && error.isConflict) {
        toast.error('This record was modified. Please refresh and try again.');
        refresh();
      } else {
        const message = error instanceof Error ? error.message : 'An error occurred';
        toast.error(message);
      }
    }
  };

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
  };

  return (
    <Container>
      <PageHeader description="Manage team rate limit allocations">
        <Button variant="secondary" onClick={refresh}>
          Refresh
        </Button>
        {permissions.canCreateBudgets && <Button onClick={handleCreate}>Create Rate Limit</Button>}
      </PageHeader>

      {loading ? (
        <Loading />
      ) : rateLimits && rateLimits.length > 0 ? (
        <>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>
                    <Tooltip text="Team this rate limit applies to">Team</Tooltip>
                  </TableHeader>
                  <TableHeader>
                    <Tooltip text="Model pattern for matching (e.g., gpt-4*, *)">Model</Tooltip>
                  </TableHeader>
                  <TableHeader>
                    <Tooltip text="Token limit per time unit">Token Limit</Tooltip>
                  </TableHeader>
                  <TableHeader>
                    <Tooltip text="Request limit per time unit">Request Limit</Tooltip>
                  </TableHeader>
                  <TableHeader>
                    <Tooltip text="Current status of this rate limit">Status</Tooltip>
                  </TableHeader>
                  <TableHeader>Created</TableHeader>
                  <TableHeader align="right">Actions</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {rateLimits.map(rateLimit => (
                  <DisabledRow key={rateLimit.id} disabled={!rateLimit.enabled}>
                    <TableCell>{rateLimit.team_id}</TableCell>
                    <TableCell>
                      <code style={{ fontSize: fontSize.xs }}>{rateLimit.model_pattern}</code>
                    </TableCell>
                    <TableCell>
                      {rateLimit.token_limit ? (
                        <LimitCell>
                          <LimitValue>{formatNumber(rateLimit.token_limit)}</LimitValue>
                          <LimitUnit>{formatUnit(rateLimit.token_unit)}</LimitUnit>
                        </LimitCell>
                      ) : (
                        <span style={{ color: colors.mutedForeground }}>-</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {rateLimit.request_limit ? (
                        <LimitCell>
                          <LimitValue>{formatNumber(rateLimit.request_limit)}</LimitValue>
                          <LimitUnit>{formatUnit(rateLimit.request_unit)}</LimitUnit>
                        </LimitCell>
                      ) : (
                        <span style={{ color: colors.mutedForeground }}>-</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant={getStatusVariant(rateLimit)}>
                        {getStatusLabel(rateLimit)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <span style={{ fontSize: fontSize.xs, color: colors.mutedForeground }}>
                        {formatDate(rateLimit.created_at)}
                      </span>
                    </TableCell>
                    <TableCell align="right">
                      <ActionButtons>
                        {rateLimit.approval_status === 'approved' &&
                          rateLimit.can_enable !== false && (
                            <ActionButton onClick={e => handleToggleEnabled(rateLimit, e)}>
                              {rateLimit.enabled ? 'Disable' : 'Enable'}
                            </ActionButton>
                          )}
                        <ActionButton onClick={e => handleEdit(rateLimit, e)}>Edit</ActionButton>
                        {permissions.canDeleteBudgets && (
                          <ActionButton onClick={e => handleDeleteClick(rateLimit, e)}>
                            Delete
                          </ActionButton>
                        )}
                      </ActionButtons>
                    </TableCell>
                  </DisabledRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
          {pagination && pagination.total_pages > 1 && (
            <PaginationWrapper>
              <Pagination
                page={pagination.page}
                totalPages={pagination.total_pages}
                onPageChange={handlePageChange}
              />
            </PaginationWrapper>
          )}
        </>
      ) : (
        <EmptyState>
          <EmptyStateText>No rate limits configured yet.</EmptyStateText>
          {permissions.canCreateBudgets && (
            <Button onClick={handleCreate}>Create Rate Limit</Button>
          )}
        </EmptyState>
      )}

      <RateLimitForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onSubmit={handleFormSubmit}
        editingRateLimit={editingRateLimit}
        loading={createMutation.loading || updateMutation.loading}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete Rate Limit"
        message={`Are you sure you want to delete the rate limit for team "${deleteTarget?.team_id}"? This action cannot be undone.`}
        confirmLabel="Delete"
        loading={deleteMutation.loading}
      />
    </Container>
  );
}
