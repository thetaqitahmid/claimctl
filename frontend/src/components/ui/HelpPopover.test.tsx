import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import HelpPopover from './HelpPopover';

describe('HelpPopover', () => {
  it('renders the label text', () => {
    render(<HelpPopover label="Help">Content</HelpPopover>);

    expect(screen.getByText(/Help/)).toBeInTheDocument();
  });

  it('does not show children by default', () => {
    render(<HelpPopover label="Help">Hidden Content</HelpPopover>);

    expect(screen.queryByText('Hidden Content')).not.toBeInTheDocument();
  });

  it('shows collapse indicator when closed', () => {
    render(<HelpPopover label="Help">Content</HelpPopover>);

    expect(screen.getByText(/▶/)).toBeInTheDocument();
  });

  it('shows children when button is clicked', () => {
    render(<HelpPopover label="Help">Visible Content</HelpPopover>);

    fireEvent.click(screen.getByRole('button'));

    expect(screen.getByText('Visible Content')).toBeInTheDocument();
  });

  it('shows expand indicator when open', () => {
    render(<HelpPopover label="Help">Content</HelpPopover>);

    fireEvent.click(screen.getByRole('button'));

    expect(screen.getByText(/▼/)).toBeInTheDocument();
  });

  it('hides children when button is clicked again', () => {
    render(<HelpPopover label="Help">Toggle Content</HelpPopover>);

    const button = screen.getByRole('button');

    // Open
    fireEvent.click(button);
    expect(screen.getByText('Toggle Content')).toBeInTheDocument();

    // Close
    fireEvent.click(button);
    expect(screen.queryByText('Toggle Content')).not.toBeInTheDocument();
  });

  it('renders complex children content', () => {
    render(
      <HelpPopover label="Info">
        <p>Paragraph one</p>
        <p>Paragraph two</p>
      </HelpPopover>
    );

    fireEvent.click(screen.getByRole('button'));

    expect(screen.getByText('Paragraph one')).toBeInTheDocument();
    expect(screen.getByText('Paragraph two')).toBeInTheDocument();
  });
});
