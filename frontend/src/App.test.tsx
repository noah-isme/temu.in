
import { render, screen } from '@testing-library/react';
import App from './App';

describe('App', () => {
  it('renders the hero headline', () => {
    render(<App />);
    expect(screen.getByText(/Solusi Smart Booking/i)).toBeInTheDocument();
  });
});
