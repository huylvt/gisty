import { Heart, Zap } from 'lucide-react';

export function Footer() {
  return (
    <footer className="mt-auto border-t border-[var(--border)] bg-[var(--bg-card)]/30">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
          {/* Logo & tagline */}
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-[var(--primary)] via-[var(--secondary)] to-[var(--accent)] flex items-center justify-center">
              <Zap className="w-4 h-4 text-white" strokeWidth={2.5} />
            </div>
            <div className="text-sm text-[var(--text-muted)]">
              <span className="font-semibold text-[var(--text-main)]">Gisty</span>
              {' '}&mdash; Fast code sharing for developers
            </div>
          </div>

          {/* Links & copyright */}
          <div className="flex items-center gap-6 text-sm text-[var(--text-muted)]">
            <a
              href="https://github.com/huylvt/gisty"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-[var(--primary)] transition-colors"
            >
              GitHub
            </a>
            <span className="flex items-center gap-1">
              Made with <Heart size={14} className="text-[var(--accent)]" /> by{' '}
              <a
                href="https://github.com/huylvt"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-[var(--primary)] transition-colors"
              >
                huylvt
              </a>
            </span>
          </div>
        </div>
      </div>
    </footer>
  );
}
