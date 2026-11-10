import React, { useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import './style.css';

function App() {
  const [url, setURL] = useState('');
  const [shortURL, setShortURL] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [copyMessage, setCopyMessage] = useState('');
  const submitting = useRef(false);
  const resultInput = useRef(null);

  async function shorten(event) {
    event.preventDefault();
    if (submitting.current) return;
    setError('');
    setShortURL('');
    setCopyMessage('');
    const value = url.trim();
    try {
      const parsed = new URL(value);
      if (!['http:', 'https:'].includes(parsed.protocol) || parsed.username || parsed.password || new TextEncoder().encode(value).length > 8192) throw new Error();
    } catch {
      setError('Введите полную ссылку с http:// или https:// без логина и пароля, длиной до 8192 байт.');
      return;
    }

    submitting.current = true;
    setBusy(true);
    try {
      const response = await fetch('/api/links', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: value }),
        signal: AbortSignal.timeout(15000),
      });
      if (!response.ok) {
        throw new Error(response.status === 400
          ? 'Не удалось сократить этот URL. Проверьте адрес ссылки.'
          : 'Сервис временно недоступен. Попробуйте ещё раз.');
      }
      const data = await response.json();
      const result = new URL(data.short_url);
      if (!['http:', 'https:'].includes(result.protocol) || result.username || result.password) throw new Error('Сервис вернул некорректную ссылку.');
      setShortURL(result.href);
    } catch (failure) {
      setError(failure.name === 'TimeoutError'
        ? 'Сервис не ответил вовремя. Попробуйте ещё раз.'
        : failure instanceof TypeError || failure instanceof SyntaxError
          ? 'Не удалось получить ответ сервиса. Проверьте соединение и повторите попытку.'
          : failure.message);
    } finally {
      submitting.current = false;
      setBusy(false);
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(shortURL);
      setCopyMessage('Ссылка скопирована');
    } catch {
      resultInput.current?.focus();
      resultInput.current?.select();
      setCopyMessage('Скопируйте выделенную ссылку вручную.');
    }
  }

  return (
    <main className="shortener" aria-label="Сокращение ссылок">
      <form onSubmit={shorten} aria-busy={busy}>
        <div className="input-row">
          <input
            id="url"
            aria-label="Исходная ссылка"
            type="url"
            inputMode="url"
            placeholder="https://example.com"
            value={url}
            onChange={(event) => setURL(event.target.value)}
            disabled={busy}
            required
            aria-describedby={error ? 'form-error' : undefined}
            aria-invalid={Boolean(error)}
            autoComplete="url"
          />
          <button type="submit" disabled={busy} aria-label={busy ? 'Сокращение ссылки' : 'Сократить ссылку'} title="Сократить ссылку">
            <span aria-hidden="true">{busy ? '…' : '→'}</span>
          </button>
        </div>
      </form>
      <div className="input-row result">
        <input
          ref={resultInput}
          id="short-url"
          aria-label="Короткая ссылка"
          placeholder="Короткая ссылка"
          value={shortURL}
          readOnly
          onFocus={(event) => event.target.select()}
        />
        <button type="button" onClick={copy} disabled={!shortURL} aria-label="Копировать ссылку" title="Копировать ссылку">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
            {copyMessage === 'Ссылка скопирована'
              ? <path d="m5 12 4 4L19 6" />
              : <><rect x="8" y="8" width="12" height="12" rx="2" /><path d="M16 8V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h3" /></>}
          </svg>
        </button>
        {shortURL && <a className="icon-button" href={shortURL} target="_blank" rel="noreferrer" aria-label="Открыть ссылку" title="Открыть ссылку">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"><path d="M14 3h7v7M21 3 10 14M10 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-5" /></svg>
        </a>}
      </div>
      <span className="sr-only" role="status">{shortURL ? (copyMessage || 'Короткая ссылка готова') : ''}</span>
      {error && <p id="form-error" className="error" role="alert">{error}</p>}
    </main>
  );
}

createRoot(document.getElementById('root')).render(<React.StrictMode><App /></React.StrictMode>);
