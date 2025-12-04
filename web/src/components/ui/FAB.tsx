import { useState, useCallback } from 'react';

interface FABProps {
  className?: string;
}

export function FAB({ className = '' }: FABProps) {
  const [isRunning, setIsRunning] = useState(false);

  const handleClick = useCallback(() => {
    if (isRunning) return;

    setIsRunning(true);

    // Dispatch event to trigger all tests
    window.dispatchEvent(new CustomEvent('runAllTests'));

    // Reset after a reasonable time for tests to complete
    setTimeout(() => {
      setIsRunning(false);
    }, 30000);
  }, [isRunning]);

  return (
    <button
      onClick={handleClick}
      disabled={isRunning}
      className={`fixed bottom-6 right-6 w-14 h-14 rounded-full bg-brand-primary text-text-inverse shadow-lg hover:bg-brand-accent active:scale-95 transition-all flex items-center justify-center touch-manipulation z-50 ${
        isRunning ? 'opacity-75 cursor-not-allowed' : ''
      } ${className}`}
      title="Run All Tests"
      aria-label="Run All Tests"
    >
      {isRunning ? (
        <svg
          className="w-6 h-6 animate-spin"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>
      ) : (
        <svg
          className="w-6 h-6"
          fill="currentColor"
          viewBox="0 0 24 24"
        >
          <path d="M8 5v14l11-7z" />
        </svg>
      )}
    </button>
  );
}
