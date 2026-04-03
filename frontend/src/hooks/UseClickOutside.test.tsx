import { describe, it, expect, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useRef } from 'react';
import { useClickOutside } from './UseClickOutside';

describe('useClickOutside', () => {
  it('does not add event listener when isOpen is false', () => {
    const addEventListenerSpy = vi.spyOn(document, 'addEventListener');

    expect(addEventListenerSpy).not.toHaveBeenCalledWith('mousedown', expect.any(Function));

    addEventListenerSpy.mockRestore();
  });

  it('adds event listener when isOpen is true', () => {
    const addEventListenerSpy = vi.spyOn(document, 'addEventListener');

    renderHook(() => {
      const ref = useRef<HTMLDivElement>(null);
      useClickOutside({ ref, isOpen: true, onClose: vi.fn() });
      return ref;
    });

    expect(addEventListenerSpy).toHaveBeenCalledWith('mousedown', expect.any(Function));

    addEventListenerSpy.mockRestore();
  });

  it('removes event listener on cleanup', () => {
    const removeEventListenerSpy = vi.spyOn(document, 'removeEventListener');

    const { unmount } = renderHook(() => {
      const ref = useRef<HTMLDivElement>(null);
      useClickOutside({ ref, isOpen: true, onClose: vi.fn() });
      return ref;
    });

    unmount();

    expect(removeEventListenerSpy).toHaveBeenCalledWith('mousedown', expect.any(Function));

    removeEventListenerSpy.mockRestore();
  });

  it('calls onClose when clicking outside the element', () => {
    const onClose = vi.fn();

    // Create a real DOM element
    const container = document.createElement('div');
    document.body.appendChild(container);

    renderHook(() => {
      const ref = useRef<HTMLDivElement>(container);
      useClickOutside({ ref, isOpen: true, onClose });
      return ref;
    });

    // Click outside the element
    const outsideElement = document.createElement('div');
    document.body.appendChild(outsideElement);

    act(() => {
      outsideElement.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });

    expect(onClose).toHaveBeenCalled();

    // Cleanup
    document.body.removeChild(container);
    document.body.removeChild(outsideElement);
  });

  it('does not call onClose when clicking inside the element', () => {
    const onClose = vi.fn();

    // Create a real DOM element
    const container = document.createElement('div');
    const innerElement = document.createElement('button');
    container.appendChild(innerElement);
    document.body.appendChild(container);

    renderHook(() => {
      const ref = useRef<HTMLDivElement>(container);
      useClickOutside({ ref, isOpen: true, onClose });
      return ref;
    });

    // Click inside the element
    act(() => {
      innerElement.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });

    expect(onClose).not.toHaveBeenCalled();

    // Cleanup
    document.body.removeChild(container);
  });
});
