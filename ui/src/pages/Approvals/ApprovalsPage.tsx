import { useState, useCallback, useEffect, useRef } from 'react';
import styled from '@emotion/styled';
import toast from 'react-hot-toast';
import { colors, spacing, fontSize, radius } from '../../styles';
import { PageHeader } from '../../components/layout/PageHeader';
import { Button } from '../../components/common/Button';
import { Badge } from '../../components/common/Badge';
import { Pagination } from '../../components/common/Pagination';
import { Loading } from '../../components/common/Spinner';
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
import { useMutation } from '../../hooks/useApi';
import { useSWRApi, CacheKeys, invalidateKey } from '../../hooks/useSWR';
import { approvalsApi } from '../../api/approvals';
import { rateLimitsApi } from '../../api/rate-limits';
import { useAuth } from '../../contexts/AuthContext';
import type {
  ApprovalWithBudget,
  RateLimitAllocation,
  RateLimitApprovalWithAllocation,
} from '../../api/types';

const Container = styled.div``;

const TabBar = styled.div`
  display: flex;
  gap: 0;
  border-bottom: 1px solid ${colors.border};
  margin-bottom: ${spacing[6]};
`;

const Tab = styled.button<{ active: boolean }>`
  padding: ${spacing[3]} ${spacing[4]};
  font-size: ${fontSize.sm};
  color: ${({ active }) => (active ? colors.foreground : colors.mutedForeground)};
  background: transparent;
  border: none;
  border-bottom: 2px solid ${({ active }) => (active ? colors.primary : 'transparent')};
  cursor: pointer;
  transition: all 0.15s ease;
  margin-bottom: -1px;

  &:hover {
    color: ${colors.foreground};
  }
`;

const ActionButtons = styled.div`
  display: flex;
  gap: ${spacing[2]};
  justify-content: flex-end;
`;

const ApproveButton = styled.button`
  padding: ${spacing[1]} ${spacing[3]};
  border-radius: ${radius.sm};
  font-size: ${fontSize.xs};
  font-weight: 500;
  background: ${colors.successBg};
  color: ${colors.success};
  border: 1px solid ${colors.successBorder};
  cursor: pointer;
  transition: all 0.15s ease;

  &:hover {
    background: ${colors.success};
    color: #fff;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`;

const RejectButton = styled.button`
  padding: ${spacing[1]} ${spacing[3]};
  border-radius: ${radius.sm};
  font-size: ${fontSize.xs};
  font-weight: 500;
  background: ${colors.errorBg};
  color: ${colors.error};
  border: 1px solid ${colors.errorBorder};
  cursor: pointer;
  transition: all 0.15s ease;

  &:hover {
    background: ${colors.error};
    color: #fff;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`;

const ResubmitButton = styled.button`
  padding: ${spacing[1]} ${spacing[3]};
  border-radius: ${radius.sm};
  font-size: ${fontSize.xs};
  font-weight: 500;
  background: ${colors.infoBg};
  color: ${colors.info};
  border: 1px solid ${colors.infoBorder};
  cursor: pointer;
  transition: all 0.15s ease;

  &:hover {
    background: ${colors.info};
    color: #fff;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`;

const RejectionReason = styled.div`
  font-size: ${fontSize.xs};
  color: ${colors.error};
  margin-top: ${spacing[1]};
  font-style: italic;
`;

const PaginationWrapper = styled.div`
  display: flex;
  justify-content: flex-end;
  padding: ${spacing[4]} 0;
`;

const SectionHeader = styled.h3`
  font-size: ${fontSize.sm};
  font-weight: 600;
  color: ${colors.foreground};
  margin: ${spacing[6]} 0 ${spacing[3]};
  display: flex;
  align-items: center;
  gap: ${spacing[2]};

  &:first-of-type {
    margin-top: 0;
  }
`;

const TypeBadge = styled.span<{ type: 'budget' | 'rate_limit' }>`
  display: inline-flex;
  align-items: center;
  padding: ${spacing[1]} ${spacing[2]};
  border-radius: ${radius.sm};
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background: ${({ type }) => (type === 'budget' ? colors.warningBg : colors.infoBg)};
  color: ${({ type }) => (type === 'budget' ? colors.warning : colors.info)};
  border: 1px solid ${({ type }) => (type === 'budget' ? colors.warningBorder : colors.infoBorder)};
`;

