import { useState, useCallback, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { Send, Clock, Code } from 'lucide-react';
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
    <div className="flex-1 flex flex-col p-4 sm:p-6 lg:p-8">
      <LoadingBar isLoading={isLoading} />

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-3 mb-4">
        {/* Language selector */}
        <div className="flex items-center gap-2">
          <Code size={18} className="text-[var(--text-main)] opacity-60" />
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

        {/* Expiration selector */}
        <div className="flex items-center gap-2">
          <Clock size={18} className="text-[var(--text-main)] opacity-60" />
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

        {/* Save button */}
        <button
          onClick={handleSave}
          disabled={isLoading || !content.trim()}
          className="btn-primary flex items-center gap-2 ml-auto disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Send size={18} />
          <span>Gistify</span>
        </button>
      </div>

      {/* Keyboard shortcut hint */}
      <div className="text-xs text-[var(--text-main)] opacity-40 mb-2">
        Press <kbd className="px-1.5 py-0.5 bg-[var(--editor-bg)] rounded text-[var(--primary)]">Ctrl+S</kbd> or{' '}
        <kbd className="px-1.5 py-0.5 bg-[var(--editor-bg)] rounded text-[var(--primary)]">Cmd+S</kbd> to save
      </div>

      {/* Editor */}
      <div className="editor-container flex-1 min-h-[60vh]">
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
          }}
        />
      </div>
    </div>
  );
}
