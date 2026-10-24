import React from 'react';
import { LayoutGrid, Key, Beaker, BookOpen, LogOut } from 'lucide-react';
import { UserProfile } from '../types';

interface NavbarProps {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  user: UserProfile | null;
  onOpenAuth: () => void;
  onSignOut: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  activeTab,
  setActiveTab,
  user,
  onOpenAuth,
  onSignOut,
}) => {
  return (
    <header className="navbar">
      <div className="nav-container">
        <div className="logo" onClick={() => setActiveTab('overview')}>
          <span className="logo-icon">⚡</span>
          <span className="logo-text">Sukami</span>
          <span className="badge-tag">AI Gateway</span>
        </div>

        <nav className="nav-tabs">
          <button
            className={`nav-tab ${activeTab === 'overview' ? 'active' : ''}`}
            onClick={() => setActiveTab('overview')}
          >
            <LayoutGrid size={15} />
            <span>Overview</span>
          </button>
          <button
            className={`nav-tab ${activeTab === 'keys' ? 'active' : ''}`}
            onClick={() => setActiveTab('keys')}
          >
            <Key size={15} />
            <span>API Keys</span>
          </button>
          <button
            className={`nav-tab ${activeTab === 'playground' ? 'active' : ''}`}
            onClick={() => setActiveTab('playground')}
          >
            <Beaker size={15} />
            <span>Playground</span>
          </button>
          <button
            className={`nav-tab ${activeTab === 'docs' ? 'active' : ''}`}
            onClick={() => setActiveTab('docs')}
          >
            <BookOpen size={15} />
            <span>Docs</span>
          </button>
        </nav>

        <div className="nav-auth-section">
          {user ? (
            <div className="user-pill">
              {user.avatarUrl && (
                <img src={user.avatarUrl} alt="Avatar" className="user-avatar" />
              )}
              <span className="user-name">{user.name}</span>
              <button
                className="btn-signout"
                onClick={onSignOut}
                title="Sign Out"
              >
                <LogOut size={13} style={{ verticalAlign: 'middle', marginRight: 3 }} />
                Sign Out
              </button>
            </div>
          ) : (
            <button className="btn-signin" onClick={onOpenAuth}>
              Sign In
            </button>
          )}
        </div>
      </div>
    </header>
  );
};