// Reject Modal
const ModalOverlay = styled.div`
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
`;

const ModalCard = styled.div`
  background: ${colors.cardBg};
  border: 1px solid ${colors.border};
  border-radius: ${radius.xl};
  padding: ${spacing[6]};
  width: 100%;
  max-width: 480px;
`;

const ModalTitle = styled.h3`
  font-size: ${fontSize.md};
  font-weight: 500;
  color: ${colors.foreground};
  margin: 0 0 ${spacing[4]};
`;

const ModalLabel = styled.label`
  display: block;
  font-size: ${fontSize.sm};
  color: ${colors.mutedForeground};
  margin-bottom: ${spacing[2]};
`;

const ReasonTextArea = styled.textarea`
  width: 100%;
  min-height: 100px;
  background: ${colors.surfaceBg};
  border: 1px solid ${colors.border};
  border-radius: ${radius.md};
  color: ${colors.foreground};
  font-size: ${fontSize.sm};
  padding: ${spacing[3]};
  resize: vertical;
  box-sizing: border-box;
  font-family: inherit;

  &:focus {
    outline: none;
    border-color: ${colors.primary};
  }

  &::placeholder {
    color: ${colors.dimForeground};
  }
`;

const ModalActions = styled.div`
  display: flex;
  justify-content: flex-end;
  gap: ${spacing[3]};
  margin-top: ${spacing[5]};
`;

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}

function formatCurrency(amount: number): string {
  if (amount === 0) return '$0.00';
  if (amount >= 1) return `$${amount.toFixed(2)}`;
  if (amount >= 0.01) return `$${amount.toFixed(4)}`;
  if (amount >= 0.001) return `$${amount.toFixed(5)}`;
  return `$${amount.toFixed(6)}`;
}

interface RejectModalProps {
  itemId: string;
  itemName: string;
  itemType: 'budget' | 'rate_limit';
  onClose: () => void;
  onReject: (id: string, reason: string) => Promise<void>;
  loading: boolean;
}

function RejectModal({ itemId, itemName, itemType, onClose, onReject, loading }: RejectModalProps) {
  const [reason, setReason] = useState('');

  const handleSubmit = async () => {
    if (!reason.trim()) return;
    await onReject(itemId, reason.trim());
  };

  const typeLabel = itemType === 'budget' ? 'Budget' : 'Rate Limit';

  return (
    <ModalOverlay onClick={onClose}>
      <ModalCard onClick={e => e.stopPropagation()}>
        <ModalTitle>
          Reject {typeLabel}: {itemName}
        </ModalTitle>
        <ModalLabel htmlFor="reject-reason">Reason (required)</ModalLabel>
        <ReasonTextArea
          id="reject-reason"
          placeholder="Provide a reason for rejection..."
          value={reason}
          onChange={e => setReason(e.target.value)}
          autoFocus
        />
        <ModalActions>
          <Button variant="secondary" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button variant="danger" onClick={handleSubmit} disabled={!reason.trim() || loading}>
            {loading ? 'Rejecting...' : 'Reject'}
          </Button>
        </ModalActions>
      </ModalCard>
    </ModalOverlay>
  );
}

interface OrgPendingTabProps {
  budgetItems: ApprovalWithBudget[];
  rateLimitItems: RateLimitAllocation[];
  onRefresh: () => void;
}

