import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import HealthStatusIndicator from './HealthStatusIndicator';

describe('HealthStatusIndicator', () => {
  it('renders healthy status', () => {
    render(<HealthStatusIndicator status="healthy" />);

    expect(screen.getByText('Healthy')).toBeInTheDocument();
  });

  it('renders degraded status', () => {
    render(<HealthStatusIndicator status="degraded" />);

    expect(screen.getByText('Degraded')).toBeInTheDocument();
  });

  it('renders down status', () => {
    render(<HealthStatusIndicator status="down" />);

    expect(screen.getByText('Down')).toBeInTheDocument();
  });

  it('renders unknown status when status is undefined', () => {
    render(<HealthStatusIndicator />);

    expect(screen.getByText('Unknown')).toBeInTheDocument();
  });

  it('applies healthy badge class', () => {
    render(<HealthStatusIndicator status="healthy" />);

    expect(screen.getByText('Healthy')).toHaveClass('badge-healthy');
  });

  it('applies degraded badge class', () => {
    render(<HealthStatusIndicator status="degraded" />);

    expect(screen.getByText('Degraded')).toHaveClass('badge-degraded');
  });

  it('applies down badge class', () => {
    render(<HealthStatusIndicator status="down" />);

    expect(screen.getByText('Down')).toHaveClass('badge-down');
  });

  it('applies unknown badge class when status is undefined', () => {
    render(<HealthStatusIndicator />);

    expect(screen.getByText('Unknown')).toHaveClass('badge-unknown');
  });

  it('includes response time in tooltip when provided', () => {
    render(<HealthStatusIndicator status="healthy" responseTimeMs={150} />);

    const badge = screen.getByText('Healthy');
    expect(badge.getAttribute('title')).toContain('Response time: 150ms');
  });

  it('includes error message in tooltip when provided', () => {
    render(
      <HealthStatusIndicator status="down" errorMessage="Connection refused" />
    );

    const badge = screen.getByText('Down');
    expect(badge.getAttribute('title')).toContain('Error: Connection refused');
  });

  it('shows "Health check not configured" tooltip when no status', () => {
    render(<HealthStatusIndicator />);

    const badge = screen.getByText('Unknown');
    expect(badge.getAttribute('title')).toBe('Health check not configured');
  });
});
