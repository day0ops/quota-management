import { useState, useEffect } from 'react';
import styled from '@emotion/styled';
import { spacing, colors } from '../../styles';
import { Modal } from '../../components/common/Modal';
import { Button } from '../../components/common/Button';
import { Input, FormField } from '../../components/common/Input';
import { Select } from '../../components/common/Select';
import {
  RateLimitAllocation,
  CreateRateLimitRequest,
  TimeUnit,
  Enforcement,
} from '../../api/types';
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

const InfoText = styled.div`
  color: ${colors.mutedForeground};
  font-size: 12px;
  font-style: italic;
  margin-top: ${spacing[1]};
`;

const FieldGroup = styled.div`
  border: 1px solid ${colors.border};
  border-radius: 8px;
  padding: ${spacing[4]};
  display: flex;
  flex-direction: column;
  gap: ${spacing[3]};
`;

const FieldGroupLabel = styled.div`
  font-size: 12px;
  font-weight: 500;
  color: ${colors.mutedForeground};
  margin-bottom: ${spacing[1]};
`;

const FieldGroupRow = styled.div`
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: ${spacing[3]};
`;

interface RateLimitFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateRateLimitRequest) => Promise<void>;
  editingRateLimit?: RateLimitAllocation | null;
  loading?: boolean;
}

