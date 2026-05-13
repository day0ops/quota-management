import { useState, useEffect, useCallback } from 'react';
import styled from '@emotion/styled';
import { spacing, colors } from '../../styles';
import { Modal } from '../../components/common/Modal';
import { Button } from '../../components/common/Button';
import { Input, Textarea, FormField } from '../../components/common/Input';
import { Select } from '../../components/common/Select';
import { BudgetDefinition, CreateBudgetRequest, EntityType, BudgetPeriod } from '../../api/types';
import { budgetsApi } from '../../api/budgets';
import { useAuth } from '../../contexts/AuthContext';

const Form = styled.form`
  display: flex;
  flex-direction: column;
  gap: ${spacing[4]};
`;

const Row = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: ${spacing[4]};
`;

const Footer = styled.div`
  display: flex;
  gap: ${spacing[3]};
  justify-content: flex-end;
`;

const ErrorText = styled.div`
  color: ${colors.error};
  font-size: 12px;
  margin-top: ${spacing[1]};
`;

const InfoText = styled.div`
  color: ${colors.mutedForeground};
  font-size: 12px;
  font-style: italic;
  margin-top: ${spacing[1]};
`;

interface ParentCandidate {
  id: string;
  name: string;
}

interface BudgetFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateBudgetRequest) => Promise<void>;
  editingBudget?: BudgetDefinition | null;
  parentCandidates?: ParentCandidate[];
  loading?: boolean;
  submitError?: string | null;
}

export function BudgetForm({
  open,
  onClose,
  onSubmit,
  editingBudget,
  parentCandidates = [],
  loading = false,
  submitError,
}: BudgetFormProps) {
  const { identity, permissions } = useAuth();
  const isOrgAdmin = permissions.isOrgAdmin;
  const isTeamMember = permissions.isTeamMember;

  const [formData, setFormData] = useState<CreateBudgetRequest>({
    entity_type: 'org',
    name: '',
    match_expression: 'true',
    budget_amount_usd: 100,
    period: 'monthly',
    warning_threshold_pct: 80,
    isolated: false, // Default to non-isolated
    allow_fallback: false,
    enabled: true,
    owner_org_id: identity?.org_id,
    owner_team_id: identity?.team_id,
  });

  const [celError, setCelError] = useState<string | null>(null);
  const [celValidating, setCelValidating] = useState(false);
  const [budgetAmountStr, setBudgetAmountStr] = useState('100');
  const [warningThresholdStr, setWarningThresholdStr] = useState('80');
  const [customPeriodStr, setCustomPeriodStr] = useState('');
  const [customPeriodError, setCustomPeriodError] = useState<string | null>(null);

  const validateCEL = useCallback(async (expression: string) => {
    if (!expression.trim()) {
      setCelError('Match expression is required');
      return;
    }

    setCelValidating(true);
    try {
      const result = await budgetsApi.validateCEL(expression);
      if (result.valid) {
        setCelError(null);
      } else {
        setCelError(result.error || 'Invalid CEL expression');
      }
    } catch {
      setCelError('Failed to validate expression');
    } finally {
      setCelValidating(false);
    }
  }, []);

  // Reset form only when modal opens or when switching between create/edit modes
  // Removed identity from dependencies - it's only needed for initial values and doesn't change
  useEffect(() => {
    if (!open) return; // Only reset when modal is opening

    if (editingBudget) {
      setFormData({
        entity_type: editingBudget.entity_type,
        name: editingBudget.name,
        match_expression: editingBudget.match_expression,
        budget_amount_usd: editingBudget.budget_amount_usd,
        period: editingBudget.period,
        custom_period_seconds: editingBudget.custom_period_seconds,
        warning_threshold_pct: editingBudget.warning_threshold_pct,
        parent_id: editingBudget.parent_id,
        isolated: editingBudget.isolated,
        allow_fallback: editingBudget.allow_fallback,
        enabled: editingBudget.enabled,
        description: editingBudget.description,
        owner_org_id: editingBudget.owner_org_id,
        owner_team_id: editingBudget.owner_team_id,
      });
      setBudgetAmountStr(editingBudget.budget_amount_usd.toString());
      setWarningThresholdStr(editingBudget.warning_threshold_pct.toString());
      setCustomPeriodStr(editingBudget.custom_period_seconds?.toString() || '');
      setCelError(null);
      setCustomPeriodError(null);
    } else {
      setFormData({
        entity_type: 'org',
        name: '',
        match_expression: 'true',
        budget_amount_usd: 100,
        period: 'monthly',
        warning_threshold_pct: 80,
        isolated: false, // Default to non-isolated
        allow_fallback: false,
        enabled: true,
        owner_org_id: identity?.org_id,
        owner_team_id: identity?.team_id,
      });
      setBudgetAmountStr('100');
      setWarningThresholdStr('80');
      setCustomPeriodStr('');
      setCelError(null);
      setCustomPeriodError(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editingBudget, open]); // identity intentionally omitted - only needed for initial values

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (formData.period === 'custom') {
      const secs = formData.custom_period_seconds;
      if (!secs || secs < 3600) {
        setCustomPeriodError('Custom period must be at least 3600 seconds (1 hour)');
        return;
      }
    }
    await onSubmit(formData);
  };

  const handleChange = <K extends keyof CreateBudgetRequest>(
    field: K,
    value: CreateBudgetRequest[K]
  ) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (field === 'match_expression') {
      setCelError(null);
    }
    if (field === 'parent_id' && !value) {
      setFormData(prev => ({ ...prev, allow_fallback: false }));
    }
    if (field === 'entity_type' && (value === 'org' || value === 'user')) {
      setFormData(prev => ({ ...prev, parent_id: undefined, allow_fallback: false }));
    }
  };

  const handleCELBlur = () => {
    if (formData.match_expression.trim()) {
      validateCEL(formData.match_expression);
    }
  };

  const isEditing = !!editingBudget;
  const title = isEditing ? 'Edit Budget' : 'Create Budget';

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      width="650px"
      footer={
        <Footer>
          <Button variant="secondary" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? 'Saving...' : isEditing ? 'Save Changes' : 'Create'}
          </Button>
        </Footer>
      }
    >
      <Form onSubmit={handleSubmit}>
        <Row>
          <FormField
            label="Entity Type"
            tooltip={`The type of entity this budget applies to.\nOrganization: Limits spend for the entire org.\nTeam: Limits spend for a specific team.\nUser: Limits spend for an individual user.`}
            fullWidth
          >
            <Select
              value={formData.entity_type}
              onChange={e => handleChange('entity_type', e.target.value as EntityType)}
              disabled={isEditing}
            >
              {isOrgAdmin && <option value="org">Organization</option>}
              <option value="team">Team</option>
              <option value="user">User</option>
            </Select>
          </FormField>
          <FormField
            label="Name"
            tooltip={`A unique identifier for this budget.\nUsed for display and reference purposes.`}
            fullWidth
          >
            <Input
              value={formData.name}
              onChange={e => handleChange('name', e.target.value)}
              placeholder="e.g., openai, acme-corp, ml-platform"
              disabled={isEditing}
              required
            />
          </FormField>
        </Row>
        {(isOrgAdmin || isTeamMember) && (
          <Row>
            <FormField
              label="Organization"
              tooltip={`The organization that owns this budget.\nAutomatically set to your organization.`}
              fullWidth
            >
              <Input value={formData.owner_org_id || ''} disabled placeholder="Your organization" />
            </FormField>
            {formData.entity_type !== 'org' &&
              (isOrgAdmin ? (
                <FormField
                  label="Team (Optional)"
                  tooltip={`Optionally assign this budget to a specific team within your organization.\nLeave empty for an org-level budget.`}
                  fullWidth
                >
                  <Input
                    value={formData.owner_team_id || ''}
                    onChange={e => handleChange('owner_team_id', e.target.value || undefined)}
                    placeholder="e.g., ml-platform, data-science"
                  />
                </FormField>
              ) : (
                <FormField
                  label="Team"
                  tooltip={`The team this budget belongs to.\nAutomatically set to your team.`}
                  fullWidth
                >
                  <Input value={formData.owner_team_id || ''} disabled />
                </FormField>
              ))}
          </Row>
        )}
        <Row>
          <FormField label="Budget Amount (USD)" fullWidth>
            <Input
              type="text"
              inputMode="decimal"
              value={budgetAmountStr}
              onChange={e => {
                setBudgetAmountStr(e.target.value);
                const num = parseFloat(e.target.value);
                if (!isNaN(num) && num >= 0) {
                  handleChange('budget_amount_usd', num);
                }
              }}
              onBlur={() => {
                const num = parseFloat(budgetAmountStr);
                if (isNaN(num) || num < 0) {
                  setBudgetAmountStr(formData.budget_amount_usd.toString());
                } else {
                  setBudgetAmountStr(num.toString());
                }
              }}
              placeholder="0.00"
              required
            />
          </FormField>
          <FormField
            label="Period"
            tooltip={`How often the budget resets.\nHourly/Daily/Weekly/Monthly: Align to calendar boundaries (start of hour, midnight UTC, Monday, 1st of month).\nCustom: Specify an exact duration in seconds.`}
            fullWidth
          >
            <Select
              value={formData.period}
              onChange={e => handleChange('period', e.target.value as BudgetPeriod)}
            >
              <option value="hourly">Hourly</option>
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
              <option value="custom">Custom</option>
            </Select>
          </FormField>
        </Row>
        {formData.period === 'custom' && (
          <FormField label="Custom Period (seconds)" fullWidth>
            <Input
              type="text"
              inputMode="numeric"
              value={customPeriodStr}
              onChange={e => {
                setCustomPeriodStr(e.target.value);
                const num = parseInt(e.target.value);
                if (!isNaN(num) && num > 0) {
                  handleChange('custom_period_seconds', num);
                  if (num < 3600) {
                    setCustomPeriodError('Minimum 3600 seconds (1 hour)');
                  } else {
                    setCustomPeriodError(null);
                  }
                } else if (e.target.value === '') {
                  handleChange('custom_period_seconds', undefined);
                  setCustomPeriodError(null);
                }
              }}
              onBlur={() => {
                const num = parseInt(customPeriodStr);
                if (customPeriodStr !== '' && (isNaN(num) || num <= 0)) {
                  setCustomPeriodStr(formData.custom_period_seconds?.toString() || '');
                  setCustomPeriodError(null);
                }
              }}
              placeholder="e.g., 3600 for 1 hour (minimum)"
            />
            {customPeriodError && <ErrorText>{customPeriodError}</ErrorText>}
          </FormField>
        )}
        <FormField
          label="Match Expression (CEL)"
          tooltip={`CEL expression that determines which requests this budget applies to.\nUse "true" to match all requests.\nExamples:\n- request.headers["x-team"] == "ml-platform"\n- request.path.startsWith("/openai")`}
          fullWidth
        >
          <Textarea
            value={formData.match_expression}
            onChange={e => handleChange('match_expression', e.target.value)}
            onBlur={handleCELBlur}
            placeholder='e.g., "x-team" in request.headers && request.headers["x-team"] == "ml-platform"'
            required
          />
          {celValidating && (
            <ErrorText style={{ color: colors.mutedForeground }}>Validating...</ErrorText>
          )}
          {celError && !celValidating && <ErrorText>{celError}</ErrorText>}
          {submitError && !celValidating && <ErrorText>{submitError}</ErrorText>}
          <InfoText>
            Test expressions in the{' '}
            <a href="https://playcel.undistro.io/" target="_blank" rel="noopener noreferrer">
              CEL Playground
            </a>
          </InfoText>
        </FormField>
        <Row>
          <FormField
            label="Warning Threshold (%)"
            tooltip={`Percentage of budget usage that triggers a warning.\nAlerts are generated when usage exceeds this threshold.\nRequests are still allowed until 100% is reached.`}
            fullWidth
          >
            <Input
              type="text"
              inputMode="numeric"
              value={warningThresholdStr}
              onChange={e => {
                setWarningThresholdStr(e.target.value);
                const num = parseInt(e.target.value);
                if (!isNaN(num) && num >= 0 && num <= 100) {
                  handleChange('warning_threshold_pct', num);
                }
              }}
              onBlur={() => {
                const num = parseInt(warningThresholdStr);
                if (isNaN(num) || num < 0 || num > 100) {
                  setWarningThresholdStr((formData.warning_threshold_pct ?? 80).toString());
                } else {
                  setWarningThresholdStr(num.toString());
                }
              }}
              placeholder="0-100"
            />
          </FormField>
          <FormField
            label="Enabled"
            tooltip={`Controls whether this budget is enforced.\nYes (default): Budget limits are enforced.\nNo: Requests bypass this budget's limits.`}
            fullWidth
          >
            <Select
              value={formData.enabled ? 'true' : 'false'}
              onChange={e => handleChange('enabled', e.target.value === 'true')}
            >
              <option value="true">Yes (enforce budget)</option>
              <option value="false">No (bypass limits)</option>
            </Select>
          </FormField>
        </Row>
        {/* Team-specific: Parent Budget selection */}
        {formData.entity_type === 'team' && (
          <FormField
            label="Parent Budget (Organization)"
            tooltip={`Link this team budget to a parent org budget.\nTeam budgets inherit Isolated and Allow Fallback settings from their parent.`}
            fullWidth
          >
            <Select
              value={formData.parent_id || ''}
              onChange={e => handleChange('parent_id', e.target.value || undefined)}
            >
              <option value="">None (standalone team budget)</option>
              {parentCandidates
                .filter(c => c.id !== editingBudget?.id)
                .map(candidate => (
                  <option key={candidate.id} value={candidate.id}>
                    {candidate.name}
                  </option>
                ))}
            </Select>
          </FormField>
        )}
        {/* Org: Isolated + Allow Fallback (Allow Fallback disabled when Isolated) */}
        {formData.entity_type === 'org' && (
          <Row>
            <FormField
              label="Isolated"
              tooltip={`Controls how team budget usage is tracked.\nNo (default): Team usage counts against both the team budget and this org budget.\nYes: Each team tracks usage independently without affecting org totals.\nNote: When Isolated is Yes, Allow Fallback is not available.`}
              fullWidth
            >
              <Select
                value={formData.isolated ? 'true' : 'false'}
                onChange={e => {
                  const isIsolated = e.target.value === 'true';
                  handleChange('isolated', isIsolated);
                  // Clear allow_fallback when isolated (they're mutually exclusive)
                  if (isIsolated) {
                    handleChange('allow_fallback', false);
                  }
                }}
              >
                <option value="false">No (default)</option>
                <option value="true">Yes</option>
              </Select>
            </FormField>
            {!formData.isolated && (
              <FormField
                label="Allow Fallback"
                tooltip={`Controls what happens when a team budget is exhausted.\nNo (default): Requests are blocked.\nYes: Requests fall back to this org budget if it has remaining balance.\nNote: Only available when Isolated is No.`}
                fullWidth
              >
                <Select
                  value={formData.allow_fallback ? 'true' : 'false'}
                  onChange={e => handleChange('allow_fallback', e.target.value === 'true')}
                >
                  <option value="false">No (default)</option>
                  <option value="true">Yes</option>
                </Select>
              </FormField>
            )}
          </Row>
        )}
        {/* Team with parent - only org-admins can override inherited settings */}
        {formData.entity_type === 'team' && formData.parent_id && isOrgAdmin && (
          <Row>
            <FormField
              label="Isolated"
              tooltip={`Inherited from parent org.\nNo: Usage counts against both team and org budgets.\nYes: Track usage independently without affecting org totals.\nNote: When Isolated is Yes, Allow Fallback is not available.`}
              fullWidth
            >
              <Select
                value={formData.isolated ? 'true' : 'false'}
                onChange={e => {
                  const isIsolated = e.target.value === 'true';
                  handleChange('isolated', isIsolated);
                  // Clear allow_fallback when isolated (they're mutually exclusive)
                  if (isIsolated) {
                    handleChange('allow_fallback', false);
                  }
                }}
              >
                <option value="false">No (default)</option>
                <option value="true">Yes</option>
              </Select>
            </FormField>
            {!formData.isolated && (
              <FormField
                label="Allow Fallback"
                tooltip={`Inherited from parent org.\nNo: Block requests when this team budget is exhausted.\nYes: Fall back to parent org budget if it has remaining balance.\nNote: Only available when Isolated is No.`}
                fullWidth
              >
                <Select
                  value={formData.allow_fallback ? 'true' : 'false'}
                  onChange={e => handleChange('allow_fallback', e.target.value === 'true')}
                >
                  <option value="false">No (block when exhausted)</option>
                  <option value="true">Yes (use org budget)</option>
                </Select>
              </FormField>
            )}
          </Row>
        )}
        <FormField label="Description" fullWidth>
          <Input
            value={formData.description || ''}
            onChange={e => handleChange('description', e.target.value || undefined)}
            placeholder="Optional description"
          />
        </FormField>
      </Form>
    </Modal>
  );
}
