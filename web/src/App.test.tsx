import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { App } from './App';

describe('Sukami Frontend App & Auth Guard', () => {
  beforeEach(() => {
    // Mock global fetch
    vi.stubGlobal('fetch', vi.fn((url: string) => {
      if (url.includes('/api/config')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ supabase_url: '', supabase_anon_key: '' }),
        });
      }
      if (url.includes('/api/metrics')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            total_tokens_saved: 3090,
            savings_percentage: 46.3,
            estimated_cost_saved_usd: 0.0093,
            cache_hits: 1,
            cache_hit_rate: 33.3,
            total_requests: 3,
            recent_requests: [],
          }),
        });
      }
      if (url.includes('/api/keys')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve([]),
        });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
    }));
  });

  it('renders brand logo and navigation tabs', async () => {
    render(<App />);

    expect(screen.getByText('Sukami')).toBeInTheDocument();
    expect(screen.getByText('Overview')).toBeInTheDocument();
    expect(screen.getByText('API Keys')).toBeInTheDocument();
    expect(screen.getByText('Playground')).toBeInTheDocument();
    expect(screen.getByText('Docs')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
  });

  it('enforces Auth Guard: displays lock card when unauthenticated user accesses API Keys', async () => {
    render(<App />);

    // Click API Keys tab
    const keysTab = screen.getByText('API Keys');
    fireEvent.click(keysTab);

    // Should show Auth Guard card, NOT the secret keys vault!
    expect(screen.getByText('Authentication Required')).toBeInTheDocument();
    expect(
      screen.getByText(/Please sign in to provision, inspect, and revoke your Sukami API credentials/i)
    ).toBeInTheDocument();
    expect(screen.getByText('Sign In to Access Keys')).toBeInTheDocument();
  });

  it('renders the interactive playground with instant token estimation', async () => {
    render(<App />);

    const playgroundTab = screen.getByText('Playground');
    fireEvent.click(playgroundTab);

    expect(screen.getByText('🧪 Interactive Token Optimizer Playground')).toBeInTheDocument();
    expect(screen.getByText('⚡ Optimize Prompt')).toBeInTheDocument();
  });

  it('renders integration documentation code tabs', async () => {
    render(<App />);

    const docsTab = screen.getByText('Docs');
    fireEvent.click(docsTab);

    expect(screen.getByText('🔌 1-Line Drop-in Integration')).toBeInTheDocument();
    expect(screen.getByText('Python')).toBeInTheDocument();
    expect(screen.getByText('Node.js')).toBeInTheDocument();
    expect(screen.getByText('cURL')).toBeInTheDocument();
  });
});
