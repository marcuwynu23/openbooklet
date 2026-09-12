import { marked } from 'marked';

marked.setOptions({ breaks: true });

// renderMarkdown renders user Markdown to HTML for preview display.
export function renderMarkdown(content: string): string {
  const html = marked.parse(content, { async: false });
  return typeof html === 'string' ? html : '';
}

let mermaidId = 0;
let mermaidReady: Promise<typeof import('mermaid').default> | null = null;

async function loadMermaid(): Promise<typeof import('mermaid').default> {
  if (mermaidReady === null) {
    mermaidReady = import('mermaid').then((module) => {
      module.default.initialize({ startOnLoad: false });
      return module.default;
    });
  }
  return mermaidReady;
}

// enhanceMermaid finds fenced mermaid blocks inside rendered preview HTML and
// replaces them with rendered SVG diagrams. It lazy-loads the mermaid library
// so pages without diagrams never download it. Blocks with invalid syntax keep
// their source visible with an error note instead of breaking the preview.
export async function enhanceMermaid(container: HTMLElement): Promise<void> {
  const blocks = container.querySelectorAll('pre > code.language-mermaid');
  if (blocks.length === 0) {
    return;
  }
  const mermaid = await loadMermaid();
  for (const block of Array.from(blocks)) {
    const pre = block.parentElement;
    const source = block.textContent ?? '';
    if (pre === null || source.trim() === '') {
      continue;
    }
    mermaidId += 1;
    try {
      const { svg } = await mermaid.render(`mermaid-diagram-${mermaidId}`, source);
      const figure = document.createElement('div');
      figure.className = 'mermaid-diagram';
      figure.innerHTML = svg;
      pre.replaceWith(figure);
    } catch {
      pre.classList.add('mermaid-error');
      const note = document.createElement('p');
      note.className = 'error';
      note.textContent = 'Could not render this diagram; showing source instead.';
      pre.after(note);
    }
  }
}
