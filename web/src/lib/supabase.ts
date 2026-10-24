import { createClient, SupabaseClient } from '@supabase/supabase-js';

let client: SupabaseClient | null = null;

export async function getSupabase(): Promise<SupabaseClient | null> {
  if (client) return client;

  try {
    const res = await fetch('/api/config');
    if (!res.ok) return null;
    const config = await res.json();

    if (!config.supabase_url || !config.supabase_anon_key) {
      return null;
    }

    client = createClient(config.supabase_url, config.supabase_anon_key);
    return client;
  } catch (err) {
    console.warn('Failed to load Supabase config:', err);
    return null;
  }
}
