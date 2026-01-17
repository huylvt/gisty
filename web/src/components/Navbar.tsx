import { Link, useLocation } from 'react-router-dom';
import { Plus, Moon, Sun, LogIn } from 'lucide-react';
import { useState } from 'react';

export function Navbar() {
  const location = useLocation();
  const [isDark, setIsDark] = useState(true);
  const isHome = location.pathname === '/';

  return (
    <nav className="glass sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link to="/" className="flex items-center gap-2 group">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-[var(--primary)] to-[var(--accent)] flex items-center justify-center">
              <span className="text-white font-bold text-lg">G</span>
            </div>
            <span className="text-xl font-bold text-[var(--text-main)] group-hover:text-[var(--primary)] transition-colors">
              Gisty
            </span>
          </Link>

          {/* Right side */}
          <div className="flex items-center gap-3">
            {!isHome && (
              <Link to="/" className="btn-primary flex items-center gap-2">
                <Plus size={18} />
                <span>New Paste</span>
              </Link>
            )}

            <button className="btn-ghost flex items-center gap-2">
              <LogIn size={18} />
              <span className="hidden sm:inline">Login</span>
            </button>

            <button
              onClick={() => setIsDark(!isDark)}
              className="btn-icon"
              title={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
            >
              {isDark ? <Sun size={20} /> : <Moon size={20} />}
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
}
