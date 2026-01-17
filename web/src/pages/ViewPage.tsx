import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { Copy, Download, Edit3, Clock, Code, Eye, Loader2 } from 'lucide-react';
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
    return date.toLocaleString();
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
        <div className="flex items-center gap-3 text-[var(--text-main)] opacity-60">
          <Loader2 className="animate-spin" size={24} />
          <span>Loading paste...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-[var(--text-main)] mb-2">
            Paste Not Found
          </h2>
          <p className="text-[var(--text-main)] opacity-60 mb-6">
            {error === 'paste has expired'
              ? 'This paste has expired and is no longer available.'
              : 'The paste you are looking for does not exist or has been deleted.'}
          </p>
          <Link to="/" className="btn-primary">
            Create New Paste
          </Link>
        </div>
      </div>
    );
  }

  if (!paste) return null;

  return (
    <div className="flex-1 flex flex-col p-4 sm:p-6 lg:p-8">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4 mb-4">
        {/* Paste info */}
        <div className="flex flex-wrap items-center gap-4 text-sm text-[var(--text-main)]">
          <div className="flex items-center gap-1.5 opacity-70">
            <Code size={16} />
            <span className="capitalize">{paste.syntax_type || 'plaintext'}</span>
          </div>
          <div className="flex items-center gap-1.5 opacity-70">
            <Clock size={16} />
            <span>{formatDate(paste.created_at)}</span>
          </div>
          {paste.view_count !== undefined && (
            <div className="flex items-center gap-1.5 opacity-70">
              <Eye size={16} />
              <span>{paste.view_count} views</span>
            </div>
          )}
          {paste.expires_at && (
            <div className="flex items-center gap-1.5 text-yellow-400">
              <Clock size={16} />
              <span>Expires: {formatDate(paste.expires_at)}</span>
            </div>
          )}
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-2">
          <button
            onClick={handleCopyLink}
            className="btn-ghost flex items-center gap-2 text-sm"
            title="Copy link"
          >
            <Copy size={16} />
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

      {/* Code viewer */}
      <div className="editor-container flex-1 min-h-[60vh]">
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
  );
}
