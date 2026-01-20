import { useState, useCallback, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { Send, Clock, Code, Flame } from 'lucide-react';
import toast from 'react-hot-toast';
import { api } from '../services/api';
import { LoadingBar } from '../components/LoadingBar';
import { EXPIRATION_OPTIONS, LANGUAGE_OPTIONS } from '../types';

export function EditorPage() {
  const navigate = useNavigate();
  const [content, setContent] = useState('');
  const [language, setLanguage] = useState('');
  const [expiration, setExpiration] = useState('1d');
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
      const response = await api.createPaste({
        content,
        syntax_type: language || undefined,
        expires_in: expiration || undefined,
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
    <div className="flex-1 w-full flex flex-col">
      <LoadingBar isLoading={isLoading} />

      <div className="flex-1 w-full px-6 sm:px-8 lg:px-12 py-6 flex flex-col">
        {/* Toolbar */}
        <div className="flex flex-wrap items-center gap-4 mb-4">
          {/* Language selector */}
          <div className="flex items-center gap-2">
            <Code size={16} className="text-[var(--text-muted)]" />
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
            <Clock size={16} className="text-[var(--text-muted)]" />
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

          {/* Burn warning */}
          {expiration === 'burn' && (
            <div className="flex items-center gap-1.5 text-xs text-[var(--error)]">
              <Flame size={14} />
              <span>Burns after read</span>
            </div>
          )}

          {/* Spacer */}
          <div className="flex-1" />

          {/* Keyboard shortcut hint */}
          <div className="hidden md:flex items-center gap-1.5 text-xs text-[var(--text-muted)]">
            <kbd>Ctrl</kbd>
            <span>+</span>
            <kbd>S</kbd>
          </div>

          {/* Submit Button */}
          <button
            onClick={handleSave}
            disabled={isLoading || !content.trim()}
            className="btn-primary flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
          >
            <Send size={16} />
            <span>Create Paste</span>
          </button>
        </div>

        {/* Editor */}
        <div className="flex-1 card overflow-hidden">
          {/* Editor Header */}
          <div className="code-header">
            <div className="code-dots">
              <div className="code-dot code-dot-red" />
              <div className="code-dot code-dot-yellow" />
              <div className="code-dot code-dot-green" />
            </div>
            <span className="text-xs text-[var(--text-muted)] font-mono">
              {language ? `untitled.${getFileExtension(language)}` : 'untitled.txt'}
            </span>
            <div className="w-[52px]" />
          </div>

          {/* Monaco Editor */}
          <div className="h-[calc(100vh-220px)] min-h-[400px]">
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
                readOnly: false,
                domReadOnly: false,
              }}
            />
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
