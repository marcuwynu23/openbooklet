import { useState } from 'react';
import { api } from '../api';
import { useBookletStore } from '../stores';

// GeneratePanel is the creation interface: describe the document in chat
// style, watch the Markdown stream in, then work with the resulting sections.
export function GeneratePanel({ bookletId }: { bookletId: string }) {
  const refresh = useBookletStore((s) => s.refresh);
  const [prompt, setPrompt] = useState('');
  const [streaming, setStreaming] = useState(false);
  const [streamed, setStreamed] = useState('');
  const [error, setError] = useState<string | null>(null);

  async function generate(): Promise<void> {
    const text = prompt.trim();
    if (text === '' || streaming) return;
    setStreaming(true);
    setStreamed('');
    setError(null);
    try {
      await api.generate(bookletId, text, (token) => {
        setStreamed((prev) => prev + token);
      });
      setPrompt('');
      setStreamed('');
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'generate failed');
    } finally {
      setStreaming(false);
    }
  }

  return (
    <section className="generate">
      <h2>Describe the document</h2>
      <textarea
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        placeholder="e.g. Write an SOP for deploying our web service to Kubernetes…"
        rows={3}
        disabled={streaming}
      />
      <div className="section-foot">
        <button type="button" disabled={prompt.trim() === '' || streaming} onClick={() => void generate()}>
          {streaming ? 'Generating…' : 'Generate sections'}
        </button>
        {error !== null && <span className="error">{error}</span>}
      </div>
      {streamed !== '' && <pre className="preview streaming">{streamed}</pre>}
    </section>
  );
}
