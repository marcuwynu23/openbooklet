import { useState } from 'react';
import { useBookletStore } from '../stores';

function MenuIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M3 5h14M3 10h14M3 15h14" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  );
}

function PlusIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M10 4v12M4 10h12" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  );
}

const COLLAPSE_KEY = 'obk-sidebar-collapsed';

export function Sidebar() {
  const booklets = useBookletStore((s) => s.booklets);
  const selectedId = useBookletStore((s) => s.selectedId);
  const select = useBookletStore((s) => s.select);
  const create = useBookletStore((s) => s.create);
  const [title, setTitle] = useState('');
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return window.localStorage.getItem(COLLAPSE_KEY) === '1';
    } catch {
      return false;
    }
  });

  function toggle(): void {
    setCollapsed((prev) => {
      try {
        window.localStorage.setItem(COLLAPSE_KEY, prev ? '0' : '1');
      } catch {
        // Storage unavailable; collapse for this session only.
      }
      return !prev;
    });
  }

  async function submit(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    const name = title.trim();
    if (name === '') return;
    setTitle('');
    await create(name, 'general');
  }

  return (
    <aside className={collapsed ? 'sidebar collapsed' : 'sidebar'}>
      <div className="sidebar-top">
        <button
          type="button"
          className="icon-btn"
          title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          onClick={toggle}
        >
          <MenuIcon />
        </button>
        {!collapsed && <h1>OpenBooklet</h1>}
      </div>
      {collapsed ? (
        <button type="button" className="icon-btn" title="New booklet" aria-label="New booklet" onClick={() => void create('Untitled Booklet', 'general')}>
          <PlusIcon />
        </button>
      ) : (
        <>
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
                <li key={b.id} title={b.title}>
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
        </>
      )}
    </aside>
  );
}
