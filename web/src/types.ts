export interface APIKey {
  key: string;
  name: string;
  tier: 'free' | 'pro';
  rate_limit_per_min: number;
  created_at: string;
  total_requests: number;
  tokens_saved: number;
  quota_limit?: number;
  quota_used?: number;
}

export interface RequestRecord {
  id: string;
  timestamp: string;
  model: string;
  prompt_preview: string;
  original_tokens: number;
  saved_tokens: number;
  input_saved_tokens?: number;
  output_saved_tokens?: number;
  billed_tokens: number;
  savings_percent: number;
  cache_hit: boolean;
  latency_ms: number;
  status: string;
}

export interface MetricsSummary {
  total_requests: number;
  cache_hits: number;
  cache_misses: number;
  cache_hit_rate: number;
  total_original_tokens: number;
  total_saved_tokens: number;
  total_input_saved_tokens?: number;
  total_output_saved_tokens?: number;
  total_billed_tokens: number;
  overall_savings_pct: number;
  estimated_cost_saved_usd: number;
  recent_requests: RequestRecord[];
}

export interface UserProfile {
  id: string;
  email?: string;
  name: string;
  avatarUrl?: string;
}
