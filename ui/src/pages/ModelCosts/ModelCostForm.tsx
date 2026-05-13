import { useState, useEffect, useMemo } from 'react';
import styled from '@emotion/styled';
import { spacing, colors } from '../../styles';
import { Modal } from '../../components/common/Modal';
import { Button } from '../../components/common/Button';
import { Input, FormField } from '../../components/common/Input';
import { Select } from '../../components/common/Select';
import { ModelCost, CreateModelCostRequest } from '../../api/types';
import { fetchOpenRouterModels, ModelPreset } from '../../api/openrouter';

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

const PresetSection = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${spacing[2]};
  padding-bottom: ${spacing[4]};
  border-bottom: 1px solid ${colors.border};
  margin-bottom: ${spacing[2]};
`;

const PresetHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
`;

const PresetLabel = styled.span`
  font-size: 14px;
  font-weight: 500;
  color: ${colors.foreground};
`;

const RefreshButton = styled.button`
  background: none;
  border: none;
  color: ${colors.primary};
  font-size: 13px;
  cursor: pointer;
  padding: ${spacing[1]} ${spacing[2]};
  border-radius: 4px;

  &:hover {
    background: ${colors.hoverBg};
  }

  &:disabled {
    color: ${colors.mutedForeground};
    cursor: not-allowed;
  }
`;

const SearchInput = styled(Input)`
  margin-bottom: ${spacing[2]};
`;

const ModelSelectWrapper = styled.div`
  position: relative;
`;

const StatusText = styled.span<{ variant?: 'error' | 'info' }>`
  font-size: 12px;
  color: ${props => (props.variant === 'error' ? colors.error : colors.mutedForeground)};
`;

const OrDivider = styled.div`
  display: flex;
  align-items: center;
  gap: ${spacing[3]};
  margin: ${spacing[2]} 0;
  color: ${colors.mutedForeground};
  font-size: 13px;

  &::before,
  &::after {
    content: '';
    flex: 1;
    height: 1px;
    background: ${colors.border};
  }
`;

interface ModelCostFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateModelCostRequest) => Promise<void>;
  editingCost?: ModelCost | null;
  loading?: boolean;
}

