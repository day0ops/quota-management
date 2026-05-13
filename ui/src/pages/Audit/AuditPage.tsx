import { useState, useCallback, useEffect } from 'react';
import styled from '@emotion/styled';
import { colors, spacing, fontSize, radius } from '../../styles';
import { PageHeader } from '../../components/layout/PageHeader';
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
import { Pagination } from '../../components/common/Pagination';
import { Loading } from '../../components/common/Spinner';
import { Button } from '../../components/common/Button';
import { auditApi, AuditFilters } from '../../api/audit';
import { AuditLogEntry, PaginatedResponse } from '../../api/types';

const Container = styled.div``;

const FiltersRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${spacing[3]};
  margin-bottom: ${spacing[6]};
  align-items: flex-end;
`;

const FilterGroup = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${spacing[1]};
`;

const FilterLabel = styled.label`
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
`;

const FilterSelect = styled.select`
  background: ${colors.cardBg};
  border: 1px solid ${colors.border};
  color: ${colors.foreground};
  font-size: ${fontSize.sm};
  padding: ${spacing[2]} ${spacing[3]};
  border-radius: ${radius.md};
  min-width: 140px;
  outline: none;
  cursor: pointer;

  &:focus {
    border-color: ${colors.primary};
  }

  option {
    background: ${colors.cardBg};
    color: ${colors.foreground};
  }
`;

const FilterInput = styled.input`
  background: ${colors.cardBg};
  border: 1px solid ${colors.border};
  color: ${colors.foreground};
  font-size: ${fontSize.sm};
  padding: ${spacing[2]} ${spacing[3]};
  border-radius: ${radius.md};
  min-width: 140px;
  outline: none;
  color-scheme: var(--color-scheme);

  &:focus {
    border-color: ${colors.primary};
  }

  &::placeholder {
    color: ${colors.mutedForeground};
  }
`;

const PaginationRow = styled.div`
  display: flex;
  justify-content: flex-end;
  margin-top: ${spacing[4]};
`;

const ActionBadge = styled.span<{ action: string }>`
  display: inline-block;
  padding: ${spacing[1]} ${spacing[2]};
  border-radius: ${radius.sm};
  font-size: ${fontSize.xs};
  font-weight: 500;
  background: ${({ action }) => {
    switch (action) {
      case 'created':
        return 'rgba(34, 197, 94, 0.15)';
      case 'updated':
        return 'rgba(59, 130, 246, 0.15)';
      case 'deleted':
        return 'rgba(239, 68, 68, 0.15)';
      case 'approved':
        return 'rgba(34, 197, 94, 0.15)';
      case 'rejected':
        return 'rgba(239, 68, 68, 0.15)';
      case 'budget_reset':
        return 'rgba(251, 191, 36, 0.15)';
      default:
        return 'rgba(148, 163, 184, 0.15)';
    }
  }};
  color: ${({ action }) => {
    switch (action) {
      case 'created':
        return '#4ade80';
      case 'updated':
        return '#60a5fa';
      case 'deleted':
        return '#f87171';
      case 'approved':
        return '#4ade80';
      case 'rejected':
        return '#f87171';
      case 'budget_reset':
        return '#fbbf24';
      default:
        return colors.mutedForeground;
    }
  }};
`;

const ExpandButton = styled.button`
  padding: ${spacing[1]} ${spacing[2]};
  border-radius: ${radius.sm};
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
  border: 1px solid ${colors.border};
  background: transparent;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;

  &:hover {
    background: ${colors.hoverBg};
    color: ${colors.foreground};
  }
`;

const MetadataRow = styled.tr`
  background: ${colors.surfaceBg};
  border-bottom: 1px solid ${colors.borderDark};
`;

const MetadataCell = styled.td`
  padding: ${spacing[3]} ${spacing[4]};
`;

const MetadataPre = styled.pre`
  margin: 0;
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: ${fontSize.xs};
  color: ${colors.foreground};
  background: ${colors.cardBg};
  border: 1px solid ${colors.border};
  border-radius: ${radius.md};
  padding: ${spacing[3]};
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
`;

const EntityTypeBadge = styled.span`
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
`;

const ActorCell = styled.div`
  font-size: ${fontSize.sm};
`;

const ActorEmail = styled.div`
  font-size: ${fontSize.xs};
  color: ${colors.mutedForeground};
`;

