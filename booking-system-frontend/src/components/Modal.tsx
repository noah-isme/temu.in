import { ReactNode, useEffect } from 'react';

type Props = {
  title?: string;
  children?: ReactNode;
  open: boolean;
  onClose: () => void;
};

export default function Modal({ title, children, open, onClose }: Props) {
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    if (open) {
      document.addEventListener('keydown', onKey);
      document.body.style.overflow = 'hidden';
    }
    return () => {
      document.removeEventListener('keydown', onKey);
      document.body.style.overflow = '';
    };
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div role="dialog" aria-modal="true" aria-label={title ?? 'dialog'} className="relative z-10 max-w-lg w-full rounded bg-white p-6 shadow-lg">
        {title && <h3 className="mb-4 text-lg font-semibold">{title}</h3>}
        <div>{children}</div>
        <div className="mt-4 flex justify-end">
          <button className="rounded border px-3 py-1" onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}
