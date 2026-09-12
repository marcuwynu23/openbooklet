import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SectionCard } from './SectionCard';
import { bookletRoutes, makeBooklet, makeSection, mockFetch } from '../test-utils';

describe('SectionCard', () => {
  it('renders the numbered header and Markdown preview', () => {
    const section = makeSection({ content: '**bold** text\n' });
    render(<SectionCard bookletId="b1" section={section} index={1} />);
    expect(screen.getByText('1. ##')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Purpose' })).toBeInTheDocument();
    expect(screen.getByText('bold')).toBeInTheDocument();
    expect(screen.getByText('draft')).toBeInTheDocument();
  });

  it('collapses and expands like an accordion', async () => {
    const user = userEvent.setup();
    const section = makeSection({ content: 'Visible body\n' });
    render(<SectionCard bookletId="b1" section={section} index={2} />);
    expect(screen.getByText('2. ##')).toBeInTheDocument();
    expect(screen.getByText('Visible body')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Collapse section Purpose' }));
    expect(screen.queryByText('Visible body')).not.toBeInTheDocument();
    expect(screen.getByText('Purpose')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Expand section Purpose' }));
    expect(screen.getByText('Visible body')).toBeInTheDocument();
  });

  it('saves edited content through the API', async () => {
    const user = userEvent.setup();
    const booklet = makeBooklet();
    const calls = mockFetch((url, init) => {
      const base = bookletRoutes(booklet)(url, init);
      if (base !== undefined) return base;
      if (url === '/api/v1/booklets/b1/sections/s1' && init?.method === 'PUT') {
        const patch = JSON.parse(String(init.body)) as { content?: string };
        return {
          status: 200,
          body: { data: makeSection({ content: patch.content ?? '' }) },
        };
      }
      return undefined;
    });

    render(<SectionCard bookletId="b1" section={makeSection()} index={1} />);
    await user.click(screen.getByRole('tab', { name: /Edit/ }));
    const editor = screen.getByRole('textbox');
    await user.clear(editor);
    await user.type(editor, 'Rewritten body.');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const put = calls.find((c) => c.init?.method === 'PUT');
    expect(put).toBeDefined();
    expect(String(put?.init?.body)).toContain('Rewritten body.');
  });

  it('renames inline from the header', async () => {
    const user = userEvent.setup();
    const booklet = makeBooklet();
    const calls = mockFetch((url, init) => {
      const base = bookletRoutes(booklet)(url, init);
      if (base !== undefined) return base;
      if (url === '/api/v1/booklets/b1/sections/s1' && init?.method === 'PUT') {
        return { status: 200, body: { data: makeSection() } };
      }
      return undefined;
    });

    render(<SectionCard bookletId="b1" section={makeSection()} index={1} />);
    await user.click(screen.getByRole('heading', { name: 'Purpose' }));
    const input = screen.getByLabelText('Section title');
    await user.clear(input);
    await user.type(input, 'Goal{Enter}');

    const put = calls.find((c) => c.init?.method === 'PUT');
    expect(put).toBeDefined();
    expect(String(put?.init?.body)).toContain('"title":"Goal"');
  });

  it('changes the heading level through the API', async () => {
    const user = userEvent.setup();
    const booklet = makeBooklet();
    const calls = mockFetch((url, init) => {
      const base = bookletRoutes(booklet)(url, init);
      if (base !== undefined) return base;
      if (url === '/api/v1/booklets/b1/sections/s1' && init?.method === 'PUT') {
        return { status: 200, body: { data: makeSection({ level: 3 }) } };
      }
      return undefined;
    });

    render(<SectionCard bookletId="b1" section={makeSection()} index={1} />);
    await user.selectOptions(screen.getByLabelText('Heading level for section Purpose'), '3');

    const put = calls.find((c) => c.init?.method === 'PUT');
    expect(put).toBeDefined();
    expect(String(put?.init?.body)).toContain('"level":3');
  });

  it('deletes the section after confirmation', async () => {
    const user = userEvent.setup();
    const booklet = makeBooklet();
    const calls = mockFetch((url, init) => {
      const base = bookletRoutes(booklet)(url, init);
      if (base !== undefined) return base;
      if (url === '/api/v1/booklets/b1/sections/s1' && init?.method === 'DELETE') {
        return { status: 204, body: null };
      }
      return undefined;
    });
    const realConfirm = window.confirm;
    window.confirm = () => true;
    try {
      render(<SectionCard bookletId="b1" section={makeSection()} index={1} />);
      await user.click(screen.getByRole('button', { name: 'Delete' }));
    } finally {
      window.confirm = realConfirm;
    }

    expect(calls.some((c) => c.init?.method === 'DELETE')).toBe(true);
  });

  it('keeps the section when deletion is cancelled', async () => {
    const user = userEvent.setup();
    const booklet = makeBooklet();
    const calls = mockFetch(bookletRoutes(booklet));
    const realConfirm = window.confirm;
    window.confirm = () => false;
    try {
      render(<SectionCard bookletId="b1" section={makeSection()} index={1} />);
      await user.click(screen.getByRole('button', { name: 'Delete' }));
    } finally {
      window.confirm = realConfirm;
    }

    expect(calls.some((c) => c.init?.method === 'DELETE')).toBe(false);
  });
});
