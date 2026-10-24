import React, { useState } from 'react';
import { getSupabase } from '../lib/supabase';

interface AuthModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AuthModal: React.FC<AuthModalProps> = ({ isOpen, onClose }) => {
  const [loadingProvider, setLoadingProvider] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleOAuth = async (provider: 'github' | 'google') => {
    try {
      setLoadingProvider(provider);
      const supabase = await getSupabase();
      if (!supabase) {
        alert('Supabase credentials not available.');
        setLoadingProvider(null);
        return;
      }

      const { error } = await supabase.auth.signInWithOAuth({
        provider,
        options: {
          redirectTo: window.location.origin,
        },
      });

      if (error) {
        alert(`${provider} login failed: ${error.message}`);
        setLoadingProvider(null);
      }
    } catch (err: unknown) {
      alert(`Login error: ${err instanceof Error ? err.message : String(err)}`);
      setLoadingProvider(null);
    }
  };

  return (
    <div
      className="modal-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="modal-card">
        <button className="modal-close-btn" onClick={onClose}>
          &times;
        </button>

        <div className="modal-header-centered">
          <div className="modal-logo">⚡</div>
          <h3 className="modal-title">Sign in to Sukami</h3>
          <p className="modal-desc">
            Access your dashboard, provision secure API credentials, and monitor real-time token savings.
          </p>
        </div>

        <div className="modal-oauth-group">
          <button
            className="btn-oauth-provider"
            disabled={loadingProvider !== null}
            onClick={() => handleOAuth('github')}
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
            </svg>
            <span>{loadingProvider === 'github' ? 'Connecting GitHub...' : 'Continue with GitHub'}</span>
          </button>

          <button
            className="btn-oauth-provider"
            disabled={loadingProvider !== null}
            onClick={() => handleOAuth('google')}
          >
            <svg width="18" height="18" viewBox="0 0 24 24">
              <path fill="#EA4335" d="M12 5c1.6 0 3 .6 4.1 1.7l3.1-3.1C17.3 1.8 14.8 1 12 1 7.4 1 3.5 3.6 1.6 7.4l3.7 2.9C6.2 7.4 8.9 5 12 5z" />
              <path fill="#4285F4" d="M23.5 12.3c0-.8-.1-1.6-.2-2.3H12v4.6h6.5c-.3 1.5-1.1 2.8-2.4 3.7l3.7 2.9c2.2-2 3.7-5 3.7-8.9z" />
              <path fill="#FBBC05" d="M5.3 14.7c-.2-.7-.4-1.5-.4-2.7s.1-2 .4-2.7L1.6 6.4C.6 8.3 0 10.1 0 12s.6 3.7 1.6 5.6l3.7-2.9z" />
              <path fill="#34A853" d="M12 23c3.2 0 6-1.1 8-3l-3.7-2.9c-1.1.7-2.5 1.2-4.3 1.2-3.1 0-5.8-2.4-6.7-5.3L1.6 15.9C3.5 19.8 7.4 23 12 23z" />
            </svg>
            <span>{loadingProvider === 'google' ? 'Connecting Google...' : 'Continue with Google'}</span>
          </button>
        </div>

        <div className="modal-footer-text">
          <span>Powered by Supabase Auth</span>
        </div>
      </div>
    </div>
  );
};
