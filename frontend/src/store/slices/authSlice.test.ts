import { describe, it, expect } from 'vitest';
import authReducer, { setCredentials, clearCredentials } from './authSlice';

describe('authSlice', () => {
  const initialState = {
    user: null,
    email: null,
    role: null,
    authProvider: null,
  };

  it('should return the initial state', () => {
    expect(authReducer(undefined, { type: 'unknown' })).toEqual(initialState);
  });

  it('should handle setCredentials', () => {
    const credentials = {
      user: 'testuser',
      email: 'test@example.com',
      role: 'admin',
      authProvider: 'local',
    };

    const actual = authReducer(initialState, setCredentials(credentials));

    expect(actual.user).toBe('testuser');
    expect(actual.email).toBe('test@example.com');
    expect(actual.role).toBe('admin');
    expect(actual.authProvider).toBe('local');
  });

  it('should handle clearCredentials', () => {
    const loggedInState = {
      user: 'testuser',
      email: 'test@example.com',
      role: 'admin',
      authProvider: 'local',
    };

    const actual = authReducer(loggedInState, clearCredentials());

    expect(actual).toEqual(initialState);
  });
});
