import { useRef, useState } from 'react';
import { api } from '../api';
import { useBookletStore } from '../stores';

// TransferButtons downloads the booklet as Markdown or imports a Markdown
// file as new sections.
export function TransferButtons({ bookletId }: { bookletId: string }) {
  const refresh = useBookletStore((s) => s.refresh);
  const fileRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function exportFile(): Promise<void> {
    setBusy(true);
    setError(null);
    try {
      const { blob, filename } = await api.exportBooklet(bookletId);
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'export failed');
    } finally {
      setBusy(false);
    }
  }

  async function importFile(file: File): Promise<void> {
    setBusy(true);
    setError(null);
    try {
      await api.importMarkdown(bookletId, file);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'import failed');
    } finally {
      setBusy(false);
      if (fileRef.current !== null) {
        fileRef.current.value = '';
      }
    }
  }

  return (
    <div className="transfer-row">
      <button type="button" disabled={busy} onClick={() => void exportFile()}>
        Export .md
      </button>
      <button type="button" disabled={busy} onClick={() => fileRef.current?.click()}>
        Import .md
      </button>
      <input
        ref={fileRef}
        type="file"
        accept=".md,.markdown,.txt"
        hidden
        aria-label="Import Markdown file"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file !== undefined) void importFile(file);
        }}
      />
      {error !== null && <span className="error">{error}</span>}
    </div>
  );
}
