import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { Copy, Download, Edit3, Clock, Code, Eye, Loader2, Link as LinkIcon, FileText, AlertTriangle, Flame } from 'lucide-react';
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
    // Store content in sessionStorage and navigate to editor
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
        <div className="flex flex-col items-center gap-4 animate-fade-in">
          <div className="relative">
            <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-[var(--primary)] to-[var(--secondary)] flex items-center justify-center animate-pulse-glow">
              <Loader2 className="w-8 h-8 text-white animate-spin" />
            </div>
          </div>
          <p className="text-[var(--text-muted)]">Loading paste...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center px-4">
        <div className="text-center max-w-md animate-fade-in">
          <div className="w-20 h-20 rounded-2xl bg-[var(--error)]/10 border border-[var(--error)]/30 flex items-center justify-center mx-auto mb-6">
            <AlertTriangle className="w-10 h-10 text-[var(--error)]" />
          </div>
          <h2 className="text-2xl font-bold text-[var(--text-main)] mb-3">
            Paste Not Found
          </h2>
          <p className="text-[var(--text-muted)] mb-8">
            {error === 'paste has expired'
              ? 'This paste has expired and is no longer available.'
              : 'The paste you are looking for does not exist or has been deleted.'}
          </p>
          <Link to="/" className="btn-primary inline-flex items-center gap-2">
            <FileText size={18} />
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
    <div className="flex-1 flex flex-col px-4 sm:px-6 lg:px-8 py-6">
      <div className="max-w-6xl mx-auto w-full animate-fade-in">
        {/* Header Card */}
        <div className="card p-4 sm:p-6 mb-4">
          <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            {/* Paste Info */}
            <div className="flex flex-wrap items-center gap-3">
              {/* Language badge */}
              <div className="badge badge-primary">
                <Code size={14} />
                <span className="capitalize">{paste.syntax_type || 'plaintext'}</span>
              </div>

              {/* Stats */}
              <div className="badge">
                <Eye size={14} />
                <span>{paste.view_count || 0} views</span>
              </div>

              <div className="badge">
                <FileText size={14} />
                <span>{lineCount} lines</span>
              </div>

              <div className="badge hidden sm:inline-flex">
                <span>{formatBytes(charCount)}</span>
              </div>

              {/* Expiration warning */}
              {paste.expires_at && (
                <div className="badge badge-warning">
                  <Clock size={14} />
                  <span>Expires in {getRelativeTime(paste.expires_at)}</span>
                </div>
              )}

              {/* Burn after read indicator */}
              {paste.burn_after_read && (
                <div className="badge" style={{ background: 'rgba(239, 68, 68, 0.1)', borderColor: 'rgba(239, 68, 68, 0.3)', color: '#EF4444' }}>
                  <Flame size={14} />
                  <span>Burns after read</span>
                </div>
              )}
            </div>

            {/* Action buttons */}
            <div className="flex items-center gap-2 flex-wrap">
              <button
                onClick={handleCopyLink}
                className="btn-ghost flex items-center gap-2 text-sm"
                title="Copy link"
              >
                <LinkIcon size={16} />
                <span className="hidden sm:inline">Copy Link</span>
              </button>
              <button
                onClick={handleCopyRaw}
                className="btn-ghost flex items-center gap-2 text-sm"
                title="Copy raw content"
              >
                <Copy size={16} />
                <span className="hidden sm:inline">Copy Raw</span>
              </button>
              <button
                onClick={handleDownload}
                className="btn-ghost flex items-center gap-2 text-sm"
                title="Download"
              >
                <Download size={16} />
                <span className="hidden sm:inline">Download</span>
              </button>
              <button
                onClick={handleClone}
                className="btn-primary flex items-center gap-2 text-sm"
                title="Clone and edit"
              >
                <Edit3 size={16} />
                <span>Clone</span>
              </button>
            </div>
          </div>

          {/* Created time */}
          <div className="mt-4 pt-4 border-t border-[var(--border)] flex items-center gap-2 text-sm text-[var(--text-muted)]">
            <Clock size={14} />
            <span>Created {formatDate(paste.created_at)}</span>
          </div>
        </div>

        {/* Code viewer */}
        <div className="card overflow-hidden animate-slide-up" style={{ animationDelay: '0.1s' }}>
          {/* Window Header */}
          <div className="code-header">
            <div className="code-dots">
              <div className="code-dot code-dot-red" />
              <div className="code-dot code-dot-yellow" />
              <div className="code-dot code-dot-green" />
            </div>
            <span className="text-xs text-[var(--text-muted)] font-mono">
              {paste.short_id}{getFileExtension(paste.syntax_type)}
            </span>
            <div className="w-[52px]" />
          </div>

          {/* Editor */}
          <div className="min-h-[50vh] lg:min-h-[60vh]">
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
