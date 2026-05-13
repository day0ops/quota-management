import { useState, useCallback, useEffect } from 'react';
import styled from '@emotion/styled';
import toast from 'react-hot-toast';
import { spacing, colors, fontSize, radius } from '../../styles';
import { PageHeader } from '../../components/layout/PageHeader';
import { Button } from '../../components/common/Button';
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
import { ConfirmDialog } from '../../components/common/ConfirmDialog';
import { Loading } from '../../components/common/Spinner';
import { Pagination } from '../../components/common/Pagination';
import { useMutation } from '../../hooks/useApi';
import { useSWRApi, CacheKeys } from '../../hooks/useSWR';
import { modelCostsApi } from '../../api/model-costs';
import { ModelCost, CreateModelCostRequest } from '../../api/types';
import { ModelCostForm } from './ModelCostForm';
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

const PaginationWrapper = styled.div`
  display: flex;
  justify-content: flex-end;
  padding: ${spacing[4]} 0;
`;

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

function formatCost(cost: number): string {
  if (cost === 0) return '$0.00';
  if (cost >= 1) return `$${cost.toFixed(2)}`;
  if (cost >= 0.01) return `$${cost.toFixed(4)}`;
  if (cost >= 0.001) return `$${cost.toFixed(5)}`;
  if (cost >= 0.0001) return `$${cost.toFixed(6)}`;
  return `$${cost.toPrecision(2)}`;
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '—';
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

export function ModelCostsPage() {
  const { permissions } = useAuth();
  const [formOpen, setFormOpen] = useState(false);
  const [editingCost, setEditingCost] = useState<ModelCost | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ModelCost | null>(null);
  const [page, setPage] = useState(1);
  const [provider, setProvider] = useState('');
  const [providers, setProviders] = useState<string[]>([]);
  const [sort, setSort] = useState('');

  // Fetch providers list with SWR
  const { data: providersData } = useSWRApi('model-costs:providers', modelCostsApi.providers, {
    revalidateOnFocus: false,
  });
  useEffect(() => {
    if (providersData) {
      setProviders(providersData);
    }
  }, [providersData]);

  const [sortBy, sortDir] = sort ? (sort.split(':') as [string, string]) : ['', 'asc'];

  // Use SWR for model costs list with background refresh
  const {
    data: pageData,
    loading,
    refresh,
  } = useSWRApi(
    `${CacheKeys.modelCosts}?page=${page}&provider=${provider}&sort=${sort}`,
    () => modelCostsApi.list(page, 30, provider, sortBy, sortDir),
    { refreshInterval: 15000 }
  );

  const costs = pageData?.data ?? null;
  const pagination = pageData?.pagination ?? null;

  const createMutation = useMutation(modelCostsApi.create);
  const updateMutation = useMutation(
    useCallback(
      (modelId: string, data: CreateModelCostRequest) => modelCostsApi.update(modelId, data),
      []
    )
  );
  const deleteMutation = useMutation(modelCostsApi.delete);

  const handleCreate = () => {
    setEditingCost(null);
    setFormOpen(true);
  };

  const handleEdit = (cost: ModelCost) => {
    setEditingCost(cost);
    setFormOpen(true);
  };

  const handleFormSubmit = async (data: CreateModelCostRequest) => {
    try {
      if (editingCost) {
        await updateMutation.execute(editingCost.model_id, data);
        toast.success('Model cost updated');
      } else {
        await createMutation.execute(data);
        toast.success('Model cost created');
      }
      setFormOpen(false);
      refresh();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'An error occurred';
      toast.error(message);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await deleteMutation.execute(deleteTarget.model_id);
      toast.success('Model cost deleted');
      setDeleteTarget(null);
      refresh();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'An error occurred';
      toast.error(message);
    }
  };

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
  };

  const handleSortClick = (col: 'input' | 'output') => {
    if (col === 'input') {
      if (sort === 'input_cost:asc') setSort('both:asc');
      else if (sort === 'both:asc') setSort('both:desc');
      else if (sort === 'both:desc') setSort('');
      else setSort('input_cost:asc');
    } else {
      if (sort === 'output_cost:asc') setSort('output_cost:desc');
      else if (sort === 'output_cost:desc') setSort('');
      else setSort('output_cost:asc');
    }
    setPage(1);
  };

  const getSortIndicator = (col: 'input' | 'output'): string => {
    if (col === 'input') {
      if (sort === 'input_cost:asc' || sort === 'both:asc') return ' ↑';
      if (sort === 'both:desc') return ' ↓';
      return '';
    } else {
      if (sort === 'output_cost:asc' || sort === 'both:asc') return ' ↑';
      if (sort === 'output_cost:desc' || sort === 'both:desc') return ' ↓';
      return '';
    }
  };

  return (
    <Container>
      <PageHeader description="Configure pricing for LLM models. All costs are in USD per million tokens.">
        <Button variant="secondary" onClick={refresh}>
          Refresh
        </Button>
        {permissions.canManageModelCosts && <Button onClick={handleCreate}>Add Model Cost</Button>}
      </PageHeader>

      <FiltersRow>
        <FilterGroup>
          <FilterLabel htmlFor="provider-filter">Provider</FilterLabel>
          <FilterSelect
            id="provider-filter"
            value={provider}
            onChange={e => {
              setProvider(e.target.value);
              setPage(1);
            }}
          >
            <option value="">All Providers</option>
            {providers.map(p => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </FilterSelect>
        </FilterGroup>
      </FiltersRow>

      {loading ? (
        <Loading />
      ) : costs && costs.length > 0 ? (
        <>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Model ID</TableHeader>
                  <TableHeader>Provider</TableHeader>
                  <TableHeader
                    align="right"
                    onClick={() => handleSortClick('input')}
                    style={{ cursor: 'pointer', userSelect: 'none' }}
                  >
                    Input Cost{getSortIndicator('input')}
                  </TableHeader>
                  <TableHeader
                    align="right"
                    onClick={() => handleSortClick('output')}
                    style={{ cursor: 'pointer', userSelect: 'none' }}
                  >
                    Output Cost{getSortIndicator('output')}
                  </TableHeader>
                  <TableHeader>Pattern</TableHeader>
                  <TableHeader>Created At</TableHeader>
                  <TableHeader>Created By</TableHeader>
                  {permissions.canManageModelCosts && (
                    <TableHeader align="right">Actions</TableHeader>
                  )}
                </TableRow>
              </TableHead>
              <TableBody>
                {costs.map(cost => (
                  <TableRow key={cost.model_id}>
                    <TableCell>{cost.model_id}</TableCell>
                    <TableCell>{cost.provider}</TableCell>
                    <TableCell align="right">{formatCost(cost.input_cost_per_million)}</TableCell>
                    <TableCell align="right">{formatCost(cost.output_cost_per_million)}</TableCell>
                    <TableCell>{cost.model_pattern || '—'}</TableCell>
                    <TableCell>
                      <span style={{ fontSize: fontSize.xs, color: colors.mutedForeground }}>
                        {formatDate(cost.created_at)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span style={{ fontSize: fontSize.xs, color: colors.mutedForeground }}>
                        {cost.created_by_email || cost.created_by_user_id || 'System'}
                      </span>
                    </TableCell>
                    {permissions.canManageModelCosts && (
                      <TableCell align="right">
                        <ActionButtons>
                          <ActionButton onClick={() => handleEdit(cost)}>Edit</ActionButton>
                          <ActionButton onClick={() => setDeleteTarget(cost)}>Delete</ActionButton>
                        </ActionButtons>
                      </TableCell>
                    )}
                  </TableRow>
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
          <EmptyStateText>
            {provider
              ? 'No model costs found for this provider.'
              : 'No model costs configured yet.'}
          </EmptyStateText>
          {!provider && permissions.canManageModelCosts && (
            <Button onClick={handleCreate}>Add Model Cost</Button>
          )}
        </EmptyState>
      )}

      <ModelCostForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onSubmit={handleFormSubmit}
        editingCost={editingCost}
        loading={createMutation.loading || updateMutation.loading}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete Model Cost"
        message={`Are you sure you want to delete the cost configuration for "${deleteTarget?.model_id}"? This action cannot be undone.`}
        confirmLabel="Delete"
        loading={deleteMutation.loading}
      />
    </Container>
  );
}
