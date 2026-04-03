import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import Tabs from './Tabs';

describe('Tabs', () => {
  const mockTabs = [
    { id: 'tab1', label: 'Tab One' },
    { id: 'tab2', label: 'Tab Two' },
    { id: 'tab3', label: 'Tab Three' },
  ];

  it('renders all tabs', () => {
    render(<Tabs tabs={mockTabs} activeTab="tab1" onTabChange={() => {}} />);

    expect(screen.getByText('Tab One')).toBeInTheDocument();
    expect(screen.getByText('Tab Two')).toBeInTheDocument();
    expect(screen.getByText('Tab Three')).toBeInTheDocument();
  });

  it('highlights the active tab with correct styling', () => {
    render(<Tabs tabs={mockTabs} activeTab="tab2" onTabChange={() => {}} />);

    const activeTab = screen.getByText('Tab Two');
    expect(activeTab).toHaveClass('text-cyan-400');
  });

  it('non-active tabs have different styling', () => {
    render(<Tabs tabs={mockTabs} activeTab="tab1" onTabChange={() => {}} />);

    const inactiveTab = screen.getByText('Tab Two');
    expect(inactiveTab).toHaveClass('text-slate-400');
  });

  it('calls onTabChange with correct id when tab is clicked', () => {
    const onTabChange = vi.fn();
    render(<Tabs tabs={mockTabs} activeTab="tab1" onTabChange={onTabChange} />);

    fireEvent.click(screen.getByText('Tab Two'));

    expect(onTabChange).toHaveBeenCalledWith('tab2');
  });

  it('renders active indicator for active tab only', () => {
    const { container } = render(
      <Tabs tabs={mockTabs} activeTab="tab1" onTabChange={() => {}} />
    );

    // Should only have one indicator (the span with bg-cyan-400)
    const indicators = container.querySelectorAll('.bg-cyan-400');
    expect(indicators).toHaveLength(1);
  });

  it('applies custom className when provided', () => {
    const { container } = render(
      <Tabs
        tabs={mockTabs}
        activeTab="tab1"
        onTabChange={() => {}}
        className="custom-class"
      />
    );

    expect(container.firstChild).toHaveClass('custom-class');
  });
});
