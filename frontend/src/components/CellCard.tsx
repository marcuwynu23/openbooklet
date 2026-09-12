import { useEffect, useState } from 'react';
import { marked } from 'marked';
import { api, type RegenerateMode, type Section } from '../api';
import { useBookletStore } from '../stores';

marked.setOptions({ breaks: true });

function statusColor(status: string): string {
  switch (status) {
    case 'approved':
      return '#1a7f37';
    case 'reviewed':
      return '#8250df';
    case 'generated':
      return '#0969da';
    case 'edited':
      return '#9a6700';
    default:
      return '#57606a';
  }
}

function renderMarkdown(content: string): string {
  const html = marked.parse(content, { async: false });
  return typeof html === 'string' ? html : '';
}

type Tab = 'preview' | 'edit';

export function CellCard({ bookletId, section }: { bookletId: string; section: Section }) {
  const refresh = useBookletStore((s) => s.refresh);
  const [tab, setTab] = useState<Tab>('preview');
  const [title, setTitle] = useState(section.title);
  const [content, setContent] = useState(section.content);
  const [saving, setSaving] = useState(false);
  const [aiBusy, setAiBusy] = useState<RegenerateMode | null>(null);
  const [instruction, setInstruction] = useState('');
  const [showEdit, setShowEdit] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setTitle(section.title);
    setContent(section.content);
  }, [section.id, section.title, section.content]);

  const dirty = title !== section.title || content !== section.content;

  async function save(): Promise<void> {
    setSaving(true);
    setError(null);
    try {
      await api.updateSection(bookletId, section.id, { title, content });
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'save failed');
    } finally {
      setSaving(false);
    }
  }

  async function ai(mode: RegenerateMode): Promise<void> {
    if (aiBusy !== null) return;
    if (mode === 'edit' && instruction.trim() === '') return;
    setAiBusy(mode);
    setError(null);
    try {
      await api.regenerateSection(bookletId, section.id, { mode, instruction });
      setShowEdit(false);
      setInstruction('');
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'AI action failed');
    } finally {
      setAiBusy(null);
    }
  }

  return (
    <article className="cell">
      <header className="cell-head">
        <span className="cell-level">{'#'.repeat(Math.min(section.level, 6))}</span>
        <h2>{section.title}</h2>
        <span className="badge" style={{ background: statusColor(section.status) }}>
          {section.status}
        </span>
      </header>
      {section.prompt !== '' && (
        <details className="prompt">
          <summary>Prompt</summary>
          <pre>{section.prompt}</pre>
        </details>
      )}
      <div className="tabs" role="tablist">
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'preview'}
          className={tab === 'preview' ? 'active' : ''}
          onClick={() => setTab('preview')}
        >
          Preview
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'edit'}
          className={tab === 'edit' ? 'active' : ''}
          onClick={() => setTab('edit')}
        >
          Edit{dirty ? ' ●' : ''}
        </button>
      </div>
      {tab === 'preview' ? (
        <div
          className="markdown"
          dangerouslySetInnerHTML={{
            __html: section.content === '' ? '<i>(empty)</i>' : renderMarkdown(section.content),
          }}
        />
      ) : (
        <>
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            aria-label="Section title"
            className="title-input"
          />
          <textarea
            className="editor"
            value={content}
            onChange={(e) => setContent(e.target.value)}
            rows={Math.max(6, content.split('\n').length + 1)}
          />
        </>
      )}
      <footer className="cell-foot">
        <button type="button" disabled={!dirty || saving} onClick={() => void save()}>
          {saving ? 'Saving…' : 'Save'}
        </button>
        <span className="ai-group">
          <button type="button" disabled={aiBusy !== null} onClick={() => void ai('regenerate')}>
            {aiBusy === 'regenerate' ? 'Working…' : 'Regenerate'}
          </button>
          <button type="button" disabled={aiBusy !== null} onClick={() => void ai('expand')}>
            {aiBusy === 'expand' ? 'Working…' : 'Expand'}
          </button>
          <button type="button" disabled={aiBusy !== null} onClick={() => void ai('shorten')}>
            {aiBusy === 'shorten' ? 'Working…' : 'Shorten'}
          </button>
          <button type="button" disabled={aiBusy !== null} onClick={() => setShowEdit((v) => !v)}>
            AI edit
          </button>
        </span>
        {error !== null && <span className="error">{error}</span>}
      </footer>
      {showEdit && (
        <div className="cell-foot">
          <input
            value={instruction}
            onChange={(e) => setInstruction(e.target.value)}
            placeholder="e.g. Use bullet points and add an example…"
            aria-label="AI edit instruction"
            className="ai-input"
          />
          <button
            type="button"
            disabled={instruction.trim() === '' || aiBusy !== null}
            onClick={() => void ai('edit')}
          >
            {aiBusy === 'edit' ? 'Working…' : 'Apply'}
          </button>
        </div>
      )}
    </article>
  );
}
