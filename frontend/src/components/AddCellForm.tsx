import { useState } from 'react';
import { api, type Section } from '../api';
import { useBookletStore } from '../stores';

// AddCellForm appends a manually written cell. The level defaults to the
// previous cell's level; nesting picks the nearest preceding lower-level
// cell as parent.
export function AddCellForm({ bookletId, sections }: { bookletId: string; sections: Section[] }) {
  const refresh = useBookletStore((s) => s.refresh);
  const lastLevel = sections.length > 0 ? (sections[sections.length - 1]?.level ?? 2) : 2;
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState('');
  const [level, setLevel] = useState(lastLevel);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function parentFor(newLevel: number): string | null {
    for (let i = sections.length - 1; i >= 0; i--) {
      const s = sections[i];
      if (s !== undefined && s.level < newLevel) {
        return s.id;
      }
    }
    return null;
  }

  async function submit(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    if (title.trim() === '' || busy) return;
    setBusy(true);
    setError(null);
    try {
      await api.createSection(bookletId, {
        title: title.trim(),
        level,
        parentId: parentFor(level),
        content: '',
      });
      setTitle('');
      setLevel(lastLevel);
      setOpen(false);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'add failed');
    } finally {
      setBusy(false);
    }
  }

  if (!open) {
    return (
      <button type="button" className="add-cell" onClick={() => setOpen(true)}>
        ＋ Add cell
      </button>
    );
  }

  return (
    <form className="cell add-form" onSubmit={(e) => void submit(e)}>
      <div className="cell-foot">
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Cell title…"
          aria-label="New cell title"
          className="ai-input"
          autoFocus
        />
        <label>
          Level{' '}
          <select value={level} onChange={(e) => setLevel(Number(e.target.value))} aria-label="Heading level">
            {[1, 2, 3, 4, 5, 6].map((n) => (
              <option key={n} value={n}>
                {'#'.repeat(n)}
              </option>
            ))}
          </select>
        </label>
        <button type="submit" disabled={title.trim() === '' || busy}>
          {busy ? 'Adding…' : 'Add'}
        </button>
        <button type="button" onClick={() => setOpen(false)}>
          Cancel
        </button>
      </div>
      {error !== null && <span className="error">{error}</span>}
    </form>
  );
}