function formatTimestamp(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleString(undefined, {
      year: 'numeric',
      month: 'short',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  } catch {
    return iso;
  }
}

function formatEntityType(type: string): string {
  return type
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

function formatAction(action: string): string {
  return action
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

const PAGE_SIZE = 30;

interface FiltersState {
  entity_type: string;
  action: string;
  actor: string;
  from: string;
  to: string;
}

function buildApiFilters(filters: FiltersState): AuditFilters {
  const apiFilters: AuditFilters = {};
  if (filters.entity_type && filters.entity_type !== 'all') {
    apiFilters.entity_type = filters.entity_type;
  }
  if (filters.action && filters.action !== 'all') {
    apiFilters.action = filters.action;
  }
  if (filters.actor.trim()) {
    apiFilters.actor = filters.actor.trim();
  }
  if (filters.from) {
    apiFilters.from = `${filters.from}T00:00:00Z`;
  }
  if (filters.to) {
    apiFilters.to = `${filters.to}T23:59:59Z`;
  }
  return apiFilters;
}

export function AuditPage() {
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<FiltersState>({
    entity_type: 'all',
    action: 'all',
    actor: '',
    from: '',
    to: '',
  });
  const [data, setData] = useState<PaginatedResponse<AuditLogEntry> | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await auditApi.list(page, PAGE_SIZE, buildApiFilters(filters));
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, [page, filters]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleFilterChange = useCallback(
    <K extends keyof FiltersState>(key: K, value: FiltersState[K]) => {
      setFilters(prev => ({ ...prev, [key]: value }));
      setPage(1);
    },
    []
  );

  const toggleExpand = useCallback((id: string) => {
    setExpandedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const isFiltered =
    filters.entity_type !== 'all' ||
    filters.action !== 'all' ||
    filters.actor.trim() !== '' ||
    filters.from !== '' ||
    filters.to !== '';

  const handleResetFilters = useCallback(() => {
    setFilters({ entity_type: 'all', action: 'all', actor: '', from: '', to: '' });
    setPage(1);
    setExpandedIds(new Set());
  }, []);

  const entries = data?.data ?? [];
  const totalPages = data?.pagination.total_pages ?? 1;

  return (
    <Container>
      <PageHeader description="Track all changes and actions across budgets, rate limits, model costs, and approvals." />

      <FiltersRow>
        <FilterGroup>
          <FilterLabel>Entity Type</FilterLabel>
          <FilterSelect
            value={filters.entity_type}
            onChange={e => handleFilterChange('entity_type', e.target.value)}
          >
            <option value="all">All</option>
            <option value="budget">Budget</option>
            <option value="rate_limit">Rate Limit</option>
            <option value="model_cost">Model Cost</option>
          </FilterSelect>
        </FilterGroup>

        <FilterGroup>
          <FilterLabel>Action</FilterLabel>
          <FilterSelect
            value={filters.action}
            onChange={e => handleFilterChange('action', e.target.value)}
          >
            <option value="all">All</option>
            <option value="created">Created</option>
            <option value="updated">Updated</option>
            <option value="deleted">Deleted</option>
            <option value="approved">Approved</option>
            <option value="rejected">Rejected</option>
            <option value="budget_reset">Budget Reset</option>
          </FilterSelect>
        </FilterGroup>

        <FilterGroup>
          <FilterLabel>Actor</FilterLabel>
          <FilterInput
            type="text"
            placeholder="Filter by actor..."
            value={filters.actor}
            onChange={e => handleFilterChange('actor', e.target.value)}
          />
        </FilterGroup>

        <FilterGroup>
          <FilterLabel>From</FilterLabel>
          <FilterInput
            type="date"
            value={filters.from}
            onChange={e => handleFilterChange('from', e.target.value)}
          />
        </FilterGroup>

        <FilterGroup>
          <FilterLabel>To</FilterLabel>
          <FilterInput
            type="date"
            value={filters.to}
            onChange={e => handleFilterChange('to', e.target.value)}
          />
        </FilterGroup>

        {isFiltered && (
          <Button variant="ghost" size="sm" onClick={handleResetFilters}>
            Reset filters
          </Button>
        )}
      </FiltersRow>

      {loading ? (
        <Loading />
      ) : error ? (
        <EmptyState>
          <EmptyStateText>Failed to load audit logs: {error.message}</EmptyStateText>
        </EmptyState>
      ) : (
        <>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Timestamp</TableHeader>
                  <TableHeader>Action</TableHeader>
                  <TableHeader>Entity</TableHeader>
                  <TableHeader>Name</TableHeader>
                  <TableHeader>Actor</TableHeader>
                  <TableHeader>Team</TableHeader>
                  <TableHeader>Details</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {entries.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7}>
                      <EmptyState>
                        <EmptyStateText>No audit log entries found.</EmptyStateText>
                      </EmptyState>
                    </TableCell>
                  </TableRow>
                ) : (
                  entries.map(entry => (
                    <>
                      <TableRow key={entry.id}>
                        <TableCell>{formatTimestamp(entry.created_at)}</TableCell>
                        <TableCell>
                          <ActionBadge action={entry.action}>
                            {formatAction(entry.action)}
                          </ActionBadge>
                        </TableCell>
                        <TableCell>
                          <EntityTypeBadge>{formatEntityType(entry.entity_type)}</EntityTypeBadge>
                        </TableCell>
                        <TableCell>{entry.entity_id}</TableCell>
                        <TableCell>
                          <ActorCell>
                            {entry.actor_user_id && !entry.actor_email ? (
                              entry.actor_user_id
                            ) : entry.actor_email ? (
                              <>
                                <div>{entry.actor_email}</div>
                                {entry.actor_user_id && (
                                  <ActorEmail>{entry.actor_user_id}</ActorEmail>
                                )}
                              </>
                            ) : (
                              <span style={{ color: colors.mutedForeground }}>—</span>
                            )}
                          </ActorCell>
                        </TableCell>
                        <TableCell>
                          {entry.team_id ? (
                            entry.team_id
                          ) : (
                            <span style={{ color: colors.mutedForeground }}>—</span>
                          )}
                        </TableCell>
                        <TableCell>
                          {entry.metadata && Object.keys(entry.metadata).length > 0 ? (
                            <ExpandButton onClick={() => toggleExpand(entry.id)}>
                              {expandedIds.has(entry.id) ? 'collapse' : 'expand'}
                            </ExpandButton>
                          ) : (
                            <span style={{ color: colors.mutedForeground, fontSize: fontSize.xs }}>
                              —
                            </span>
                          )}
                        </TableCell>
                      </TableRow>
                      {expandedIds.has(entry.id) && entry.metadata && (
                        <MetadataRow key={`${entry.id}-meta`}>
                          <MetadataCell colSpan={7}>
                            <MetadataPre>{JSON.stringify(entry.metadata, null, 2)}</MetadataPre>
                          </MetadataCell>
                        </MetadataRow>
                      )}
                    </>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>

          <PaginationRow>
            <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
          </PaginationRow>
        </>
      )}
    </Container>
  );
}
