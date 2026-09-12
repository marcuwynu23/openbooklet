import { useEffect, useState } from 'react';
import { Sidebar } from './components/Sidebar';
import { SectionCard } from './components/SectionCard';
import { GeneratePanel } from './components/GeneratePanel';
import { AddSectionForm } from './components/AddSectionForm';
import { useBookletStore } from './stores';
import './styles.css';

function BookletHeader() {
  const booklet = useBookletStore((s) => s.booklet);
  const rename = useBookletStore((s) => s.rename);
  const remove = useBookletStore((s) => s.remove);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState('');

  if (booklet === null) return null;

  async function commit(): Promise<void> {
    const next = title.trim();
    setEditing(false);
    if (next !== '' && next !== booklet?.title) {
      await rename(next);
    }
  }

  function confirmDelete(): void {
    if (window.confirm(`Delete booklet "${booklet?.title}" and all its sections? This cannot be undone.`)) {
      void remove();
    }
  }

  return (
    <header className="booklet-head">
      <div className="booklet-title-row">
        {editing ? (
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onBlur={() => void commit()}
            onKeyDown={(e) => {
              if (e.key === 'Enter') void commit();
              if (e.key === 'Escape') setEditing(false);
            }}
            aria-label="Booklet title"
            className="booklet-title-input"
            autoFocus
          />
        ) : (
          <>
            <h1>{booklet.title}</h1>
            <button
              type="button"
              title="Rename booklet"
              aria-label="Rename booklet"
              className="icon-btn"
              onClick={() => {
                setTitle(booklet.title);
                setEditing(true);
              }}
            >
              ✎
            </button>
          </>
        )}
        <span className="spacer" />
        <button type="button" className="danger-btn" onClick={confirmDelete}>
          Delete
        </button>
      </div>
      <span className="muted">
                {booklet.type} · {booklet.status} · {booklet.sections.length} sections
      </span>
    </header>
  );
}

export function App() {
  const loadBooklets = useBookletStore((s) => s.loadBooklets);
  const booklet = useBookletStore((s) => s.booklet);
  const loading = useBookletStore((s) => s.loading);
  const error = useBookletStore((s) => s.error);

  useEffect(() => {
    void loadBooklets();
  }, [loadBooklets]);

  return (
    <div className="layout">
      <Sidebar />
      <main className="main">
        {loading && <p className="muted">Loading…</p>}
        {error !== null && <p className="error">Error: {error}</p>}
        {booklet === null && !loading && error === null && (
          <p className="muted">Select a booklet, or create one to get started.</p>
        )}
        {booklet !== null && (
          <>
            <BookletHeader />
            {booklet.sections.length === 0 && (
              <p className="muted">No sections yet — describe the document below and generate them.</p>
            )}
            <GeneratePanel bookletId={booklet.id} />
            {booklet.sections.map((section, i) => (
              <SectionCard key={section.id} bookletId={booklet.id} section={section} index={i + 1} />
            ))}
            <AddSectionForm bookletId={booklet.id} sections={booklet.sections} />
          </>
        )}
      </main>
    </div>
  );
}
