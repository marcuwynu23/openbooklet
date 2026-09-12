import { marked } from 'marked';

marked.setOptions({ breaks: true });

// renderMarkdown renders user Markdown to HTML for preview display.
export function renderMarkdown(content: string): string {
  const html = marked.parse(content, { async: false });
  return typeof html === 'string' ? html : '';
}
