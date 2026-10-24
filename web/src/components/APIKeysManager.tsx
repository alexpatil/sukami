import React, { useState } from 'react';
import { Lock, Shield, Copy, Trash2, Key as KeyIcon } from 'lucide-react';
import { APIKey, UserProfile } from '../types';

interface APIKeysManagerProps {
  user: UserProfile | null;
  keys: APIKey[];
  onOpenAuth: () => void;
  onRefreshKeys: () => void;
  onShowToast: (msg: string) => void;
}

export const APIKeysManager: React.FC<APIKeysManagerProps> = ({
  user,
  keys,
  onOpenAuth,
  onRefreshKeys,
  onShowToast,
}) => {
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [keyName, setKeyName] = useState('');
  const [keyTier, setKeyTier] = useState<'free' | 'pro'>('free');
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Success dialog state
  const [newlyCreatedKey, setNewlyCreatedKey] = useState<string | null>(null);

  const maskKey = (key: string) => {
    if (!key || key.length <= 16) return key;
    return `${key.substring(0, 12)}••••••••${key.substring(key.length - 4)}`;
  };

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!keyName.trim()) return;

    try {
      setIsSubmitting(true);
      const res = await fetch('/api/keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: keyName.trim(), tier: keyTier }),
      });

      if (res.ok) {
        const created = await res.json();
        setKeyName('');
        setIsCreateOpen(false);
        setNewlyCreatedKey(created.key);
        onRefreshKeys();
      } else {
        const err = await res.json();
        alert(`Failed to create key: ${err.error?.message || 'Server error'}`);
      }
    } catch (err: unknown) {
      alert(`Error creating key: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRevokeKey = async (key: string, name: string) => {
    if (!confirm(`Are you sure you want to revoke API key "${name}"? This action is permanent and cannot be undone.`)) {
      return;
    }

    try {
      const res = await fetch(`/api/keys?key=${encodeURIComponent(key)}`, {
        method: 'DELETE',
      });
      if (res.ok) {
        onShowToast(`API key "${name}" revoked.`);
        onRefreshKeys();
      } else {
        const err = await res.json();
        alert(`Failed to revoke key: ${err.error?.message || 'Server error'}`);
      }
    } catch (err: unknown) {
      alert(`Error revoking key: ${err instanceof Error ? err.message : String(err)}`);
    }
  };

  const copyToClipboard = (text: string, label: string = 'Key copied to clipboard!') => {
    navigator.clipboard.writeText(text);
    onShowToast(label);
  };

  // If user is not logged in: Show Auth Guarded Lock Card
  if (!user) {
    return (
      <div className="auth-gated-box">
        <div className="auth-gated-icon">
          <Lock size={44} color="#38bdf8" />
        </div>
        <h2 className="auth-gated-title">Authentication Required</h2>
        <p className="auth-gated-desc">
          Please sign in to provision, inspect, and revoke your Sukami API credentials.
        </p>
        <button className="btn-primary-action" onClick={onOpenAuth}>
          Sign In to Access Keys
        </button>
      </div>
    );
  }

  return (
    <div className="dashboard-card">
      <div className="card-title-row">
        <div>
          <h2 className="card-heading">
            <KeyIcon size={18} style={{ verticalAlign: 'middle', marginRight: 6, color: '#38bdf8' }} />
            Secret API Keys & Trial Quotas
          </h2>
          <p className="card-subheading">
            Your secret API keys authenticate requests to Sukami. Monitor your active trial token consumption and tier limits.
          </p>
        </div>
        <button className="btn-primary-action" onClick={() => setIsCreateOpen(true)}>
          + Create New Secret Key
        </button>
      </div>

      <div className="security-banner">
        <Shield size={18} color="#6366f1" />
        <div>
          <strong>Security Recommendation:</strong> Use dedicated keys per environment (e.g. <code>production</code>, <code>staging</code>) with automated trial quotas.
        </div>
      </div>

      <div className="table-container">
        <table className="activity-table">
          <thead>
            <tr>
              <th>Key Name</th>
              <th>Secret Key</th>
              <th>Tier</th>
              <th>Trial Quota Usage</th>
              <th>Rate Limit</th>
              <th>Total Requests</th>
              <th>Tokens Saved</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {keys.length === 0 ? (
              <tr>
                <td colSpan={9} className="empty-state">
                  No API keys created yet. Click "+ Create New Secret Key" above.
                </td>
              </tr>
            ) : (
              keys.map((k) => {
                const limit = k.quota_limit ?? 0;
                const used = k.quota_used ?? 0;
                const hasQuota = limit > 0;
                const pct = hasQuota ? Math.min(100, Math.round((used / limit) * 100)) : 0;
                const progressColor = pct >= 100 ? '#ef4444' : pct > 80 ? '#f59e0b' : '#38bdf8';

                return (
                  <tr key={k.key}>
                    <td><strong>{k.name}</strong></td>
                    <td><code className="key-code">{maskKey(k.key)}</code></td>
                    <td>
                      <span
                        className="badge-tag"
                        style={
                          k.tier === 'pro'
                            ? { color: '#38bdf8', borderColor: 'rgba(56,189,248,0.4)' }
                            : { color: '#10b981', borderColor: 'rgba(16,185,129,0.3)' }
                        }
                      >
                        {k.tier === 'free' ? '🎁 FREE TRIAL' : '⚡ PRO'}
                      </span>
                    </td>
                    <td>
                      {hasQuota ? (
                        <div style={{ minWidth: 140 }}>
                          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.72rem', marginBottom: 4 }}>
                            <span>{used.toLocaleString()} / {limit.toLocaleString()}</span>
                            <span style={{ color: progressColor, fontWeight: 600 }}>{pct}%</span>
                          </div>
                          <div style={{ height: 5, background: 'rgba(255,255,255,0.08)', borderRadius: 3, overflow: 'hidden' }}>
                            <div style={{ width: `${pct}%`, height: '100%', background: progressColor, transition: 'width 0.3s ease' }}></div>
                          </div>
                        </div>
                      ) : (
                        <span className="badge-tag" style={{ color: '#38bdf8', borderColor: 'rgba(56,189,248,0.3)', fontSize: '0.75rem' }}>
                          ∞ Unlimited
                        </span>
                      )}
                    </td>
                    <td>{k.rate_limit_per_min || 20} req/min</td>
                    <td>{(k.total_requests || 0).toLocaleString()}</td>
                    <td className="saved-highlight">+{(k.tokens_saved || 0).toLocaleString()}</td>
                    <td>{k.created_at ? new Date(k.created_at).toLocaleDateString() : 'Active'}</td>
                    <td>
                      <button
                        className="btn-copy-sm"
                        onClick={() => copyToClipboard(k.key)}
                        title="Copy Key"
                      >
                        <Copy size={12} style={{ verticalAlign: 'middle', marginRight: 3 }} />
                        Copy
                      </button>
                      <button
                        className="btn-danger-sm"
                        onClick={() => handleRevokeKey(k.key, k.name)}
                        title="Revoke Key"
                      >
                        <Trash2 size={12} style={{ verticalAlign: 'middle', marginRight: 3 }} />
                        Revoke
                      </button>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Modal 1: Create Key */}
      {isCreateOpen && (
        <div
          className="modal-backdrop"
          onClick={(e) => {
            if (e.target === e.currentTarget) setIsCreateOpen(false);
          }}
        >
          <div className="modal-card">
            <button className="modal-close-btn" onClick={() => setIsCreateOpen(false)}>
              &times;
            </button>
            <h3 className="modal-title">Create Secret Key</h3>
            <p className="modal-desc">
              Enter a name for this key and select your quota tier.
            </p>

            <form onSubmit={handleCreateKey}>
              <div className="form-group">
                <label htmlFor="create-key-name">Key Name</label>
                <input
                  id="create-key-name"
                  type="text"
                  placeholder="e.g. My App Backend, Cursor IDE"
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  autoFocus
                  required
                />
              </div>

              <div className="form-group">
                <label htmlFor="create-key-tier">Tier Plan & Quota</label>
                <select
                  id="create-key-tier"
                  value={keyTier}
                  onChange={(e) => setKeyTier(e.target.value as 'free' | 'pro')}
                >
                  <option value="free">🎁 Free Trial Tier (100,000 Free Tokens, 20 req/min)</option>
                  <option value="pro">⚡ Pro Tier (Unlimited Tokens, 120 req/min)</option>
                </select>
              </div>

              <div className="modal-btn-row">
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => setIsCreateOpen(false)}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn-primary-action"
                  disabled={isSubmitting}
                >
                  {isSubmitting ? 'Creating...' : 'Create Secret Key'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Modal 2: Success Key Display */}
      {newlyCreatedKey && (
        <div
          className="modal-backdrop"
          onClick={(e) => {
            if (e.target === e.currentTarget) setNewlyCreatedKey(null);
          }}
        >
          <div className="modal-card">
            <h3 className="modal-title">Save Your Secret Key</h3>
            <div className="alert-warning-box">
              <strong>⚠️ Please save this secret key now.</strong>
              <p>
                For your security, you will not be able to view this full key again. If you lose it, you will need to generate a new one.
              </p>
            </div>

            <div className="key-display-box">
              <input type="text" value={newlyCreatedKey} readOnly />
              <button
                className="btn-copy-action"
                onClick={() => copyToClipboard(newlyCreatedKey, 'Secret key copied!')}
              >
                Copy
              </button>
            </div>

            <div className="modal-btn-row" style={{ justifyContent: 'flex-end' }}>
              <button
                type="button"
                className="btn-primary-action"
                onClick={() => setNewlyCreatedKey(null)}
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
