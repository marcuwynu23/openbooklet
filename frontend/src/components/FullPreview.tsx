import { useMemo, useState } from 'react';
import type { Booklet } from '../api';
import { renderMarkdown } from '../markdown';
import { useBookletStore } from '../stores';

function formatDate(value: string): string {
  if (value === '') return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

// FullPreview renders the whole booklet as one composed document: a header
// block with dates, description and metadata, the title plus every section
// in document order, and an optional footer.
export function FullPreview({ booklet }: { booklet: Booklet }) {
  const updateDetails = useBookletStore((s) => s.updateDetails);
  const [showFooter, setShowFooter] = useState(true);
  const [editingDesc, setEditingDesc] = useState(false);
  const [description, setDescription] = useState('');

  const html = useMemo(() => {
    const parts = [`# ${booklet.title}`];
    for (const section of booklet.sections) {
      parts.push(`${'#'.repeat(Math.min(Math.max(section.level, 1), 6))} ${section.title}`);
      if (section.content.trim() !== '') {
        parts.push(section.content.replace(/\s+$/, ''));
      }
    }
    return renderMarkdown(parts.join('\n\n') + '\n');
  }, [booklet]);

  async function saveDescription(): Promise<void> {
    setEditingDesc(false);
    if (description !== booklet.instructions) {
      await updateDetails({ instructions: description });
    }
  }

  const chips: Array<[string, string]> = [
    ['Updated', formatDate(booklet.updatedAt)],
    ['Created', formatDate(booklet.createdAt)],
    ['Type', booklet.type === '' ? '—' : booklet.type],
    ['Status', booklet.status],
  ];
  if (booklet.audience !== '') {
    chips.push(['Audience', booklet.audience]);
  }

  return (
    <article className="section full-preview">
      <header className="doc-header">
        <dl className="doc-meta">
          {chips.map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{value}</dd>
            </div>
          ))}
        </dl>
        <div className="doc-description">
          {editingDesc ? (
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              onBlur={() => void saveDescription()}
              onKeyDown={(e) => {
                if (e.key === 'Escape') setEditingDesc(false);
              }}
              rows={3}
              aria-label="Booklet description"
              autoFocus
            />
          ) : (
            <p>
              {booklet.instructions === '' ? (
                <i className="muted">No description yet.</i>
              ) : (
                booklet.instructions
              )}{' '}
              <button
                type="button"
                className="icon-btn"
                title="Edit description"
                aria-label="Edit description"
                onClick={() => {
                  setDescription(booklet.instructions);
                  setEditingDesc(true);
                }}
              >
                ✎
              </button>
            </p>
          )}
        </div>
        <label className="footer-toggle">
          <input type="checkbox" checked={showFooter} onChange={(e) => setShowFooter(e.target.checked)} />
          Show footer
        </label>
      </header>
      <div className="markdown" dangerouslySetInnerHTML={{ __html: html }} />
      {showFooter && (
        <footer className="doc-footer">
          {booklet.title} · Updated {formatDate(booklet.updatedAt)} · Generated with OpenBooklet
        </footer>
      )}
    </article>
  );
}