export function RateLimitForm({
  open,
  onClose,
  onSubmit,
  editingRateLimit,
  loading = false,
}: RateLimitFormProps) {
  const { identity, permissions } = useAuth();
  const isTeamMember = permissions.isTeamMember;

  const [formData, setFormData] = useState<CreateRateLimitRequest>({
    org_id: identity?.org_id,
    team_id: identity?.team_id || '',
    model_pattern: '*',
    token_limit: undefined,
    token_unit: 'MINUTE',
    request_limit: undefined,
    request_unit: 'MINUTE',
    burst_percentage: 0,
    enforcement: 'enforced',
    enabled: true,
    description: '',
  });

  const [tokenLimitStr, setTokenLimitStr] = useState('');
  const [requestLimitStr, setRequestLimitStr] = useState('');
  const [burstStr, setBurstStr] = useState('0');

  // Reset form only when modal opens or when switching between create/edit modes
  // Removed identity from dependencies - it's only needed for initial values and doesn't change
  useEffect(() => {
    if (!open) return; // Only reset when modal is opening

    if (editingRateLimit) {
      setFormData({
        org_id: editingRateLimit.org_id,
        team_id: editingRateLimit.team_id,
        model_pattern: editingRateLimit.model_pattern,
        token_limit: editingRateLimit.token_limit,
        token_unit: editingRateLimit.token_unit || 'MINUTE',
        request_limit: editingRateLimit.request_limit,
        request_unit: editingRateLimit.request_unit || 'MINUTE',
        burst_percentage: editingRateLimit.burst_percentage,
        enforcement: editingRateLimit.enforcement,
        enabled: editingRateLimit.enabled,
        description: editingRateLimit.description,
      });
      setTokenLimitStr(editingRateLimit.token_limit?.toString() || '');
      setRequestLimitStr(editingRateLimit.request_limit?.toString() || '');
      setBurstStr(editingRateLimit.burst_percentage.toString());
    } else {
      setFormData({
        org_id: identity?.org_id,
        team_id: identity?.team_id || '',
        model_pattern: '*',
        token_limit: undefined,
        token_unit: 'MINUTE',
        request_limit: undefined,
        request_unit: 'MINUTE',
        burst_percentage: 0,
        enforcement: 'enforced',
        enabled: true,
        description: '',
      });
      setTokenLimitStr('');
      setRequestLimitStr('');
      setBurstStr('0');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editingRateLimit, open]); // identity intentionally omitted - only needed for initial values

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await onSubmit(formData);
  };

  const handleChange = <K extends keyof CreateRateLimitRequest>(
    field: K,
    value: CreateRateLimitRequest[K]
  ) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const isEditing = !!editingRateLimit;
  const title = isEditing ? 'Edit Rate Limit' : 'Create Rate Limit';

  const formatUnitDisplay = (unit: TimeUnit): string => {
    switch (unit) {
      case 'SECOND':
        return 'Second';
      case 'MINUTE':
        return 'Minute';
      case 'HOUR':
        return 'Hour';
      case 'DAY':
        return 'Day';
      default:
        return unit;
    }
  };

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
            label="Organization"
            tooltip={`The organization that owns this rate limit.\nAutomatically set to your organization.`}
            fullWidth
          >
            <Input value={formData.org_id || identity?.org_id || ''} disabled />
          </FormField>
          <FormField
            label="Team ID"
            tooltip={`The team this rate limit applies to.\nAutomatically set to your team.`}
            fullWidth
          >
            <Input
              value={formData.team_id}
              onChange={e => handleChange('team_id', e.target.value)}
              placeholder="e.g., ml-platform, data-science"
              disabled={isEditing || isTeamMember}
              required
            />
          </FormField>
        </Row>

        <FormField
          label="Model Pattern"
          tooltip={`Model pattern to match.\nUse * for all models, or patterns like gpt-4* for specific models.`}
          fullWidth
        >
          <Input
            value={formData.model_pattern}
            onChange={e => handleChange('model_pattern', e.target.value)}
            placeholder="e.g., gpt-4*, claude-*, *"
            required
          />
        </FormField>

        <FieldGroup>
          <FieldGroupLabel>Token Limit</FieldGroupLabel>
          <FieldGroupRow>
            <FormField
              label="Limit"
              tooltip={`Maximum tokens per time unit.\nLeave empty if not limiting by tokens.`}
              fullWidth
            >
              <Input
                type="text"
                inputMode="numeric"
                value={tokenLimitStr}
                onChange={e => {
                  setTokenLimitStr(e.target.value);
                  const num = parseInt(e.target.value);
                  if (!isNaN(num) && num >= 0) {
                    handleChange('token_limit', num);
                  } else if (e.target.value === '') {
                    handleChange('token_limit', undefined);
                  }
                }}
                placeholder="e.g., 100000"
              />
            </FormField>
            <FormField label="Per" fullWidth>
              <Select
                value={formData.token_unit || 'MINUTE'}
                onChange={e => handleChange('token_unit', e.target.value as TimeUnit)}
              >
                <option value="SECOND">{formatUnitDisplay('SECOND')}</option>
                <option value="MINUTE">{formatUnitDisplay('MINUTE')}</option>
                <option value="HOUR">{formatUnitDisplay('HOUR')}</option>
                <option value="DAY">{formatUnitDisplay('DAY')}</option>
              </Select>
            </FormField>
          </FieldGroupRow>
        </FieldGroup>

        <FieldGroup>
          <FieldGroupLabel>Request Limit</FieldGroupLabel>
          <FieldGroupRow>
            <FormField
              label="Limit"
              tooltip={`Maximum requests per time unit.\nLeave empty if not limiting by requests.`}
              fullWidth
            >
              <Input
                type="text"
                inputMode="numeric"
                value={requestLimitStr}
                onChange={e => {
                  setRequestLimitStr(e.target.value);
                  const num = parseInt(e.target.value);
                  if (!isNaN(num) && num >= 0) {
                    handleChange('request_limit', num);
                  } else if (e.target.value === '') {
                    handleChange('request_limit', undefined);
                  }
                }}
                placeholder="e.g., 60"
              />
            </FormField>
            <FormField label="Per" fullWidth>
              <Select
                value={formData.request_unit || 'MINUTE'}
                onChange={e => handleChange('request_unit', e.target.value as TimeUnit)}
              >
                <option value="SECOND">{formatUnitDisplay('SECOND')}</option>
                <option value="MINUTE">{formatUnitDisplay('MINUTE')}</option>
                <option value="HOUR">{formatUnitDisplay('HOUR')}</option>
                <option value="DAY">{formatUnitDisplay('DAY')}</option>
              </Select>
            </FormField>
          </FieldGroupRow>
        </FieldGroup>

        {!tokenLimitStr && !requestLimitStr && (
          <InfoText style={{ color: colors.warning }}>
            At least one of Token Limit or Request Limit is required.
          </InfoText>
        )}

        <Row>
          <FormField
            label="Burst Percentage"
            tooltip={`Burst allowance as a percentage of the limit (0-100).\n0 means no burst.`}
            fullWidth
          >
            <Input
              type="text"
              inputMode="numeric"
              value={burstStr}
              onChange={e => {
                setBurstStr(e.target.value);
                const num = parseInt(e.target.value);
                if (!isNaN(num) && num >= 0 && num <= 100) {
                  handleChange('burst_percentage', num);
                }
              }}
              onBlur={() => {
                const num = parseInt(burstStr);
                if (isNaN(num) || num < 0 || num > 100) {
                  setBurstStr((formData.burst_percentage ?? 0).toString());
                }
              }}
              placeholder="0-100"
            />
          </FormField>
          <FormField
            label="Enforcement"
            tooltip={`Enforced: Requests blocked when over limit.\nMonitoring: Log only, don't block.`}
            fullWidth
          >
            <Select
              value={formData.enforcement || 'enforced'}
              onChange={e => handleChange('enforcement', e.target.value as Enforcement)}
            >
              <option value="enforced">Enforced (block requests)</option>
              <option value="monitoring">Monitoring (log only)</option>
            </Select>
          </FormField>
        </Row>

        <Row>
          <FormField
            label="Enabled"
            tooltip={`When disabled, this rate limit is not active.`}
            fullWidth
          >
            <Select
              value={formData.enabled ? 'true' : 'false'}
              onChange={e => handleChange('enabled', e.target.value === 'true')}
            >
              <option value="true">Yes (active)</option>
              <option value="false">No (inactive)</option>
            </Select>
          </FormField>
          <FormField label="Description" fullWidth>
            <Input
              value={formData.description || ''}
              onChange={e => handleChange('description', e.target.value || undefined)}
              placeholder="Optional description"
            />
          </FormField>
        </Row>
      </Form>
    </Modal>
  );
}
