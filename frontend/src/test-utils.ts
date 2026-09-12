import type { Booklet, Section } from './api';

// Shared fixtures and fetch mock for component tests.

export function makeSection(overrides: Partial<Section> = {}): Section {
  return {
    id: 's1',
    parentId: null,
    title: 'Purpose',
    level: 2,
    prompt: '',
    content: 'Why this exists.\n',
    status: 'draft',
    updatedAt: '2026-09-12T12:00:00Z',
    ...overrides,
  };
}

export function makeBooklet(overrides: Partial<Booklet> = {}): Booklet {
  return {
    id: 'b1',
    title: 'Guide',
    type: 'sop',
    status: 'draft',
    audience: '',
    instructions: '',
    header: '',
    footer: '',
    showFooter: false,
    sections: [makeSection()],
    createdAt: '2026-09-12T12:00:00Z',
    updatedAt: '2026-09-12T12:00:00Z',
    ...overrides,
  };
}

export interface FetchCall {
  url: string;
  init?: RequestInit;
}

export function mockFetch(
  handler: (url: string, init?: RequestInit) => { status: number; body: unknown } | undefined,
): FetchCall[] {
  const calls: FetchCall[] = [];
  globalThis.fetch = (async (url: unknown, init?: RequestInit) => {
    const urlString = String(url);
    calls.push({ url: urlString, init });
    const res = handler(urlString, init);
    if (res === undefined) {
      throw new Error(`unmocked fetch: ${urlString}`);
    }
    return new Response(JSON.stringify(res.body), {
      status: res.status,
      headers: { 'Content-Type': 'application/json' },
    });
  }) as typeof fetch;
  return calls;
}

export function bookletRoutes(booklet: Booklet) {
  return (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET';
    if (url === '/api/v1/booklets' && method === 'GET') {
      return { status: 200, body: { data: [] } };
    }
    if (url === `/api/v1/booklets/${booklet.id}` && method === 'GET') {
      return { status: 200, body: { data: booklet } };
    }
    return undefined;
  };
}
