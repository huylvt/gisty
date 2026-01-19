import { useState, useCallback, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { Send, Clock, Code, Sparkles, Command } from 'lucide-react';
import toast from 'react-hot-toast';
import { api } from '../services/api';
import { LoadingBar } from '../components/LoadingBar';
import { EXPIRATION_OPTIONS, LANGUAGE_OPTIONS } from '../types';

export function EditorPage() {
  const navigate = useNavigate();
  const [content, setContent] = useState('');
  const [language, setLanguage] = useState('');
  const [expiration, setExpiration] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  // Load cloned content from sessionStorage
  useEffect(() => {
    const clonedContent = sessionStorage.getItem('gisty_clone_content');
    const clonedLanguage = sessionStorage.getItem('gisty_clone_language');

    if (clonedContent) {
      setContent(clonedContent);
      sessionStorage.removeItem('gisty_clone_content');
    }
    if (clonedLanguage) {
      setLanguage(clonedLanguage);
      sessionStorage.removeItem('gisty_clone_language');
    }
  }, []);

  const handleSave = useCallback(async () => {
    if (!content.trim()) {
      toast.error('Please enter some content');
      return;
    }

    setIsLoading(true);
    try {
      const isBurn = expiration === 'burn';
      const response = await api.createPaste({
        content,
        syntax_type: language || undefined,
        expires_in: isBurn ? '1d' : expiration || undefined,
        burn_after_read: isBurn,
      });

      const fullUrl = `${window.location.origin}/${response.short_id}`;

      // Copy to clipboard
      await navigator.clipboard.writeText(fullUrl);
      toast.success('Link copied to clipboard!');

      // Navigate to view page
      navigate(`/${response.short_id}`);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to create paste';
      toast.error(message);
    } finally {
      setIsLoading(false);
    }
  }, [content, language, expiration, navigate]);

  // Keyboard shortcut: Ctrl/Cmd + S to save
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        handleSave();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleSave]);

  // Map language to Monaco language ID
  const getMonacoLanguage = (lang: string) => {
    const mapping: Record<string, string> = {
      shell: 'shell',
      cpp: 'cpp',
      csharp: 'csharp',
    };
    return mapping[lang] || lang || 'plaintext';
  };

  return (
    <div className="flex-1 flex flex-col">
      <LoadingBar isLoading={isLoading} />

      {/* Hero Section */}
      <div className="relative overflow-hidden py-8 sm:py-12 px-4 sm:px-6 lg:px-8">
        <div className="hero-orb hero-orb-1" />
        <div className="hero-orb hero-orb-2" />

        <div className="relative max-w-4xl mx-auto text-center animate-fade-in">
          <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-[var(--bg-card)]/50 border border-[var(--border)] mb-6">
            <Sparkles size={16} className="text-[var(--primary)]" />
            <span className="text-sm text-[var(--text-muted)]">Share code instantly, no account required</span>
          </div>

          <h1 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-4">
            <span className="text-[var(--text-main)]">Share Code </span>
            <span className="gradient-text">Lightning Fast</span>
          </h1>

          <p className="text-[var(--text-muted)] text-lg max-w-2xl mx-auto">
            Paste your code, get a shareable link. Syntax highlighting, expiration options, and burn-after-read support.
          </p>
        </div>
      </div>

      {/* Editor Section */}
      <div className="flex-1 px-4 sm:px-6 lg:px-8 pb-8">
        <div className="max-w-6xl mx-auto animate-slide-up" style={{ animationDelay: '0.1s' }}>
          {/* Toolbar */}
          <div className="toolbar flex-wrap mb-4">
            {/* Language selector */}
            <div className="flex items-center gap-2">
              <Code size={18} className="text-[var(--text-muted)]" />
              <select
                value={language}
                onChange={(e) => setLanguage(e.target.value)}
                className="select-custom"
              >
                {LANGUAGE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            <div className="divider hidden sm:block" />

            {/* Expiration selector */}
            <div className="flex items-center gap-2">
              <Clock size={18} className="text-[var(--text-muted)]" />
              <select
                value={expiration}
                onChange={(e) => setExpiration(e.target.value)}
                className="select-custom"
              >
                {EXPIRATION_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Keyboard shortcut hint */}
            <div className="hidden lg:flex items-center gap-2 text-xs text-[var(--text-muted)] ml-auto mr-4">
              <Command size={14} />
              <span>Press</span>
              <kbd>Ctrl</kbd>
              <span>+</span>
              <kbd>S</kbd>
              <span>to save</span>
            </div>

            {/* Save button */}
            <button
              onClick={handleSave}
              disabled={isLoading || !content.trim()}
              className="btn-primary flex items-center gap-2 ml-auto lg:ml-0 disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
            >
              <Send size={18} />
              <span>Gistify</span>
            </button>
          </div>

          {/* Editor with window chrome */}
          <div className="card overflow-hidden" style={{ animationDelay: '0.2s' }}>
            {/* Window Header */}
            <div className="code-header">
              <div className="code-dots">
                <div className="code-dot code-dot-red" />
                <div className="code-dot code-dot-yellow" />
                <div className="code-dot code-dot-green" />
              </div>
              <span className="text-xs text-[var(--text-muted)] font-mono">
                {language ? `untitled.${getFileExtension(language)}` : 'untitled.txt'}
              </span>
              <div className="w-[52px]" /> {/* Spacer for balance */}
            </div>

            {/* Editor */}
            <div className="min-h-[50vh] lg:min-h-[60vh]">
              <Editor
                height="100%"
                language={getMonacoLanguage(language)}
                value={content}
                onChange={(value) => setContent(value || '')}
                theme="vs-dark"
                options={{
                  fontSize: 14,
                  fontFamily: "'JetBrains Mono', 'Fira Code', Consolas, monospace",
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  lineNumbers: 'on',
                  renderLineHighlight: 'line',
                  padding: { top: 16, bottom: 16 },
                  automaticLayout: true,
                  wordWrap: 'on',
                  tabSize: 2,
                  cursorBlinking: 'smooth',
                  cursorSmoothCaretAnimation: 'on',
                  smoothScrolling: true,
                  placeholder: 'Paste your code here...',
                }}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function getFileExtension(language: string): string {
  const extensions: Record<string, string> = {
    javascript: 'js',
    typescript: 'ts',
    python: 'py',
    go: 'go',
    rust: 'rs',
    java: 'java',
    c: 'c',
    cpp: 'cpp',
    csharp: 'cs',
    php: 'php',
    ruby: 'rb',
    swift: 'swift',
    kotlin: 'kt',
    html: 'html',
    css: 'css',
    scss: 'scss',
    json: 'json',
    yaml: 'yaml',
    xml: 'xml',
    markdown: 'md',
    sql: 'sql',
    shell: 'sh',
    dockerfile: 'dockerfile',
    plaintext: 'txt',
  };
  return extensions[language] || 'txt';
}
