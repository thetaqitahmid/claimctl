import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import EmptyState from './EmptyState';

describe('EmptyState', () => {
  it('renders message text', () => {
    render(<EmptyState message="No items found" />);

    expect(screen.getByText('No items found')).toBeInTheDocument();
  });

  it('renders optional icon when provided', () => {
    const testIcon = <span data-testid="test-icon">Icon</span>;

    render(<EmptyState message="No items" icon={testIcon} />);

    expect(screen.getByTestId('test-icon')).toBeInTheDocument();
  });

  it('does not render icon container when icon is not provided', () => {
    const { container } = render(<EmptyState message="No items" />);

    // The icon container div should not exist
    const iconContainer = container.querySelector('.mb-2.text-slate-500');
    expect(iconContainer).toBeNull();
  });
});
