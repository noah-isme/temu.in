import { toast as rt } from 'react-toastify';
import AnimatedToast from '../components/AnimatedToast';
import { t } from './i18n';

function show(variant: 'success' | 'info' | 'warn' | 'error', messageOrKey: string, title?: string) {
  const message = messageOrKey.startsWith('msg:') ? t(messageOrKey.replace(/^msg:/, '')) : messageOrKey;
  rt(<AnimatedToast variant={variant} title={title} message={message} />);
}

export function success(messageOrKey: string, title?: string) {
  show('success', messageOrKey, title ?? t('booking_success'));
}

export function info(messageOrKey: string, title?: string) {
  show('info', messageOrKey, title);
}

export function warn(messageOrKey: string, title?: string) {
  show('warn', messageOrKey, title);
}

export function error(messageOrKey: string, title?: string) {
  show('error', messageOrKey, title ?? t('payment_failed'));
}

export default { success, info, warn, error };

export { setLocale } from './i18n';
