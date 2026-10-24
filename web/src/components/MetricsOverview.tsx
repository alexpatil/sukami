import React from 'react';
import { MetricsSummary } from '../types';

interface MetricsOverviewProps {
  metrics: MetricsSummary | null;
}

const fmt = (n?: number) => (typeof n === 'number' ? n.toLocaleString() : '0');
const fmtPct = (n?: number) => (typeof n === 'number' ? n.toFixed(1) : '0.0');
const fmtCost = (n?: number) => (typeof n === 'number' ? `$${n.toFixed(4)}` : '$0.00');

export const MetricsOverview: React.FC<MetricsOverviewProps> = ({ metrics }) => {
  const recentRequests = metrics?.recent_requests || [];

  return (
    <>
      <div className="hero-section">
        <div style={{ marginBottom: 16 }}>
          <span
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 8,
              padding: '6px 14px',
              borderRadius: 20,
              background: 'rgba(56, 189, 248, 0.1)',
              border: '1px solid rgba(56, 189, 248, 0.25)',
              color: '#38bdf8',
              fontSize: '0.82rem',
              fontWeight: 600,
              letterSpacing: '0.02em',
            }}
          >
            <span>🎁</span> 100,000 Free Trial Tokens on Signup • Instant Endpoint via <code>sukami.io</code>
          </span>
        </div>
        <h1 className="hero-title">
          Slash AI Token Costs by <span className="gradient-text">30% to 100%</span>
        </h1>
        <p className="hero-subtitle">
          Input prompt pruning, high-density output optimization, real-time semantic caching, and instant OpenAI drop-in acceleration.
        </p>
      </div>

      <div className="metrics-grid">
        <div className="metric-card">
          <div className="card-header">
            <span className="metric-title">Total Tokens Saved</span>
            <span className="metric-icon">📉</span>
          </div>
          <div className="metric-value-container">
            <span className="metric-value">
              {fmt(metrics?.total_saved_tokens)}
            </span>
            <span className="savings-pill">
              +{fmtPct(metrics?.overall_savings_pct)}%
            </span>
          </div>
          <div className="metric-subtext" style={{ display: 'flex', gap: 10, marginTop: 4 }}>
            <span style={{ color: '#38bdf8' }}>In: +{fmt(metrics?.total_input_saved_tokens)}</span>
            <span>•</span>
            <span style={{ color: '#a855f7' }}>Out: +{fmt(metrics?.total_output_saved_tokens)}</span>
          </div>
        </div>

        <div className="metric-card">
          <div className="card-header">
            <span className="metric-title">Estimated Cost Saved</span>
            <span className="metric-icon">💰</span>
          </div>
          <div className="metric-value-container">
            <span className="metric-value text-emerald">
              {fmtCost(metrics?.estimated_cost_saved_usd)}
            </span>
          </div>
          <div className="metric-subtext">Direct infrastructure savings</div>
        </div>

        <div className="metric-card">
          <div className="card-header">
            <span className="metric-title">Semantic Cache Hit Rate</span>
            <span className="metric-icon">⚡</span>
          </div>
          <div className="metric-value-container">
            <span className="metric-value text-cyan">
              {fmtPct(metrics?.cache_hit_rate)}%
            </span>
            <span className="hits-counter">
              {fmt(metrics?.cache_hits)} hits
            </span>
          </div>
          <div className="metric-subtext">Instant zero-cost cached queries</div>
        </div>

        <div className="metric-card">
          <div className="card-header">
            <span className="metric-title">Total API Requests</span>
            <span className="metric-icon">🌐</span>
          </div>
          <div className="metric-value-container">
            <span className="metric-value">
              {fmt(metrics?.total_requests)}
            </span>
          </div>
          <div className="metric-subtext">Ultra low-latency throughput</div>
        </div>
      </div>

      <div className="dashboard-card">
        <div className="card-title-row">
          <div>
            <h2 className="card-heading">⚡ Real-Time API Stream</h2>
            <p className="card-subheading">
              Live observability of token optimization, response caching, and latency across your models.
            </p>
          </div>
          <span className="live-indicator">
            <span className="pulse-dot"></span> Live
          </span>
        </div>

        <div className="table-container">
          <table className="activity-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Model</th>
                <th>Request Preview</th>
                <th>Original</th>
                <th>Saved</th>
                <th>Billed</th>
                <th>Efficiency</th>
                <th>Status</th>
                <th>Latency</th>
              </tr>
            </thead>
            <tbody>
              {recentRequests.length === 0 ? (
                <tr>
                  <td colSpan={9} className="empty-state">
                    Waiting for incoming API requests...
                  </td>
                </tr>
              ) : (
                recentRequests.slice(0, 10).map((r) => (
                  <tr key={r.id}>
                    <td>{r.timestamp ? new Date(r.timestamp).toLocaleTimeString() : '-'}</td>
                    <td><code>{r.model || 'gpt-4o'}</code></td>
                    <td style={{ maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {r.prompt_preview || '-'}
                    </td>
                    <td>{fmt(r.original_tokens)}</td>
                    <td className="saved-highlight">+{fmt(r.saved_tokens)}</td>
                    <td>{fmt(r.billed_tokens)}</td>
                    <td>{fmtPct(r.savings_percent)}%</td>
                    <td>
                      <span
                        className="badge-tag"
                        style={
                          r.cache_hit
                            ? { color: '#38bdf8', borderColor: 'rgba(56,189,248,0.4)' }
                            : { color: '#10b981', borderColor: 'rgba(16,185,129,0.3)' }
                        }
                      >
                        {r.cache_hit ? 'CACHE HIT' : 'OPTIMIZED'}
                      </span>
                    </td>
                    <td>{r.latency_ms ?? 0}ms</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
};
