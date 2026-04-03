import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import ConfirmationModal from './ConfirmationModal';

describe('ConfirmationModal', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    onConfirm: vi.fn(),
    title: 'Confirm Action',
    message: 'Are you sure you want to proceed?',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders nothing when isOpen is false', () => {
    const { container } = render(
      <ConfirmationModal {...defaultProps} isOpen={false} />
    );

    expect(container.firstChild).toBeNull();
  });

  it('renders modal when isOpen is true', () => {
    render(<ConfirmationModal {...defaultProps} />);

    expect(screen.getByText('Confirm Action')).toBeInTheDocument();
    expect(screen.getByText('Are you sure you want to proceed?')).toBeInTheDocument();
  });

  it('displays default button text', () => {
    render(<ConfirmationModal {...defaultProps} />);

    expect(screen.getByText('Confirm')).toBeInTheDocument();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it('displays custom button text', () => {
    render(
      <ConfirmationModal
        {...defaultProps}
        confirmText="Delete"
        cancelText="Keep"
      />
    );

    expect(screen.getByText('Delete')).toBeInTheDocument();
    expect(screen.getByText('Keep')).toBeInTheDocument();
  });

  it('calls onClose when cancel button is clicked', () => {
    render(<ConfirmationModal {...defaultProps} />);

    fireEvent.click(screen.getByText('Cancel'));

    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
    expect(defaultProps.onConfirm).not.toHaveBeenCalled();
  });

  it('calls onClose when X button is clicked', () => {
    const { container } = render(<ConfirmationModal {...defaultProps} />);

    // Find the X button (it's the first button in the modal header)
    const closeButton = container.querySelector('button');
    fireEvent.click(closeButton!);

    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  it('calls onConfirm and onClose when confirm button is clicked', () => {
    render(<ConfirmationModal {...defaultProps} />);

    fireEvent.click(screen.getByText('Confirm'));

    expect(defaultProps.onConfirm).toHaveBeenCalledTimes(1);
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it('applies destructive styling when isDestructive is true', () => {
    render(<ConfirmationModal {...defaultProps} isDestructive={true} />);

    const confirmButton = screen.getByText('Confirm');
    expect(confirmButton).toHaveClass('bg-red-500');
  });

  it('applies normal styling when isDestructive is false', () => {
    render(<ConfirmationModal {...defaultProps} isDestructive={false} />);

    const confirmButton = screen.getByText('Confirm');
    expect(confirmButton).toHaveClass('bg-cyan-600');
  });

  it('renders ReactNode message content', () => {
    render(
      <ConfirmationModal
        {...defaultProps}
        message={<span data-testid="custom-message">Custom content</span>}
      />
    );

    expect(screen.getByTestId('custom-message')).toBeInTheDocument();
  });
});
