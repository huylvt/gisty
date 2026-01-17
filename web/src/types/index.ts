export interface Paste {
  short_id: string;
  content: string;
  syntax_type: string;
  created_at: string;
  expires_at?: string;
  view_count?: number;
}

export interface CreatePasteRequest {
  content: string;
  syntax_type?: string;
  expires_in?: string;
  is_private?: boolean;
  burn_after_read?: boolean;
}

export interface CreatePasteResponse {
  short_id: string;
  url: string;
  expires_at?: string;
}

export interface ApiError {
  error: string;
  max_size?: string;
  retry_after?: number;
}

export const EXPIRATION_OPTIONS = [
  { value: '', label: 'Never' },
  { value: '10m', label: '10 Minutes' },
  { value: '1h', label: '1 Hour' },
  { value: '1d', label: '1 Day' },
  { value: '1w', label: '1 Week' },
  { value: '1M', label: '1 Month' },
  { value: 'burn', label: 'Burn After Read' },
] as const;

export const LANGUAGE_OPTIONS = [
  { value: '', label: 'Auto-detect' },
  { value: 'plaintext', label: 'Plain Text' },
  { value: 'javascript', label: 'JavaScript' },
  { value: 'typescript', label: 'TypeScript' },
  { value: 'python', label: 'Python' },
  { value: 'go', label: 'Go' },
  { value: 'rust', label: 'Rust' },
  { value: 'java', label: 'Java' },
  { value: 'c', label: 'C' },
  { value: 'cpp', label: 'C++' },
  { value: 'csharp', label: 'C#' },
  { value: 'php', label: 'PHP' },
  { value: 'ruby', label: 'Ruby' },
  { value: 'swift', label: 'Swift' },
  { value: 'kotlin', label: 'Kotlin' },
  { value: 'html', label: 'HTML' },
  { value: 'css', label: 'CSS' },
  { value: 'scss', label: 'SCSS' },
  { value: 'json', label: 'JSON' },
  { value: 'yaml', label: 'YAML' },
  { value: 'xml', label: 'XML' },
  { value: 'markdown', label: 'Markdown' },
  { value: 'sql', label: 'SQL' },
  { value: 'shell', label: 'Shell/Bash' },
  { value: 'dockerfile', label: 'Dockerfile' },
] as const;
