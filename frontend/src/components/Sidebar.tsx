import { useState } from 'react';
import { useBookletStore } from '../stores';

export function Sidebar() {
  const booklets = useBookletStore((s) => s.booklets);
  const selectedId = useBookletStore((s) => s.selectedId);
  const select = useBookletStore((s) => s.select);
  const create = useBookletStore((s) => s.create);
  const [title, setTitle] = useState('');

  async function submit(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    const name = title.trim();
    if (name === '') return;
    setTitle('');
    await create(name, 'general');
  }

  return (
    <aside className="sidebar">
      <h1>OpenBooklet</h1>
      <form onSubmit={(e) => void submit(e)} className="new-form">
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="New booklet title…"
          aria-label="New booklet title"
        />
        <button type="submit">New</button>
      </form>
      <nav>
        {booklets.length === 0 && <p className="muted">No booklets yet.</p>}
        <ul>
          {booklets.map((b) => (
            <li key={b.id}>
              <button
                type="button"
                className={b.id === selectedId ? 'active' : ''}
                onClick={() => void select(b.id)}
              >
                {b.title}
                <span className="count">{b.sections}</span>
              </button>
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  );
}
