import type { CreatePasteRequest, CreatePasteResponse, Paste, ApiError } from '../types';

const API_BASE = '/api/v1';

class ApiService {
  private async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });

    const data = await response.json();

    if (!response.ok) {
      const error = data as ApiError;
      throw new Error(error.error || 'An error occurred');
    }

    return data as T;
  }

  async createPaste(req: CreatePasteRequest): Promise<CreatePasteResponse> {
    return this.request<CreatePasteResponse>('/pastes', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async getPaste(shortId: string): Promise<Paste> {
    return this.request<Paste>(`/pastes/${shortId}`);
  }

  async deletePaste(shortId: string): Promise<void> {
    await fetch(`${API_BASE}/pastes/${shortId}`, {
      method: 'DELETE',
    });
  }

  async getRawPaste(shortId: string): Promise<string> {
    const response = await fetch(`/${shortId}`, {
      headers: {
        Accept: 'text/plain',
      },
    });

    if (!response.ok) {
      throw new Error('Paste not found');
    }

    return response.text();
  }
}

export const api = new ApiService();
