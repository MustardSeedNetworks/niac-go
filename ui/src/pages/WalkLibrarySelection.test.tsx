import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { LibraryFileEntry } from '../api/library-client';
import { WalkAnalyzerPage } from './WalkAnalyzerPage';
import { WalkValidatorPage } from './WalkValidatorPage';

const fetchLibraryWalks = vi.fn<() => Promise<LibraryFileEntry[]>>();
vi.mock('../api/library-client', () => ({ fetchLibraryWalks: () => fetchLibraryWalks() }));

const entries: LibraryFileEntry[] = ['first.walk', 'second.walk'].map((name) => ({
  name,
  sizeBytes: 100,
  modifiedAt: '',
  source: 'user',
  edited: false,
}));

describe.each([WalkAnalyzerPage, WalkValidatorPage])('walk library selection', (Page) => {
  beforeEach(() => fetchLibraryWalks.mockReset().mockResolvedValue(entries));

  it('loads once when adopting the initial selection and changing files', async () => {
    render(<Page />);
    const picker = screen.getByRole('combobox');
    await waitFor(() => expect(picker).toHaveValue('first.walk'));
    expect(fetchLibraryWalks).toHaveBeenCalledTimes(1);
    await userEvent.selectOptions(picker, 'second.walk');
    expect(picker).toHaveValue('second.walk');
    expect(fetchLibraryWalks).toHaveBeenCalledTimes(1);
  });

  it('settles an empty library without refetching', async () => {
    fetchLibraryWalks.mockResolvedValue([]);
    render(<Page />);
    await screen.findByRole('option', { name: 'No walks found' });
    expect(fetchLibraryWalks).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('combobox')).toHaveTextContent('No walks found');
  });
});
