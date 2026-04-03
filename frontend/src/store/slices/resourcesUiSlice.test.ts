import { describe, it, expect } from 'vitest';
import resourcesUiReducer, { setCategory, setSearchQuery } from './resourcesUiSlice';

describe('resourcesUiSlice', () => {
  const initialState = {
    selectedCategory: 'all',
    searchQuery: '',
  };

  it('should return the initial state', () => {
    expect(resourcesUiReducer(undefined, { type: 'unknown' })).toEqual(initialState);
  });

  it('should handle setCategory', () => {
    const actual = resourcesUiReducer(initialState, setCategory('servers'));
    expect(actual.selectedCategory).toBe('servers');
  });

  it('should handle setSearchQuery', () => {
    const actual = resourcesUiReducer(initialState, setSearchQuery('test query'));
    expect(actual.searchQuery).toBe('test query');
  });

  it('should handle multiple state changes', () => {
    let state = resourcesUiReducer(initialState, setCategory('databases'));
    state = resourcesUiReducer(state, setSearchQuery('postgres'));

    expect(state.selectedCategory).toBe('databases');
    expect(state.searchQuery).toBe('postgres');
  });

  it('should handle empty search query', () => {
    const stateWithQuery = { selectedCategory: 'all', searchQuery: 'test' };
    const actual = resourcesUiReducer(stateWithQuery, setSearchQuery(''));
    expect(actual.searchQuery).toBe('');
  });
});
