import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import UserDropDownMenu from './UserDropDown';
import { Settings, LogOut } from 'lucide-react';

describe('UserDropDownMenu', () => {
  const mockItems = [
    { name: 'Settings', icon: Settings, action: vi.fn() },
    { name: 'Log out', icon: LogOut, action: vi.fn() },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('is hidden when isOpen is false', () => {
    const { container } = render(
      <UserDropDownMenu
        isOpen={false}
        onClose={() => {}}
        dropDownPropArray={mockItems}
      />
    );

    const dropdown = container.firstChild as HTMLElement;
    expect(dropdown).toHaveClass('hidden');
  });

  it('is visible when isOpen is true', () => {
    const { container } = render(
      <UserDropDownMenu
        isOpen={true}
        onClose={() => {}}
        dropDownPropArray={mockItems}
      />
    );

    const dropdown = container.firstChild as HTMLElement;
    expect(dropdown).toHaveClass('block');
  });

  it('renders all menu items', () => {
    render(
      <UserDropDownMenu
        isOpen={true}
        onClose={() => {}}
        dropDownPropArray={mockItems}
      />
    );

    expect(screen.getByText('Settings')).toBeInTheDocument();
    expect(screen.getByText('Log out')).toBeInTheDocument();
  });

  it('calls item action when clicked', () => {
    render(
      <UserDropDownMenu
        isOpen={true}
        onClose={() => {}}
        dropDownPropArray={mockItems}
      />
    );

    fireEvent.click(screen.getByText('Settings'));
    expect(mockItems[0].action).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByText('Log out'));
    expect(mockItems[1].action).toHaveBeenCalledTimes(1);
  });

  it('renders items without icons', () => {
    const itemsWithoutIcons = [
      { name: 'Profile', action: vi.fn() },
      { name: 'Help', action: vi.fn() },
    ];

    render(
      <UserDropDownMenu
        isOpen={true}
        onClose={() => {}}
        dropDownPropArray={itemsWithoutIcons}
      />
    );

    expect(screen.getByText('Profile')).toBeInTheDocument();
    expect(screen.getByText('Help')).toBeInTheDocument();
  });

  it('renders buttons with correct type', () => {
    render(
      <UserDropDownMenu
        isOpen={true}
        onClose={() => {}}
        dropDownPropArray={mockItems}
      />
    );

    const buttons = screen.getAllByRole('button');
    buttons.forEach(button => {
      expect(button).toHaveAttribute('type', 'button');
    });
  });
});
