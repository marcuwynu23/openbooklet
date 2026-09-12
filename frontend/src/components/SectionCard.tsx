import { useEffect, useState } from 'react';
import { api, type RegenerateMode, type Section } from '../api';
import { renderMarkdown } from '../markdown';
import { useBookletStore } from '../stores';

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

type Tab = 'preview' | 'edit';

export function SectionCard({
  bookletId,
  section,
  index,
}: {
  bookletId: string;
  section: Section;
  index: number;
}) {
  const refresh = useBookletStore((s) => s.refresh);
  const [tab, setTab] = useState<Tab>('preview');
  const [title, setTitle] = useState(section.title);
  const [content, setContent] = useState(section.content);
  const [saving, setSaving] = useState(false);
  const [aiBusy, setAiBusy] = useState<RegenerateMode | null>(null);
  const [instruction, setInstruction] = useState('');
  const [showEdit, setShowEdit] = useState(false);
  const [titleEditing, setTitleEditing] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
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

  async function saveTitle(): Promise<void> {
    const next = title.trim();
    setTitleEditing(false);
    if (next === '' || next === section.title) {
      setTitle(section.title);
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await api.updateSection(bookletId, section.id, { title: next });
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'save failed');
      setTitle(section.title);
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
    <article className="section">
      <header className="section-head">
        <button
          type="button"
          className="icon-btn collapse-btn"
          title={collapsed ? 'Expand section' : 'Collapse section'}
          aria-label={collapsed ? `Expand section ${section.title}` : `Collapse section ${section.title}`}
          aria-expanded={!collapsed}
          onClick={() => setCollapsed((v) => !v)}
        >
          {collapsed ? '▶' : '▼'}
        </button>
        <span className="section-level">
          {index}. {'#'.repeat(Math.min(section.level, 6))}
        </span>
        {titleEditing ? (
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onBlur={() => void saveTitle()}
            onKeyDown={(e) => {
              if (e.key === 'Enter') void saveTitle();
              if (e.key === 'Escape') {
                setTitle(section.title);
                setTitleEditing(false);
              }
            }}
            aria-label="Section title"
            className="section-title-input"
            autoFocus
          />
        ) : (
          <>
            <h2>{section.title}</h2>
            <button
              type="button"
              title="Rename section"
              aria-label={`Rename section ${section.title}`}
              className="icon-btn"
              onClick={() => {
                setTitle(section.title);
                setTitleEditing(true);
              }}
            >
              ✎
            </button>
          </>
        )}
        <span className="badge" style={{ background: statusColor(section.status) }}>
          {section.status}
        </span>
      </header>
      {!collapsed && (
        <>
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
        <textarea
          className="editor"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          rows={Math.max(6, content.split('\n').length + 1)}
        />
      )}
      <footer className="section-foot">
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
        <div className="section-foot">
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
      </>
      )}
    </article>
  );
}
