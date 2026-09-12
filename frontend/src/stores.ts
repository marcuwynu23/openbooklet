import { create } from 'zustand';
import { api, type Booklet, type BookletSummary } from './api';

interface BookletState {
  booklets: BookletSummary[];
  selectedId: string | null;
  booklet: Booklet | null;
  loading: boolean;
  error: string | null;
  loadBooklets: () => Promise<void>;
  select: (id: string) => Promise<void>;
  create: (title: string, docType: string) => Promise<void>;
  rename: (title: string) => Promise<void>;
  remove: () => Promise<void>;
  refresh: () => Promise<void>;
}

export const useBookletStore = create<BookletState>()((set, get) => ({
  booklets: [],
  selectedId: null,
  booklet: null,
  loading: false,
  error: null,

  async loadBooklets() {
    set({ loading: true, error: null });
    try {
      const booklets = await api.booklets();
      set({ booklets, loading: false });
      const { selectedId } = get();
      if (selectedId === null && booklets.length > 0) {
        const first = booklets[0];
        if (first !== undefined) {
          await get().select(first.id);
        }
      }
    } catch (err) {
      set({ loading: false, error: err instanceof Error ? err.message : 'load failed' });
    }
  },

  async select(id: string) {
    set({ loading: true, error: null, selectedId: id });
    try {
      const booklet = await api.booklet(id);
      set({ booklet, loading: false });
    } catch (err) {
      set({ loading: false, error: err instanceof Error ? err.message : 'load failed' });
    }
  },

  async create(title: string, docType: string) {
    set({ loading: true, error: null });
    try {
      const booklet = await api.createBooklet(title, docType);
      const booklets = await api.booklets();
      set({ booklets, booklet, selectedId: booklet.id, loading: false });
    } catch (err) {
      set({ loading: false, error: err instanceof Error ? err.message : 'create failed' });
    }
  },

  async refresh() {
    const { selectedId } = get();
    await get().loadBooklets();
    if (selectedId !== null) {
      await get().select(selectedId);
    }
  },

  async rename(title: string) {
    const { booklet } = get();
    if (booklet === null) return;
    set({ loading: true, error: null });
    try {
      const updated = await api.renameBooklet(booklet.id, title);
      const booklets = await api.booklets();
      set({ booklet: updated, booklets, loading: false });
    } catch (err) {
      set({ loading: false, error: err instanceof Error ? err.message : 'rename failed' });
    }
  },

  async remove() {
    const { booklet } = get();
    if (booklet === null) return;
    set({ loading: true, error: null });
    try {
      await api.deleteBooklet(booklet.id);
      set({ booklet: null, selectedId: null, loading: false });
      await get().loadBooklets();
    } catch (err) {
      set({ loading: false, error: err instanceof Error ? err.message : 'delete failed' });
    }
  },
}));