export function ModelCostForm({
  open,
  onClose,
  onSubmit,
  editingCost,
  loading = false,
}: ModelCostFormProps) {
  const [formData, setFormData] = useState<CreateModelCostRequest>({
    model_id: '',
    provider: '',
    input_cost_per_million: 0,
    output_cost_per_million: 0,
  });

  // OpenRouter presets state
  const [presets, setPresets] = useState<ModelPreset[]>([]);
  const [presetsLoading, setPresetsLoading] = useState(false);
  const [presetsError, setPresetsError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedPresetId, setSelectedPresetId] = useState<string>('');

  // Fetch presets when modal opens
  useEffect(() => {
    if (open && presets.length === 0 && !presetsLoading) {
      loadPresets();
    }
  }, [open]);

  const loadPresets = async () => {
    setPresetsLoading(true);
    setPresetsError(null);
    try {
      const models = await fetchOpenRouterModels();
      setPresets(models);
    } catch (err) {
      setPresetsError(err instanceof Error ? err.message : 'Failed to load model presets');
    } finally {
      setPresetsLoading(false);
    }
  };

  // Filter presets based on search
  const filteredPresets = useMemo(() => {
    if (!searchQuery.trim()) return presets;
    const query = searchQuery.toLowerCase();
    return presets.filter(
      p =>
        p.name.toLowerCase().includes(query) ||
        p.model_id.toLowerCase().includes(query) ||
        p.provider.toLowerCase().includes(query)
    );
  }, [presets, searchQuery]);

  useEffect(() => {
    if (editingCost) {
      setFormData({
        model_id: editingCost.model_id,
        provider: editingCost.provider,
        input_cost_per_million: editingCost.input_cost_per_million,
        output_cost_per_million: editingCost.output_cost_per_million,
        cache_read_cost_million: editingCost.cache_read_cost_million,
        cache_write_cost_million: editingCost.cache_write_cost_million,
        model_pattern: editingCost.model_pattern,
      });
      setSelectedPresetId('');
      setSearchQuery('');
    } else {
      setFormData({
        model_id: '',
        provider: '',
        input_cost_per_million: 0,
        output_cost_per_million: 0,
      });
      setSelectedPresetId('');
      setSearchQuery('');
    }
  }, [editingCost, open]);

  const handlePresetSelect = (presetId: string) => {
    setSelectedPresetId(presetId);
    if (presetId === '') return;

    const preset = presets.find(p => p.model_id === presetId);
    if (preset) {
      setFormData({
        model_id: preset.model_id,
        provider: preset.provider,
        input_cost_per_million: preset.input_cost_per_million,
        output_cost_per_million: preset.output_cost_per_million,
        cache_read_cost_million: preset.cache_read_cost_million,
        cache_write_cost_million: preset.cache_write_cost_million,
      });
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await onSubmit(formData);
  };

  const handleChange = (
    field: keyof CreateModelCostRequest,
    value: string | number | undefined
  ) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    // Clear preset selection when manually editing
    if (selectedPresetId) {
      setSelectedPresetId('');
    }
  };

  const isEditing = !!editingCost;
  const title = isEditing ? 'Edit Model Cost' : 'Add Model Cost';

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      width="600px"
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
        {!isEditing && (
          <PresetSection>
            <PresetHeader>
              <PresetLabel>Select from OpenRouter Models</PresetLabel>
              <RefreshButton type="button" onClick={loadPresets} disabled={presetsLoading}>
                {presetsLoading ? 'Loading...' : 'Refresh'}
              </RefreshButton>
            </PresetHeader>

            {presetsError && <StatusText variant="error">{presetsError}</StatusText>}

            <SearchInput
              placeholder="Search models by name, ID, or provider..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              disabled={presetsLoading || presets.length === 0}
            />

            <ModelSelectWrapper>
              <Select
                value={selectedPresetId}
                onChange={e => handlePresetSelect(e.target.value)}
                disabled={presetsLoading || presets.length === 0}
              >
                <option value="">
                  {presetsLoading
                    ? 'Loading models...'
                    : presets.length === 0
                      ? 'No models available'
                      : `Select a model (${filteredPresets.length} available)`}
                </option>
                {filteredPresets.map(preset => (
                  <option key={preset.model_id} value={preset.model_id}>
                    {preset.name} ({preset.provider}) - ${preset.input_cost_per_million}/$
                    {preset.output_cost_per_million} per 1M
                  </option>
                ))}
              </Select>
            </ModelSelectWrapper>

            <StatusText variant="info">
              Prices fetched from OpenRouter. Select a model to auto-fill pricing.
            </StatusText>

            <OrDivider>or enter manually</OrDivider>
          </PresetSection>
        )}

        <Row>
          <FormField label="Model ID" fullWidth>
            <Input
              value={formData.model_id}
              onChange={e => handleChange('model_id', e.target.value)}
              placeholder="e.g., gpt-4o"
              disabled={isEditing}
              required
            />
          </FormField>
          <FormField label="Provider" fullWidth>
            <Input
              value={formData.provider}
              onChange={e => handleChange('provider', e.target.value)}
              placeholder="e.g., openai"
              required
            />
          </FormField>
        </Row>
        <Row>
          <FormField
            label="Input Cost (per million tokens)"
            tooltip={`Cost in USD per million input (prompt) tokens sent to the model.\nThis includes the system prompt, user messages, and any context provided to the LLM.`}
            fullWidth
          >
            <Input
              type="number"
              step="0.0001"
              min="0"
              value={formData.input_cost_per_million}
              onChange={e =>
                handleChange('input_cost_per_million', parseFloat(e.target.value) || 0)
              }
              required
            />
          </FormField>
          <FormField
            label="Output Cost (per million tokens)"
            tooltip={`Cost in USD per million output (completion) tokens generated by the model.\nThis is typically higher than input cost as it reflects the computational cost of generating new text.`}
            fullWidth
          >
            <Input
              type="number"
              step="0.0001"
              min="0"
              value={formData.output_cost_per_million}
              onChange={e =>
                handleChange('output_cost_per_million', parseFloat(e.target.value) || 0)
              }
              required
            />
          </FormField>
        </Row>
        <Row>
          <FormField
            label="Cache Read Cost (per million)"
            tooltip={`Cost in USD per million tokens read from the prompt cache.\nWhen the same prompt prefix is reused, cached tokens are significantly cheaper than regular input tokens (typically 75-90% discount).`}
            fullWidth
          >
            <Input
              type="number"
              step="0.0001"
              min="0"
              value={formData.cache_read_cost_million || ''}
              onChange={e =>
                handleChange('cache_read_cost_million', parseFloat(e.target.value) || undefined)
              }
              placeholder="Optional"
            />
          </FormField>
          <FormField
            label="Cache Write Cost (per million)"
            tooltip={`Cost in USD per million tokens written to the prompt cache.\nThis is the cost to store prompt tokens for future reuse.\nSome providers charge a premium for cache writes, while reads are discounted.`}
            fullWidth
          >
            <Input
              type="number"
              step="0.0001"
              min="0"
              value={formData.cache_write_cost_million || ''}
              onChange={e =>
                handleChange('cache_write_cost_million', parseFloat(e.target.value) || undefined)
              }
              placeholder="Optional"
            />
          </FormField>
        </Row>
        <FormField label="Model Pattern (regex)" fullWidth>
          <Input
            value={formData.model_pattern || ''}
            onChange={e => handleChange('model_pattern', e.target.value || undefined)}
            placeholder="Optional regex pattern for matching models"
          />
        </FormField>
      </Form>
    </Modal>
  );
}
