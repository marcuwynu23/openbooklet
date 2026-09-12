import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Preview } from './Preview';
import { useBookletStore } from '../stores';
import { bookletRoutes, makeBooklet, mockFetch } from '../test-utils';

describe('Preview', () => {
  it('renders the header, body, and footer blocks', () => {
    const booklet = makeBooklet({
      header: 'Date: January 1, 2020\n',
      footer: 'The end.\n',
      showFooter: true,
    });
    render(<Preview booklet={booklet} />);
    expect(screen.getByText('Date: January 1, 2020')).toBeInTheDocument();
    expect(screen.getByText('The end.')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Guide' })).toBeInTheDocument();
  });

  it('hides the footer text when the toggle is off', () => {
    const booklet = makeBooklet({ footer: 'The end.\n', showFooter: false });
    render(<Preview booklet={booklet} />);
    expect(screen.getByText('Footer hidden.')).toBeInTheDocument();
    expect(screen.queryByText('The end.')).not.toBeInTheDocument();
  });

  it('persists the footer toggle through the API', async () => {
    const user = userEvent.setup();
    const booklet = makeBooklet({ footer: 'The end.\n', showFooter: true });
    useBookletStore.setState({ booklet, selectedId: booklet.id });
    const calls = mockFetch((url, init) => {
      const base = bookletRoutes(booklet)(url, init);
      if (base !== undefined) return base;
      if (url === '/api/v1/booklets/b1' && init?.method === 'PUT') {
        return { status: 200, body: { data: booklet } };
      }
      return undefined;
    });

    render(<Preview booklet={booklet} />);
    await user.click(screen.getByRole('checkbox', { name: 'Show' }));

    const put = calls.find((c) => c.init?.method === 'PUT');
    expect(put).toBeDefined();
    expect(String(put?.init?.body)).toContain('"showFooter":false');
  });

  it('saves an edited header', async () => {
    const user = userEvent.setup();
    const booklet = makeBooklet({ header: 'Date: January 1, 2020\n' });
    useBookletStore.setState({ booklet, selectedId: booklet.id });
    const calls = mockFetch((url, init) => {
      const base = bookletRoutes(booklet)(url, init);
      if (base !== undefined) return base;
      if (url === '/api/v1/booklets/b1' && init?.method === 'PUT') {
        return { status: 200, body: { data: booklet } };
      }
      return undefined;
    });

    render(<Preview booklet={booklet} />);
    await user.click(screen.getByRole('button', { name: 'Edit header' }));
    const editor = screen.getByLabelText('Booklet header Markdown');
    await user.clear(editor);
    await user.type(editor, 'Date: March 3, 2026');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    const put = calls.find((c) => c.init?.method === 'PUT');
    expect(put).toBeDefined();
    expect(String(put?.init?.body)).toContain('March 3, 2026');
  });
});
