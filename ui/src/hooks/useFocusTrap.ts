/**
 * @fileoverview Focus Trap Hook for Accessibility
 * @description Custom hook that traps keyboard focus within a modal/drawer.
 *              Implements WCAG 2.1 AA compliance for modal dialogs.
 */

import { type RefObject, useEffect, useRef } from 'react';

/** Focusable element selectors */
const FOCUSABLE_SELECTORS: string = [
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  'a[href]',
  '[tabindex]:not([tabindex="-1"])',
].join(', ');

interface UseFocusTrapOptions {
  isActive: boolean;
  onEscape?: () => void;
  autoFocus?: boolean;
  restoreFocus?: boolean;
}

function getFocusableElements(container: HTMLElement): HTMLElement[] {
  const elements = container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTORS);
  return Array.from(elements).filter(
    (el) => el.offsetParent !== null && !el.hasAttribute('aria-hidden'),
  );
}

function handleTabKey(
  event: KeyboardEvent,
  container: HTMLElement,
  focusableElements: HTMLElement[],
): void {
  if (focusableElements.length === 0) {
    return;
  }

  const [firstElement] = focusableElements;
  const lastElement = focusableElements.at(-1);
  if (!(lastElement && firstElement)) {
    return;
  }
  const isAtFirst = document.activeElement === firstElement;
  const isAtLast = document.activeElement === lastElement;
  const isOutside = !container.contains(document.activeElement);

  if (event.shiftKey && isAtFirst) {
    event.preventDefault();
    lastElement.focus();
    return;
  }

  if (!event.shiftKey && isAtLast) {
    event.preventDefault();
    firstElement.focus();
    return;
  }

  if (isOutside) {
    event.preventDefault();
    firstElement.focus();
  }
}

/**
 * Custom hook that traps keyboard focus within a container.
 * Implements modal dialog accessibility requirements per WCAG 2.1 AA.
 */
export function useFocusTrap<T extends HTMLElement = HTMLDivElement>(
  options: UseFocusTrapOptions,
): RefObject<T | null> {
  const { isActive, onEscape, autoFocus = true, restoreFocus = true } = options;
  const containerRef = useRef<T>(null);
  const previousActiveElement = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!isActive) {
      return;
    }

    if (restoreFocus) {
      previousActiveElement.current = document.activeElement as HTMLElement;
    }

    const container = containerRef.current;
    if (!container) {
      return;
    }

    if (autoFocus) {
      const focusableElements = getFocusableElements(container);
      const [firstElement] = focusableElements;
      if (firstElement) {
        requestAnimationFrame(() => {
          firstElement.focus();
        });
      }
    }

    const handleKeyDown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape' && onEscape) {
        event.preventDefault();
        event.stopPropagation();
        onEscape();
        return;
      }

      if (event.key === 'Tab') {
        const focusableElements = getFocusableElements(container);
        handleTabKey(event, container, focusableElements);
      }
    };

    const handleFocusOut = (event: FocusEvent): void => {
      const isLeavingContainer = !container.contains(event.relatedTarget as Node);
      if (!isLeavingContainer) {
        return;
      }

      const focusableElements = getFocusableElements(container);
      if (focusableElements.length === 0) {
        return;
      }

      requestAnimationFrame(() => {
        if (!container.contains(document.activeElement)) {
          focusableElements[0]?.focus();
        }
      });
    };

    document.addEventListener('keydown', handleKeyDown);
    container.addEventListener('focusout', handleFocusOut);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      container.removeEventListener('focusout', handleFocusOut);

      if (restoreFocus && previousActiveElement.current) {
        previousActiveElement.current.focus();
      }
    };
  }, [isActive, onEscape, autoFocus, restoreFocus]);

  return containerRef;
}

export default useFocusTrap;
