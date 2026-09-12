import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AddSectionForm } from './AddSectionForm';
import { bookletRoutes, makeBooklet, makeSection, mockFetch } from '../test-utils';

describe('AddSectionForm', () => {
  it('offers H1 through H6 levels', async () => {
    const user = userEvent.setup();
    render(<AddSectionForm bookletId="b1" sections={[]} />);
    await user.click(screen.getByRole('button', { name: '＋ Add section' }));
    const select = screen.getByLabelText('Heading level');
    for (const label of ['H1', 'H2', 'H3', 'H4', 'H5', 'H6']) {
      expect(select.textContent ?? '').toContain(label);
    }
  });

  it('nests under the nearest preceding lower-level section', async () => {
    const user = userEvent.setup();
    const booklet = makeBooklet();
    const sections = [
      makeSection({ id: 'a', title: 'Top', level: 1 }),
      makeSection({ id: 'b', title: 'Mid', level: 2 }),
    ];
    const calls = mockFetch((url, init) => {
      const base = bookletRoutes(booklet)(url, init);
      if (base !== undefined) return base;
      if (url === '/api/v1/booklets/b1/sections' && init?.method === 'POST') {
        return { status: 201, body: { data: makeSection() } };
      }
      return undefined;
    });

    render(<AddSectionForm bookletId="b1" sections={sections} />);
    await user.click(screen.getByRole('button', { name: '＋ Add section' }));
    await user.type(screen.getByLabelText('New section title'), 'Deep');
    await user.selectOptions(screen.getByLabelText('Heading level'), '3');
    await user.click(screen.getByRole('button', { name: 'Add' }));

    const post = calls.find((c) => c.init?.method === 'POST');
    expect(post).toBeDefined();
    expect(String(post?.init?.body)).toContain('"parentId":"b"');
  });

  it('creates top-level sections without a parent', async () => {
    const user = userEvent.setup();
    const booklet = makeBooklet();
    const calls = mockFetch((url, init) => {
      const base = bookletRoutes(booklet)(url, init);
      if (base !== undefined) return base;
      if (url === '/api/v1/booklets/b1/sections' && init?.method === 'POST') {
        return { status: 201, body: { data: makeSection() } };
      }
      return undefined;
    });

    render(<AddSectionForm bookletId="b1" sections={[makeSection({ id: 'a', level: 2 })]} />);
    await user.click(screen.getByRole('button', { name: '＋ Add section' }));
    await user.type(screen.getByLabelText('New section title'), 'Fresh');
    await user.selectOptions(screen.getByLabelText('Heading level'), '1');
    await user.click(screen.getByRole('button', { name: 'Add' }));

    const post = calls.find((c) => c.init?.method === 'POST');
    expect(post).toBeDefined();
    expect(String(post?.init?.body)).toContain('"parentId":null');
  });
});
