import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ConfirmDeleteDialog } from './ConfirmDeleteDialog';

describe('ConfirmDeleteDialog', () => {
  const mockObject = { name: 'Test Resource' };
  const defaultProps = {
    object: mockObject,
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders confirmation message with object name', () => {
    render(<ConfirmDeleteDialog {...defaultProps} />);

    expect(screen.getByText(/Are you sure you want to delete Test Resource\?/)).toBeInTheDocument();
  });

  it('renders Delete button', () => {
    render(<ConfirmDeleteDialog {...defaultProps} />);

    expect(screen.getByText('Delete')).toBeInTheDocument();
  });

  it('renders Cancel button', () => {
    render(<ConfirmDeleteDialog {...defaultProps} />);

    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it('calls onConfirm when Delete button is clicked', () => {
    render(<ConfirmDeleteDialog {...defaultProps} />);

    fireEvent.click(screen.getByText('Delete'));

    expect(defaultProps.onConfirm).toHaveBeenCalledTimes(1);
  });

  it('calls onCancel when Cancel button is clicked', () => {
    render(<ConfirmDeleteDialog {...defaultProps} />);

    fireEvent.click(screen.getByText('Cancel'));

    expect(defaultProps.onCancel).toHaveBeenCalledTimes(1);
  });

  it('works with different object names', () => {
    const customObject = { name: 'Custom Item' };
    render(
      <ConfirmDeleteDialog
        object={customObject}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(screen.getByText(/Are you sure you want to delete Custom Item\?/)).toBeInTheDocument();
  });
});
