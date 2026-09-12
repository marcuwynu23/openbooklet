import { useEffect, useState } from 'react';
import { api, type RegenerateMode, type Section } from '../api';
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

export function CellCard({ bookletId, section }: { bookletId: string; section: Section }) {
  const refresh = useBookletStore((s) => s.refresh);
  const [editing, setEditing] = useState(false);
  const [content, setContent] = useState(section.content);
  const [saving, setSaving] = useState(false);
  const [aiBusy, setAiBusy] = useState<RegenerateMode | null>(null);
  const [instruction, setInstruction] = useState('');
  const [showEdit, setShowEdit] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setContent(section.content);
    setEditing(false);
  }, [section.content, section.id]);

  const dirty = content !== section.content;

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

  async function save(): Promise<void> {
    setSaving(true);
    setError(null);
    try {
      await api.updateSection(bookletId, section.id, { content });
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'save failed');
    } finally {
      setSaving(false);
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
      {editing ? (
        <textarea
          className="editor"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          rows={Math.max(6, content.split('\n').length + 1)}
        />
      ) : (
        <pre className="preview">{section.content === '' ? <i>(empty)</i> : section.content}</pre>
      )}
      <footer className="cell-foot">
        <button type="button" onClick={() => setEditing((v) => !v)}>
          {editing ? 'Preview' : 'Edit'}
        </button>
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
