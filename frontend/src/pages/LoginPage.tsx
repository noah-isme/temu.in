import LoginForm from '../components/LoginForm';
import PageTransition from '../components/PageTransition';
import { useNavigate, useLocation } from 'react-router-dom';

export default function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();

  const from = (location.state as any)?.from?.pathname || '/';

  return (
    <PageTransition>
      <section className="mx-auto max-w-md px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Masuk</h2>
        <LoginForm onLogin={() => navigate(from, { replace: true })} />
      </section>
    </PageTransition>
  );
}
