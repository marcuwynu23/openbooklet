// Service layer: every HTTP call goes through here. Components never fetch directly.

export interface BookletSummary {
  id: string;
  title: string;
  type: string;
  status: string;
  sections: number;
  updatedAt: string;
}

export interface Section {
  id: string;
  parentId: string | null;
  title: string;
  level: number;
  prompt: string;
  content: string;
  status: string;
  updatedAt: string;
}

export interface Booklet {
  id: string;
  title: string;
  type: string;
  status: string;
  audience: string;
  instructions: string;
  sections: Section[];
  updatedAt: string;
}

interface Envelope<T> {
  data: T;
  error?: { code: string; message: string };
}

export type RegenerateMode = 'regenerate' | 'expand' | 'shorten' | 'edit';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  });
  const body = (await res.json()) as Envelope<T>;
  if (!res.ok || body.error !== undefined) {
    throw new Error(body.error?.message ?? `request failed: ${res.status}`);
  }
  return body.data;
}

export const api = {
  version(): Promise<{ version: string; provider: string }> {
    return request('/api/v1/version');
  },
  booklets(): Promise<BookletSummary[]> {
    return request('/api/v1/booklets');
  },
  booklet(id: string): Promise<Booklet> {
    return request(`/api/v1/booklets/${encodeURIComponent(id)}`);
  },
  createBooklet(title: string, docType: string): Promise<Booklet> {
    return request('/api/v1/booklets', {
      method: 'POST',
      body: JSON.stringify({ title, type: docType }),
    });
  },
  renameBooklet(id: string, title: string): Promise<Booklet> {
    return request(`/api/v1/booklets/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify({ title }),
    });
  },
  async deleteBooklet(id: string): Promise<void> {
    const res = await fetch(`/api/v1/booklets/${encodeURIComponent(id)}`, { method: 'DELETE' });
    if (!res.ok) {
      throw new Error(`delete failed: ${res.status}`);
    }
  },
  createSection(
    bookletId: string,
    req: { title: string; level: number; parentId?: string | null; prompt?: string; content?: string },
  ): Promise<Section> {
    return request(`/api/v1/booklets/${encodeURIComponent(bookletId)}/sections`, {
      method: 'POST',
      body: JSON.stringify(req),
    });
  },
  updateSection(
    bookletId: string,
    sectionId: string,
    patch: { title?: string; prompt?: string; content?: string },
  ): Promise<Section> {
    return request(
      `/api/v1/booklets/${encodeURIComponent(bookletId)}/sections/${encodeURIComponent(sectionId)}`,
      { method: 'PUT', body: JSON.stringify(patch) },
    );
  },
  regenerateSection(
    bookletId: string,
    sectionId: string,
    req: { mode: RegenerateMode; instruction?: string; prompt?: string },
  ): Promise<Section> {
    return request(
      `/api/v1/booklets/${encodeURIComponent(bookletId)}/sections/${encodeURIComponent(sectionId)}/regenerate`,
      { method: 'POST', body: JSON.stringify(req) },
    );
  },

  // generate streams tokens via onToken and resolves with the saved sections.
  async generate(
    bookletId: string,
    prompt: string,
    onToken: (token: string) => void,
  ): Promise<Section[]> {
    const res = await fetch(`/api/v1/booklets/${encodeURIComponent(bookletId)}/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt }),
    });
    if (res.status === 503) {
      throw new Error('No AI provider is configured on the server.');
    }
    if (!res.ok || res.body === null) {
      throw new Error(`generate failed: ${res.status}`);
    }
    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let eventType = '';
    const sections: Section[] = [];
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      let idx: number;
      while ((idx = buffer.indexOf('\n\n')) >= 0) {
        const frame = buffer.slice(0, idx);
        buffer = buffer.slice(idx + 2);
        let data = '';
        for (const line of frame.split('\n')) {
          if (line.startsWith('event: ')) eventType = line.slice(7).trim();
          else if (line.startsWith('data: ')) data += line.slice(6);
        }
        if (eventType === 'token' && data !== '') {
          onToken((JSON.parse(data) as { token: string }).token);
        } else if (eventType === 'complete') {
          const payload = JSON.parse(data) as { sections: Section[] };
          sections.push(...payload.sections);
        } else if (eventType === 'error') {
          throw new Error((JSON.parse(data) as { message: string }).message);
        }
      }
    }
    return sections;
  },
};
