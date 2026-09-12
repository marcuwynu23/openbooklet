import { useMemo, useState } from 'react';
import type { Booklet } from '../api';
import { renderMarkdown } from '../markdown';
import { useBookletStore } from '../stores';

function todayBlock(): string {
  const date = new Date().toLocaleDateString('en-US', {
    month: 'long',
    day: 'numeric',
    year: 'numeric',
  });
  return `Date: ${date}\n`;
}

function BlockEditor({
  label,
  initial,
  rows,
  onSave,
  onCancel,
}: {
  label: string;
  initial: string;
  rows: number;
  onSave: (value: string) => Promise<void>;
  onCancel: () => void;
}) {
  const [value, setValue] = useState(initial);
  const [busy, setBusy] = useState(false);
  return (
    <div>
      <textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        rows={rows}
        aria-label={label}
        className="editor"
        autoFocus
      />
      <div className="section-foot">
        <button
          type="button"
          disabled={busy}
          onClick={() => {
            setBusy(true);
            void onSave(value).finally(() => setBusy(false));
          }}
        >
          {busy ? 'Saving…' : 'Save'}
        </button>
        <button type="button" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </div>
  );
}

// Preview renders the whole booklet as one composed document: an editable
// Markdown header block, the title plus every section in document order,
// and an editable Markdown footer block with a show/hide toggle.
export function Preview({ booklet }: { booklet: Booklet }) {
  const updateDetails = useBookletStore((s) => s.updateDetails);
  const [editingHeader, setEditingHeader] = useState(false);
  const [editingFooter, setEditingFooter] = useState(false);

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

  async function saveHeader(value: string): Promise<void> {
    setEditingHeader(false);
    if (value !== booklet.header) {
      await updateDetails({ header: value });
    }
  }

  async function saveFooter(value: string): Promise<void> {
    setEditingFooter(false);
    if (value !== booklet.footer) {
      await updateDetails({ footer: value });
    }
  }

  return (
    <article className="section full-preview">
      <header className="doc-block">
        <div className="doc-block-head">
          <span className="muted">Header</span>
          {!editingHeader && (
            <button
              type="button"
              className="icon-btn"
              title="Edit header"
              aria-label="Edit header"
              onClick={() => setEditingHeader(true)}
            >
              ✎
            </button>
          )}
        </div>
        {editingHeader ? (
          <BlockEditor
            label="Booklet header Markdown"
            initial={booklet.header === '' ? todayBlock() : booklet.header}
            rows={4}
            onSave={saveHeader}
            onCancel={() => setEditingHeader(false)}
          />
        ) : booklet.header === '' ? (
          <p className="muted">
            No header yet — add one with a date like <code>Date: January 1, 2020</code> and a description.
          </p>
        ) : (
          <div className="markdown" dangerouslySetInnerHTML={{ __html: renderMarkdown(booklet.header) }} />
        )}
      </header>

      <div className="markdown" dangerouslySetInnerHTML={{ __html: html }} />

      <footer className="doc-block">
        <div className="doc-block-head">
          <span className="muted">Footer</span>
          <label className="footer-toggle">
            <input
              type="checkbox"
              checked={booklet.showFooter}
              onChange={(e) => void updateDetails({ showFooter: e.target.checked })}
            />
            Show
          </label>
          {!editingFooter && (
            <button
              type="button"
              className="icon-btn"
              title="Edit footer"
              aria-label="Edit footer"
              onClick={() => setEditingFooter(true)}
            >
              ✎
            </button>
          )}
        </div>
        {editingFooter ? (
          <BlockEditor
            label="Booklet footer Markdown"
            initial={booklet.footer}
            rows={3}
            onSave={saveFooter}
            onCancel={() => setEditingFooter(false)}
          />
        ) : booklet.showFooter && booklet.footer !== '' ? (
          <div className="markdown" dangerouslySetInnerHTML={{ __html: renderMarkdown(booklet.footer) }} />
        ) : (
          <p className="muted">{booklet.showFooter ? 'No footer text yet.' : 'Footer hidden.'}</p>
        )}
      </footer>
    </article>
  );
}