function OrgPendingTab({ budgetItems, rateLimitItems, onRefresh }: OrgPendingTabProps) {
  const [rejectTarget, setRejectTarget] = useState<{
    id: string;
    name: string;
    type: 'budget' | 'rate_limit';
  } | null>(null);

  const budgetApproveMutation = useMutation(
    useCallback((id: string) => approvalsApi.approve(id), [])
  );
  const budgetRejectMutation = useMutation(
    useCallback((id: string, reason: string) => approvalsApi.reject(id, reason), [])
  );
  const rateLimitApproveMutation = useMutation(
    useCallback((id: string) => rateLimitsApi.approve(id), [])
  );
  const rateLimitRejectMutation = useMutation(
    useCallback((id: string, reason?: string) => rateLimitsApi.reject(id, reason), [])
  );

  const handleApproveBudget = async (item: ApprovalWithBudget) => {
    try {
      await budgetApproveMutation.execute(item.budget_id);
      toast.success(`Budget "${item.budget_name}" approved`);
      onRefresh();
      invalidateKey(CacheKeys.sidebarStats);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to approve';
      toast.error(message);
    }
  };

  const handleApproveRateLimit = async (item: RateLimitAllocation) => {
    try {
      await rateLimitApproveMutation.execute(item.id);
      toast.success(`Rate limit for "${item.team_id}" approved`);
      onRefresh();
      invalidateKey(CacheKeys.sidebarStats);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to approve';
      toast.error(message);
    }
  };

  const handleRejectConfirm = async (id: string, reason: string) => {
    const target = rejectTarget;
    if (!target) return;

    try {
      if (target.type === 'budget') {
        await budgetRejectMutation.execute(id, reason);
        toast.success(`Budget "${target.name}" rejected`);
      } else {
        await rateLimitRejectMutation.execute(id, reason);
        toast.success(`Rate limit for "${target.name}" rejected`);
      }
      setRejectTarget(null);
      onRefresh();
      invalidateKey(CacheKeys.sidebarStats);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to reject';
      toast.error(message);
    }
  };

  const isLoading =
    budgetApproveMutation.loading ||
    budgetRejectMutation.loading ||
    rateLimitApproveMutation.loading ||
    rateLimitRejectMutation.loading;

  const hasBudgets = budgetItems.length > 0;
  const hasRateLimits = rateLimitItems.length > 0;

  if (!hasBudgets && !hasRateLimits) {
    return (
      <EmptyState>
        <EmptyStateText>No pending approvals.</EmptyStateText>
      </EmptyState>
    );
  }

  return (
    <>
      {hasBudgets && (
        <>
          <SectionHeader>
            <TypeBadge type="budget">Budget</TypeBadge>
            Pending Budget Approvals ({budgetItems.length})
          </SectionHeader>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Budget Name</TableHeader>
                  <TableHeader>Team</TableHeader>
                  <TableHeader>Requested By</TableHeader>
                  <TableHeader align="right">Amount</TableHeader>
                  <TableHeader>Period</TableHeader>
                  <TableHeader>Submitted</TableHeader>
                  <TableHeader align="right">Actions</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {budgetItems.map(item => (
                  <TableRow key={item.id}>
                    <TableCell>{item.budget_name}</TableCell>
                    <TableCell>
                      {item.owner_team_id ? (
                        <Badge variant="default">{item.owner_team_id}</Badge>
                      ) : (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          —
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      {item.created_by_email || (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          —
                        </span>
                      )}
                    </TableCell>
                    <TableCell align="right">{formatCurrency(item.budget_amount_usd)}</TableCell>
                    <TableCell style={{ textTransform: 'capitalize' }}>
                      {item.budget_period}
                    </TableCell>
                    <TableCell>{formatDate(item.created_at)}</TableCell>
                    <TableCell align="right">
                      <ActionButtons>
                        <ApproveButton
                          onClick={() => handleApproveBudget(item)}
                          disabled={isLoading}
                        >
                          Approve
                        </ApproveButton>
                        <RejectButton
                          onClick={() =>
                            setRejectTarget({
                              id: item.budget_id,
                              name: item.budget_name,
                              type: 'budget',
                            })
                          }
                          disabled={isLoading}
                        >
                          Reject
                        </RejectButton>
                      </ActionButtons>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </>
      )}

      {hasRateLimits && (
        <>
          <SectionHeader>
            <TypeBadge type="rate_limit">Rate Limit</TypeBadge>
            Pending Rate Limit Approvals ({rateLimitItems.length})
          </SectionHeader>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Team</TableHeader>
                  <TableHeader>Model Pattern</TableHeader>
                  <TableHeader>Token Limit</TableHeader>
                  <TableHeader>Request Limit</TableHeader>
                  <TableHeader>Requested By</TableHeader>
                  <TableHeader>Submitted</TableHeader>
                  <TableHeader align="right">Actions</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {rateLimitItems.map(item => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <Badge variant="default">{item.team_id}</Badge>
                    </TableCell>
                    <TableCell>
                      <code style={{ fontSize: fontSize.xs }}>{item.model_pattern}</code>
                    </TableCell>
                    <TableCell>
                      {item.token_limit != null ? (
                        <>
                          {item.token_limit.toLocaleString()} / {item.token_unit?.toLowerCase()}
                        </>
                      ) : (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          —
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      {item.request_limit != null ? (
                        <>
                          {item.request_limit.toLocaleString()} / {item.request_unit?.toLowerCase()}
                        </>
                      ) : (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          —
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      {item.created_by_email || (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          —
                        </span>
                      )}
                    </TableCell>
                    <TableCell>{formatDate(item.created_at)}</TableCell>
                    <TableCell align="right">
                      <ActionButtons>
                        <ApproveButton
                          onClick={() => handleApproveRateLimit(item)}
                          disabled={isLoading}
                        >
                          Approve
                        </ApproveButton>
                        <RejectButton
                          onClick={() =>
                            setRejectTarget({
                              id: item.id,
                              name: item.team_id,
                              type: 'rate_limit',
                            })
                          }
                          disabled={isLoading}
                        >
                          Reject
                        </RejectButton>
                      </ActionButtons>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </>
      )}

      {rejectTarget && (
        <RejectModal
          itemId={rejectTarget.id}
          itemName={rejectTarget.name}
          itemType={rejectTarget.type}
          onClose={() => setRejectTarget(null)}
          onReject={handleRejectConfirm}
          loading={budgetRejectMutation.loading || rateLimitRejectMutation.loading}
        />
      )}
    </>
  );
}

interface TeamPendingTabProps {
  budgetItems: ApprovalWithBudget[];
  rateLimitItems: RateLimitAllocation[];
  onRefresh: () => void;
}

function TeamPendingTab({ budgetItems, rateLimitItems, onRefresh }: TeamPendingTabProps) {
  const resubmitMutation = useMutation(useCallback((id: string) => approvalsApi.resubmit(id), []));

  const handleResubmit = async (item: ApprovalWithBudget) => {
    try {
      await resubmitMutation.execute(item.budget_id);
      toast.success(`Budget "${item.budget_name}" resubmitted`);
      onRefresh();
      invalidateKey(CacheKeys.sidebarStats);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to resubmit';
      toast.error(message);
    }
  };

  const getStatusVariant = (status: string): 'info' | 'error' | 'warning' => {
    if (status === 'rejected') return 'error';
    if (status === 'pending') return 'warning';
    return 'info';
  };

  const hasBudgets = budgetItems.length > 0;
  const hasRateLimits = rateLimitItems.length > 0;

  if (!hasBudgets && !hasRateLimits) {
    return (
      <EmptyState>
        <EmptyStateText>No pending approvals.</EmptyStateText>
      </EmptyState>
    );
  }

  return (
    <>
      {hasBudgets && (
        <>
          <SectionHeader>
            <TypeBadge type="budget">Budget</TypeBadge>
            Your Budget Requests ({budgetItems.length})
          </SectionHeader>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Budget Name</TableHeader>
                  <TableHeader>Status</TableHeader>
                  <TableHeader align="right">Amount</TableHeader>
                  <TableHeader>Period</TableHeader>
                  <TableHeader>Submitted</TableHeader>
                  <TableHeader align="right">Actions</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {budgetItems.map(item => (
                  <TableRow key={item.id}>
                    <TableCell>{item.budget_name}</TableCell>
                    <TableCell>
                      <Badge variant={getStatusVariant(item.approval_status)}>
                        {item.approval_status}
                      </Badge>
                      {item.approval_status === 'rejected' && item.reason && (
                        <RejectionReason>{item.reason}</RejectionReason>
                      )}
                    </TableCell>
                    <TableCell align="right">{formatCurrency(item.budget_amount_usd)}</TableCell>
                    <TableCell style={{ textTransform: 'capitalize' }}>
                      {item.budget_period}
                    </TableCell>
                    <TableCell>{formatDate(item.created_at)}</TableCell>
                    <TableCell align="right">
                      {item.approval_status === 'rejected' && (
                        <ResubmitButton
                          onClick={() => handleResubmit(item)}
                          disabled={resubmitMutation.loading}
                        >
                          Resubmit
                        </ResubmitButton>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </>
      )}

      {hasRateLimits && (
        <>
          <SectionHeader>
            <TypeBadge type="rate_limit">Rate Limit</TypeBadge>
            Your Rate Limit Requests ({rateLimitItems.length})
          </SectionHeader>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Team</TableHeader>
                  <TableHeader>Model Pattern</TableHeader>
                  <TableHeader>Status</TableHeader>
                  <TableHeader>Token Limit</TableHeader>
                  <TableHeader>Request Limit</TableHeader>
                  <TableHeader>Submitted</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {rateLimitItems.map(item => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <Badge variant="default">{item.team_id}</Badge>
                    </TableCell>
                    <TableCell>
                      <code style={{ fontSize: fontSize.xs }}>{item.model_pattern}</code>
                    </TableCell>
                    <TableCell>
                      <Badge variant={getStatusVariant(item.approval_status)}>
                        {item.approval_status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {item.token_limit != null ? (
                        <>
                          {item.token_limit.toLocaleString()} / {item.token_unit?.toLowerCase()}
                        </>
                      ) : (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          —
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      {item.request_limit != null ? (
                        <>
                          {item.request_limit.toLocaleString()} / {item.request_unit?.toLowerCase()}
                        </>
                      ) : (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          —
                        </span>
                      )}
                    </TableCell>
                    <TableCell>{formatDate(item.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </>
      )}
    </>
  );
}

interface HistoryItem {
  id: string;
  type: 'budget' | 'rate_limit';
  name: string;
  team_id?: string;
  action: string;
  actor_email?: string;
  actor_user_id?: string;
  reason?: string;
  created_at: string;
}

interface HistoryTabProps {
  pageSize: number;
  refreshTrigger?: number;
}

function HistoryTab({ pageSize, refreshTrigger }: HistoryTabProps) {
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<HistoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [totalPages, setTotalPages] = useState(1);

  const fetchHistory = useCallback(async () => {
    setLoading(true);
    try {
      // Fetch both budget approvals and rate limit approvals from native APIs
      const [budgetData, rateLimitData] = await Promise.all([
        approvalsApi.history(page, pageSize),
        rateLimitsApi.history(page, pageSize),
      ]);

      // Convert budget approvals to unified format
      const budgetItems: HistoryItem[] = (budgetData.data ?? []).map(
        (item: ApprovalWithBudget) => ({
          id: `budget-${item.id}`,
          type: 'budget' as const,
          name: item.budget_name,
          team_id: item.owner_team_id,
          action: item.action,
          actor_email: item.actor_email,
          actor_user_id: item.actor_user_id,
          reason: item.reason,
          created_at: item.created_at,
        })
      );

      // Convert rate limit approvals to unified format
      const rateLimitItems: HistoryItem[] = (rateLimitData.data ?? []).map(
        (item: RateLimitApprovalWithAllocation) => ({
          id: `ratelimit-${item.id}`,
          type: 'rate_limit' as const,
          name: `${item.team_id} / ${item.model_pattern}`,
          team_id: item.team_id,
          action: item.action,
          actor_email: item.actor_email,
          actor_user_id: item.actor_user_id,
          reason: item.reason,
          created_at: item.created_at,
        })
      );

      // Combine and sort by date (newest first)
      const combined = [...budgetItems, ...rateLimitItems].sort(
        (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      );

      // Simple pagination on combined results
      const startIndex = (page - 1) * pageSize;
      const paginatedItems = combined.slice(startIndex, startIndex + pageSize);

      setItems(paginatedItems);
      setTotalPages(Math.ceil(combined.length / pageSize) || 1);
    } catch {
      setItems([]);
      setTotalPages(1);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => {
    fetchHistory();
  }, [fetchHistory]);

  const isInitialMount = useRef(true);
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }
    fetchHistory();
  }, [refreshTrigger]);

  return (
    <>
      {loading ? (
        <Loading />
      ) : items.length > 0 ? (
        <>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Type</TableHeader>
                  <TableHeader>Name</TableHeader>
                  <TableHeader>Team</TableHeader>
                  <TableHeader>Action</TableHeader>
                  <TableHeader>By</TableHeader>
                  <TableHeader>Reason</TableHeader>
                  <TableHeader>Timestamp</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {items.map(item => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <TypeBadge type={item.type}>
                        {item.type === 'budget' ? 'Budget' : 'Rate Limit'}
                      </TypeBadge>
                    </TableCell>
                    <TableCell>{item.name}</TableCell>
                    <TableCell>
                      {item.team_id ? (
                        <Badge variant="default">{item.team_id}</Badge>
                      ) : (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          —
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          item.action === 'approved'
                            ? 'success'
                            : item.action === 'rejected'
                              ? 'error'
                              : 'default'
                        }
                      >
                        {item.action}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {item.actor_email || (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          {item.actor_user_id || '—'}
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      {item.reason || (
                        <span style={{ color: colors.dimForeground, fontSize: fontSize.xs }}>
                          —
                        </span>
                      )}
                    </TableCell>
                    <TableCell>{formatDate(item.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
          <PaginationWrapper>
            <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
          </PaginationWrapper>
        </>
      ) : (
        <EmptyState>
          <EmptyStateText>No approval history found.</EmptyStateText>
        </EmptyState>
      )}
    </>
  );
}

type ActiveTab = 'pending' | 'history';

const HISTORY_PAGE_SIZE = 30;

export function ApprovalsPage() {
  const { identity } = useAuth();
  const [activeTab, setActiveTab] = useState<ActiveTab>('pending');
  const [historyRefreshTrigger, setHistoryRefreshTrigger] = useState(0);

  const isOrgAdmin = !!identity?.is_org;

  // Fetch pending budgets with SWR
  const {
    data: pendingBudgetData,
    loading: pendingBudgetLoading,
    refresh: refreshPendingBudgets,
  } = useSWRApi(`${CacheKeys.approvals}:pending-budgets`, () => approvalsApi.listPending(), {
    refreshInterval: 15000,
  });

  // Fetch pending rate limits with SWR
  const {
    data: pendingRateLimits,
    loading: pendingRateLimitLoading,
    refresh: refreshPendingRateLimits,
  } = useSWRApi(`${CacheKeys.approvals}:pending-ratelimits`, () => rateLimitsApi.listPending(), {
    refreshInterval: 15000,
  });

  const pendingBudgetItems: ApprovalWithBudget[] = pendingBudgetData?.data ?? [];
  const pendingRateLimitItems: RateLimitAllocation[] = pendingRateLimits ?? [];

  const refreshAll = useCallback(() => {
    refreshPendingBudgets();
    refreshPendingRateLimits();
    invalidateKey(CacheKeys.sidebarStats);
    invalidateKey(CacheKeys.approvalCount);
    invalidateKey(CacheKeys.rateLimitApprovalCount);
  }, [refreshPendingBudgets, refreshPendingRateLimits]);

  const pendingLoading = pendingBudgetLoading || pendingRateLimitLoading;

  return (
    <Container>
      <PageHeader
        description={
          isOrgAdmin
            ? 'Review and action pending budget and rate limit approval requests'
            : 'Track the status of your submitted budget and rate limit requests'
        }
      >
        {activeTab === 'pending' && (
          <Button variant="secondary" onClick={refreshAll}>
            Refresh
          </Button>
        )}
        {activeTab === 'history' && (
          <Button variant="secondary" onClick={() => setHistoryRefreshTrigger(p => p + 1)}>
            Refresh
          </Button>
        )}
      </PageHeader>

      <TabBar>
        <Tab active={activeTab === 'pending'} onClick={() => setActiveTab('pending')}>
          Pending
        </Tab>
        <Tab active={activeTab === 'history'} onClick={() => setActiveTab('history')}>
          History
        </Tab>
      </TabBar>

      {activeTab === 'pending' && (
        <>
          {pendingLoading ? (
            <Loading />
          ) : isOrgAdmin ? (
            <OrgPendingTab
              budgetItems={pendingBudgetItems}
              rateLimitItems={pendingRateLimitItems}
              onRefresh={refreshAll}
            />
          ) : (
            <TeamPendingTab
              budgetItems={pendingBudgetItems}
              rateLimitItems={pendingRateLimitItems}
              onRefresh={refreshAll}
            />
          )}
        </>
      )}

      {activeTab === 'history' && (
        <HistoryTab pageSize={HISTORY_PAGE_SIZE} refreshTrigger={historyRefreshTrigger} />
      )}
    </Container>
  );
}
