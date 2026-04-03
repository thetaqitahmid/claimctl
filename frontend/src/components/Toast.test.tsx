import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { Toast } from './Toast';

describe('Toast', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders success toast with message', () => {
    render(<Toast type="success" message="Operation successful" onClose={() => {}} />);

    expect(screen.getByText('Operation successful')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toBeInTheDocument();
  });

  it('renders error toast with correct styling', () => {
    render(<Toast type="error" message="Something went wrong" onClose={() => {}} />);

    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('bg-rose-950/90');
  });

  it('renders info toast with correct styling', () => {
    render(<Toast type="info" message="Information" onClose={() => {}} />);

    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('bg-cyan-950/90');
  });

  it('renders warning toast with correct styling', () => {
    render(<Toast type="warning" message="Warning message" onClose={() => {}} />);

    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('bg-amber-950/90');
  });

  it('has close button with aria-label', () => {
    render(<Toast type="success" message="Test" onClose={() => {}} />);

    const closeButton = screen.getByLabelText('Close');
    expect(closeButton).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', async () => {
    vi.useFakeTimers();
    const onClose = vi.fn();
    render(<Toast type="success" message="Test" onClose={onClose} />);

    // Advance past the initial animation timer (10ms)
    await act(async () => {
      vi.advanceTimersByTime(10);
    });

    fireEvent.click(screen.getByLabelText('Close'));

    // Wait for the exit animation timeout (300ms)
    await act(async () => {
      vi.advanceTimersByTime(300);
    });

    expect(onClose).toHaveBeenCalled();
  });
});
