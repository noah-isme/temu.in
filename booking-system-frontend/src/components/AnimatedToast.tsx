import { motion } from 'framer-motion';

export default function AnimatedToast({ title, message, variant = 'info' }: { title?: string; message?: string; variant?: 'success' | 'info' | 'warn' | 'error' }) {
  const variantClasses: Record<string, string> = {
    success: 'bg-green-50 border-green-200 text-green-800',
    info: 'bg-primary-50 border-primary-200 text-primary',
    warn: 'bg-yellow-50 border-amber-200 text-amber-800',
    error: 'bg-red-50 border-red-200 text-red-800'
  };

  const cls = variantClasses[variant] ?? variantClasses.info;

  return (
    <motion.div
      initial={{ opacity: 0, y: -8, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: -8, scale: 0.98 }}
      transition={{ type: 'spring', stiffness: 300, damping: 24 }}
      className={`toastify-toast border ${cls}`}
    >
      {title && <div className="font-semibold mb-1">{title}</div>}
      {message && <div className="text-sm">{message}</div>}
    </motion.div>
  );
}
