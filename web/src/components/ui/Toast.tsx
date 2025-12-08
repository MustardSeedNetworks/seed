import {
  useState,
  useEffect,
  useCallback,
  createContext,
  useContext,
  ReactNode,
} from "react";
import { ToastType, typeStyles, icons } from "./Toast.constants.tsx";

interface Toast {
  id: string;
  message: string;
  type: ToastType;
  duration?: number;
}

interface ToastContextType {
  addToast: (message: string, type?: ToastType, duration?: number) => void;
  removeToast: (id: string) => void;
}

const ToastContext = createContext<ToastContextType | null>(null);

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within a ToastProvider");
  }
  return context;
}

interface ToastProviderProps {
  children: ReactNode;
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const addToast = useCallback(
    (message: string, type: ToastType = "info", duration = 5000) => {
      const id = `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      setToasts((prev) => [...prev, { id, message, type, duration }]);
    },
    [],
  );

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ addToast, removeToast }}>
      {children}
      <ToastContainer toasts={toasts} removeToast={removeToast} />
    </ToastContext.Provider>
  );
}

interface ToastContainerProps {
  toasts: Toast[];
  removeToast: (id: string) => void;
}

function ToastContainer({ toasts, removeToast }: ToastContainerProps) {
  return (
    <div
      aria-live="polite"
      aria-label="Notifications"
      className="fixed bottom-20 right-4 z-50 flex flex-col gap-2 max-w-sm"
    >
      {toasts.map((toast) => (
        <ToastItem
          key={toast.id}
          toast={toast}
          onClose={() => removeToast(toast.id)}
        />
      ))}
    </div>
  );
}

interface ToastItemProps {
  toast: Toast;
  onClose: () => void;
}

function ToastItem({ toast, onClose }: ToastItemProps) {
  useEffect(() => {
    if (toast.duration && toast.duration > 0) {
      const timer = setTimeout(onClose, toast.duration);
      return () => clearTimeout(timer);
    }
  }, [toast.duration, onClose]);

  return (
    <div
      role="alert"
      className={`flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg ${typeStyles[toast.type]} animate-slide-in`}
      aria-label={`Notification: ${toast.type}`}
    >
      {icons[toast.type]}
      <p className="text-sm font-medium flex-1">{toast.message}</p>
      <button
        onClick={onClose}
        className="p-1 rounded hover:bg-white/20 focus:outline-none focus:ring-2 focus:ring-white/50"
        aria-label="Dismiss notification"
      >
        <svg
          className="w-4 h-4"
          fill="currentColor"
          viewBox="0 0 20 20"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
      </button>
    </div>
  );
}
