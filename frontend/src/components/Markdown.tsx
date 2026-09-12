import { useEffect, useRef } from 'react';
import { enhanceMermaid, renderMarkdown } from '../markdown';

// Markdown renders Markdown to HTML and upgrades fenced mermaid blocks to
// diagrams. Async renders that finish after a content change touch only
// detached nodes, so they cannot corrupt the current view.
export function Markdown({ content, emptyText }: { content: string; emptyText?: string }) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (ref.current !== null) {
      void enhanceMermaid(ref.current);
    }
  }, [content]);

  if (content === '') {
    return (
      <div className="markdown">
        <i>{emptyText ?? '(empty)'}</i>
      </div>
    );
  }
  return (
    <div
      ref={ref}
      className="markdown"
      dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
    />
  );
}
