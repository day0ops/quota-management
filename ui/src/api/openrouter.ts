// OpenRouter API for fetching model pricing data

export interface OpenRouterModel {
  id: string;
  name: string;
  description?: string;
  context_length: number;
  pricing: {
    prompt: string;
    completion: string;
    input_cache_read?: string;
    input_cache_write?: string;
  };
  top_provider?: {
    is_moderated: boolean;
  };
  architecture?: {
    modality: string;
    tokenizer: string;
  };
}

export interface OpenRouterResponse {
  data: OpenRouterModel[];
}

export interface ModelPreset {
  model_id: string;
  name: string;
  provider: string;
  input_cost_per_million: number;
  output_cost_per_million: number;
  cache_read_cost_million?: number;
  cache_write_cost_million?: number;
  context_length: number;
}

function extractProvider(modelId: string): string {
  // OpenRouter model IDs are formatted as "provider/model-name"
  const parts = modelId.split('/');
  return parts.length > 1 ? parts[0] : 'unknown';
}

function parseTokenCost(cost: string | undefined): number {
  if (!cost) return 0;
  const parsed = parseFloat(cost);
  return isNaN(parsed) ? 0 : parsed;
}

function convertToPerMillion(perTokenCost: number): number {
  // Convert per-token cost to per-million tokens
  // Round to 4 decimal places for cleaner display
  return Math.round(perTokenCost * 1_000_000 * 10000) / 10000;
}

export async function fetchOpenRouterModels(): Promise<ModelPreset[]> {
  const response = await fetch('https://openrouter.ai/api/v1/models');

  if (!response.ok) {
    throw new Error(`Failed to fetch models: ${response.statusText}`);
  }

  const data: OpenRouterResponse = await response.json();

  return data.data
    .filter(model => {
      // Filter out models with zero or missing pricing
      const promptCost = parseTokenCost(model.pricing?.prompt);
      const completionCost = parseTokenCost(model.pricing?.completion);
      return promptCost > 0 || completionCost > 0;
    })
    .map((model): ModelPreset => {
      const promptCost = parseTokenCost(model.pricing.prompt);
      const completionCost = parseTokenCost(model.pricing.completion);
      const cacheReadCost = parseTokenCost(model.pricing.input_cache_read);
      const cacheWriteCost = parseTokenCost(model.pricing.input_cache_write);

      return {
        model_id: model.id,
        name: model.name,
        provider: extractProvider(model.id),
        input_cost_per_million: convertToPerMillion(promptCost),
        output_cost_per_million: convertToPerMillion(completionCost),
        cache_read_cost_million: cacheReadCost > 0 ? convertToPerMillion(cacheReadCost) : undefined,
        cache_write_cost_million:
          cacheWriteCost > 0 ? convertToPerMillion(cacheWriteCost) : undefined,
        context_length: model.context_length,
      };
    })
    .sort((a, b) => a.name.localeCompare(b.name));
}
