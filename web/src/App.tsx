import React, { useState, useEffect, useCallback } from 'react';
import { Navbar } from './components/Navbar';
import { AuthModal } from './components/AuthModal';
import { MetricsOverview } from './components/MetricsOverview';
import { APIKeysManager } from './components/APIKeysManager';
import { Playground } from './components/Playground';
import { Docs } from './components/Docs';
import { APIKey, MetricsSummary, UserProfile } from './types';
import { getSupabase } from './lib/supabase';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'overview' | 'keys' | 'playground' | 'docs'>('overview');
  const [isAuthOpen, setIsAuthOpen] = useState(false);
  const [user, setUser] = useState<UserProfile | null>(null);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [metrics, setMetrics] = useState<MetricsSummary | null>(null);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const showToast = useCallback((msg: string) => {
    setToastMessage(msg);
    setTimeout(() => setToastMessage(null), 2800);
  }, []);

  const fetchKeys = useCallback(async () => {
    try {
      const res = await fetch('/api/keys');
      if (res.ok) {
        const data = await res.json();
        setKeys(data || []);
      }
    } catch (err) {
      console.warn('Failed to fetch keys:', err);
    }
  }, []);

  const fetchMetrics = useCallback(async () => {
    try {
      const res = await fetch('/api/metrics');
      if (res.ok) {
        const data = await res.json();
        setMetrics(data);
      }
    } catch (err) {
      console.warn('Failed to fetch metrics:', err);
    }
  }, []);

  // Supabase Auth listener
  useEffect(() => {
    let unsubscribe = () => {};

    const setupAuth = async () => {
      const supabase = await getSupabase();
      if (!supabase) return;

      const { data: { session } } = await supabase.auth.getSession();
      if (session?.user) {
        const u = session.user;
        setUser({
          id: u.id,
          email: u.email,
          name: u.user_metadata?.full_name || u.user_metadata?.name || u.email?.split('@')[0] || 'User',
          avatarUrl: u.user_metadata?.avatar_url || `https://api.dicebear.com/7.x/identicon/svg?seed=${encodeURIComponent(u.id)}`,
        });
      }

      const { data: listener } = supabase.auth.onAuthStateChange((_event, newSession) => {
        if (newSession?.user) {
          const u = newSession.user;
          setUser({
            id: u.id,
            email: u.email,
            name: u.user_metadata?.full_name || u.user_metadata?.name || u.email?.split('@')[0] || 'User',
            avatarUrl: u.user_metadata?.avatar_url || `https://api.dicebear.com/7.x/identicon/svg?seed=${encodeURIComponent(u.id)}`,
          });
        } else {
          setUser(null);
        }
      });

      unsubscribe = () => listener.subscription.unsubscribe();
    };

    setupAuth();
    return () => unsubscribe();
  }, []);

  // Polling data
  useEffect(() => {
    let isMounted = true;
    const loadInitial = async () => {
      if (isMounted) {
        await fetchMetrics();
        await fetchKeys();
      }
    };
    void loadInitial();

    const metricsInterval = setInterval(fetchMetrics, 2000);
    const keysInterval = setInterval(fetchKeys, 5000);

    return () => {
      isMounted = false;
      clearInterval(metricsInterval);
      clearInterval(keysInterval);
    };
  }, [fetchMetrics, fetchKeys]);

  const handleSignOut = async () => {
    const supabase = await getSupabase();
    if (supabase) {
      await supabase.auth.signOut();
    }
    setUser(null);
    showToast('Signed out successfully.');
  };

  return (
    <>
      <div className="bg-glow glow-1" />
      <div className="bg-glow glow-2" />

      <Navbar
        activeTab={activeTab}
        setActiveTab={(t) => setActiveTab(t as 'overview' | 'keys' | 'playground' | 'docs')}
        user={user}
        onOpenAuth={() => setIsAuthOpen(true)}
        onSignOut={handleSignOut}
      />

      <main className="main-container">
        {activeTab === 'overview' && <MetricsOverview metrics={metrics} />}
        {activeTab === 'keys' && (
          <APIKeysManager
            user={user}
            keys={keys}
            onOpenAuth={() => setIsAuthOpen(true)}
            onRefreshKeys={fetchKeys}
            onShowToast={showToast}
          />
        )}
        {activeTab === 'playground' && <Playground />}
        {activeTab === 'docs' && <Docs />}
      </main>

      <AuthModal isOpen={isAuthOpen} onClose={() => setIsAuthOpen(false)} />

      {toastMessage && <div className="toast">{toastMessage}</div>}

      <footer className="footer">
        <p>Sukami AI • Enterprise Token Optimization & Acceleration Platform</p>
      </footer>
    </>
  );
};
