import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { Copy, Download, Edit3, Clock, Code, Eye, Loader2, Link as LinkIcon, AlertTriangle, Flame, Plus } from 'lucide-react';
import toast from 'react-hot-toast';
import { api } from '../services/api';
import type { Paste } from '../types';

export function ViewPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [paste, setPaste] = useState<Paste | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;

    const fetchPaste = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const data = await api.getPaste(id);
        setPaste(data);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to load paste';
        setError(message);
      } finally {
        setIsLoading(false);
      }
    };

    fetchPaste();
  }, [id]);

  const handleCopyRaw = async () => {
    if (!paste) return;
    try {
      await navigator.clipboard.writeText(paste.content);
      toast.success('Copied to clipboard!');
    } catch {
      toast.error('Failed to copy');
    }
  };

  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      toast.success('Link copied to clipboard!');
    } catch {
      toast.error('Failed to copy link');
    }
  };

  const handleDownload = () => {
    if (!paste) return;

    const extension = getFileExtension(paste.syntax_type);
    const filename = `${paste.short_id}${extension}`;
    const blob = new Blob([paste.content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);

    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();

    URL.revokeObjectURL(url);
    toast.success('Download started!');
  };

  const handleClone = () => {
    if (!paste) return;
    sessionStorage.setItem('gisty_clone_content', paste.content);
    sessionStorage.setItem('gisty_clone_language', paste.syntax_type);
    navigate('/');
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getRelativeTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diff = date.getTime() - now.getTime();

    if (diff < 0) return 'Expired';

    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ${hours % 24}h`;
    if (hours > 0) return `${hours}h`;

    const minutes = Math.floor(diff / (1000 * 60));
    return `${minutes}m`;
  };

  const getFileExtension = (syntaxType: string): string => {
    const extensions: Record<string, string> = {
      javascript: '.js',
      typescript: '.ts',
      python: '.py',
      go: '.go',
      rust: '.rs',
      java: '.java',
      c: '.c',
      cpp: '.cpp',
      csharp: '.cs',
      php: '.php',
      ruby: '.rb',
      swift: '.swift',
      kotlin: '.kt',
      html: '.html',
      css: '.css',
      scss: '.scss',
      json: '.json',
      yaml: '.yaml',
      xml: '.xml',
      markdown: '.md',
      sql: '.sql',
      shell: '.sh',
      dockerfile: '.dockerfile',
      plaintext: '.txt',
    };
    return extensions[syntaxType] || '.txt';
  };

  const getMonacoLanguage = (syntaxType: string) => {
    const mapping: Record<string, string> = {
      shell: 'shell',
      cpp: 'cpp',
      csharp: 'csharp',
    };
    return mapping[syntaxType] || syntaxType || 'plaintext';
  };

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-[var(--primary)] to-[var(--secondary)] flex items-center justify-center">
            <Loader2 className="w-6 h-6 text-white animate-spin" />
          </div>
          <p className="text-[var(--text-muted)]">Loading paste...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center px-4">
        <div className="text-center max-w-md">
          <div className="w-16 h-16 rounded-xl bg-[var(--error)]/10 border border-[var(--error)]/30 flex items-center justify-center mx-auto mb-4">
            <AlertTriangle className="w-8 h-8 text-[var(--error)]" />
          </div>
          <h2 className="text-xl font-bold text-[var(--text-main)] mb-2">
            Paste Not Found
          </h2>
          <p className="text-sm text-[var(--text-muted)] mb-6">
            {error === 'paste has expired'
              ? 'This paste has expired and is no longer available.'
              : 'The paste you are looking for does not exist or has been deleted.'}
          </p>
          <Link to="/" className="btn-primary inline-flex items-center gap-2">
            <Plus size={18} />
            Create New Paste
          </Link>
        </div>
      </div>
    );
  }

  if (!paste) return null;

  const lineCount = paste.content.split('\n').length;
  const charCount = paste.content.length;

  return (
    <div className="flex-1 w-full flex flex-col">
      <div className="flex-1 w-full px-6 sm:px-8 lg:px-12 py-6 flex flex-col">
        {/* Toolbar */}
        <div className="flex flex-wrap items-center gap-3 mb-4">
          {/* Language Badge */}
          <div className="badge badge-primary">
            <Code size={14} />
            <span className="capitalize">{paste.syntax_type || 'Plain Text'}</span>
          </div>

          {/* Stats */}
          <div className="badge">
            <Eye size={14} />
            <span>{paste.view_count || 0} views</span>
          </div>

          <div className="hidden sm:flex badge">
            <Clock size={14} />
            <span>{formatDate(paste.created_at)}</span>
          </div>

          {/* Expiration warning */}
          {paste.expires_at && (
            <div className="badge badge-warning">
              <Clock size={14} />
              <span>Expires in {getRelativeTime(paste.expires_at)}</span>
            </div>
          )}

          {/* Burn warning */}
          {paste.burn_after_read && (
            <div className="badge badge-error">
              <Flame size={14} />
              <span>Burns after read</span>
            </div>
          )}

          {/* Spacer */}
          <div className="flex-1" />

          {/* Action Buttons */}
          <div className="flex items-center gap-2">
            <button
              onClick={handleCopyLink}
              className="btn-icon"
              title="Copy Link"
            >
              <LinkIcon size={16} />
            </button>
            <button
              onClick={handleCopyRaw}
              className="btn-icon"
              title="Copy Raw"
            >
              <Copy size={16} />
            </button>
            <button
              onClick={handleDownload}
              className="btn-icon"
              title="Download"
            >
              <Download size={16} />
            </button>
            <button
              onClick={handleClone}
              className="btn-primary flex items-center gap-2"
            >
              <Edit3 size={16} />
              <span className="hidden sm:inline">Clone & Edit</span>
            </button>
          </div>
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
              {paste.short_id}{getFileExtension(paste.syntax_type)}
            </span>
            <span className="text-xs text-[var(--text-muted)]">
              {lineCount} lines · {formatBytes(charCount)}
            </span>
          </div>

          {/* Monaco Editor */}
          <div className="h-[calc(100vh-320px)] min-h-[300px]">
            <Editor
              height="100%"
              language={getMonacoLanguage(paste.syntax_type)}
              value={paste.content}
              theme="vs-dark"
              options={{
                readOnly: true,
                fontSize: 14,
                fontFamily: "'JetBrains Mono', 'Fira Code', Consolas, monospace",
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                lineNumbers: 'on',
                renderLineHighlight: 'none',
                padding: { top: 16, bottom: 16 },
                automaticLayout: true,
                wordWrap: 'on',
                domReadOnly: true,
                cursorStyle: 'line-thin',
              }}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
