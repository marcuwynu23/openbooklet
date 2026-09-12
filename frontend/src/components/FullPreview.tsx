import { useMemo } from 'react';
import type { Booklet } from '../api';
import { renderMarkdown } from '../markdown';

// FullPreview renders the whole booklet as one composed Markdown document:
// the booklet title as H1 followed by every section in document order.
export function FullPreview({ booklet }: { booklet: Booklet }) {
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

  return (
    <article className="section full-preview">
      <div className="markdown" dangerouslySetInnerHTML={{ __html: html }} />
    </article>
  );
}
