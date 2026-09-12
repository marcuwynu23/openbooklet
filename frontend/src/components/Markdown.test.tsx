import { describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { Markdown } from './Markdown';

vi.mock('mermaid', () => ({
  default: {
    initialize: vi.fn(),
    render: vi.fn(async (id: string, text: string) => {
      if (text.includes('INVALID')) {
        throw new Error('bad diagram');
      }
      return { svg: `<svg data-mock="${id}"></svg>` };
    }),
  },
}));

describe('Markdown', () => {
  it('renders plain Markdown without diagrams', () => {
    const { container } = render(<Markdown content={'# Title\n\nSome **bold** text.\n'} />);
    expect(screen.getByRole('heading', { name: 'Title' })).toBeInTheDocument();
    expect(screen.getByText('bold')).toBeInTheDocument();
    expect(container.querySelector('svg')).toBeNull();
  });

  it('renders empty content with placeholder text', () => {
    render(<Markdown content="" emptyText="Nothing here" />);
    expect(screen.getByText('Nothing here')).toBeInTheDocument();
  });

  it('replaces mermaid fences with diagrams', async () => {
    const { container } = render(
      <Markdown content={'# Doc\n\n```mermaid\nflowchart TD\n  A-->B\n```\n'} />,
    );
    await waitFor(() => {
      expect(container.querySelector('.mermaid-diagram svg')).not.toBeNull();
    });
    expect(container.querySelector('pre')).toBeNull();
  });

  it('keeps source visible when a diagram is invalid', async () => {
    const { container } = render(
      <Markdown content={'```mermaid\nINVALID SYNTAX HERE\n```\n'} />,
    );
    await waitFor(() => {
      expect(screen.getByText(/Could not render this diagram/)).toBeInTheDocument();
    });
    expect(container.querySelector('pre.mermaid-error')).not.toBeNull();
  });
});
