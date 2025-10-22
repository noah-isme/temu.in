import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import BookingFlow from '../BookingFlow';
import { afterEach, beforeEach, vi } from 'vitest';

describe('BookingFlow', () => {
  beforeEach(() => {
    // stub alert
    vi.spyOn(window, 'alert').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it('allows selecting provider, date, slot and completes booking + payment', async () => {
    render(<BookingFlow />);

    // wait for service and provider options to appear
    await waitFor(() => expect(screen.getByText(/-- pilih service --/i)).toBeInTheDocument());

    // choose first service
    const serviceSelect = screen.getByLabelText(/Service/i) as HTMLSelectElement;
    fireEvent.change(serviceSelect, { target: { value: serviceSelect.querySelector('option:nth-child(2)')?.getAttribute('value') } });

    // choose first provider
    const providerSelect = screen.getByLabelText(/Provider/i) as HTMLSelectElement;
    await waitFor(() => expect(providerSelect.options.length).toBeGreaterThan(1));
    fireEvent.change(providerSelect, { target: { value: providerSelect.querySelector('option:nth-child(2)')?.getAttribute('value') } });

    // choose date
    const dateInput = screen.getByLabelText(/Date/i) as HTMLInputElement;
    fireEvent.change(dateInput, { target: { value: new Date().toISOString().slice(0, 10) } });

    // wait for slots to load
    await waitFor(() => expect(screen.queryAllByRole('radio').length).toBeGreaterThan(0));

    // pick first slot
    const firstRadio = screen.getAllByRole('radio')[0];
    fireEvent.click(firstRadio);

    // confirm booking
    const btn = screen.getByRole('button', { name: /confirm booking/i });
    fireEvent.click(btn);

    await waitFor(() => expect(window.alert).toHaveBeenCalled());
  });

  it('shows error when payment is forced to fail', async () => {
    render(<BookingFlow />);

    await waitFor(() => expect(screen.getByText(/-- pilih service --/i)).toBeInTheDocument());

    const providerSelect = screen.getByLabelText(/Provider/i) as HTMLSelectElement;
    await waitFor(() => expect(providerSelect.options.length).toBeGreaterThan(1));
    fireEvent.change(providerSelect, { target: { value: providerSelect.querySelector('option:nth-child(2)')?.getAttribute('value') } });

    const dateInput = screen.getByLabelText(/Date/i) as HTMLInputElement;
    fireEvent.change(dateInput, { target: { value: new Date().toISOString().slice(0, 10) } });

    await waitFor(() => expect(screen.queryAllByRole('radio').length).toBeGreaterThan(0));
    fireEvent.click(screen.getAllByRole('radio')[0]);

    // enable force fail
    const checkbox = screen.getByLabelText(/Force payment fail/i) as HTMLInputElement;
    fireEvent.click(checkbox);

    const btn = screen.getByRole('button', { name: /confirm booking/i });
    fireEvent.click(btn);

    await waitFor(() => expect(screen.getByText(/Payment failed/i) || window.alert));
  });
});
