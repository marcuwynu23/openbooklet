import { useEffect } from 'react';
import { Sidebar } from './components/Sidebar';
import { CellCard } from './components/CellCard';
import { GeneratePanel } from './components/GeneratePanel';
import { useBookletStore } from './stores';
import './styles.css';

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
            <header className="booklet-head">
              <h1>{booklet.title}</h1>
              <span className="muted">
                {booklet.type} · {booklet.status} · {booklet.sections.length} cells
              </span>
            </header>
            {booklet.sections.length === 0 && (
              <p className="muted">No cells yet — describe the document below and generate them.</p>
            )}
            <GeneratePanel bookletId={booklet.id} />
            {booklet.sections.map((section) => (
              <CellCard key={section.id} bookletId={booklet.id} section={section} />
            ))}
          </>
        )}
      </main>
    </div>
  );
}
