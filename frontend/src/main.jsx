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
    <div className="page">
      <header className="header">
        <a className="brand" href="/" aria-label="Shorter — главная"><span className="brand-mark" aria-hidden="true">↗</span>shorter<span className="brand-dot">.</span></a>
        <span className="header-note">Меньше символов. Больше смысла.</span>
      </header>
      <main>
        <div className="eyebrow"><span /> ПРОСТО ССЫЛКА. ТОЛЬКО КОРОЧЕ.</div>
        <h1>Длинные ссылки.<br /><span>Короткое решение.</span></h1>
        <p className="intro">Вставьте ссылку, сократите и поделитесь.<br />Всё нужное — в одном простом действии.</p>
        <section className="card" aria-labelledby="form-title">
          <h2 id="form-title">Сократить ссылку</h2>
          <form onSubmit={shorten} aria-busy={busy}>
            <label htmlFor="url">Ваша длинная ссылка</label>
            <div className="input-row">
              <input id="url" type="url" inputMode="url" placeholder="https://example.com/очень-длинная-ссылка" value={url} onChange={(event) => setURL(event.target.value)} disabled={busy} required aria-describedby={error ? 'url-hint form-error' : 'url-hint'} aria-invalid={Boolean(error)} autoComplete="url" />
              <button type="submit" disabled={busy}>{busy ? 'Сокращаем…' : 'Сократить'}<span aria-hidden="true">↗</span></button>
            </div>
            <p id="url-hint" className="hint">Поддерживаются ссылки с http:// и https://</p>
          </form>
          {error && <p id="form-error" className="error" role="alert">{error}</p>}
          <div aria-live="polite" aria-atomic="true">
            {shortURL && <div className="result">
              <label htmlFor="short-url">Готово! Ваша короткая ссылка</label>
              <div className="input-row">
                <input ref={resultInput} id="short-url" value={shortURL} readOnly onFocus={(event) => event.target.select()} />
                <button type="button" className="copy-button" onClick={copy}>Копировать</button>
              </div>
              <div className="result-footer"><a href={shortURL} target="_blank" rel="noreferrer">Открыть ссылку ↗</a><span>{copyMessage}</span></div>
            </div>}
          </div>
        </section>
        <div className="steps" aria-label="Как это работает">
          <div><span>01</span>Вставьте URL</div><span className="step-line" aria-hidden="true" /><div><span>02</span>Получите короткий</div><span className="step-line" aria-hidden="true" /><div><span>03</span>Поделитесь</div>
        </div>
      </main>
      <footer><span>shorter.</span><span>Хорошим ссылкам не нужна длина.</span></footer>
    </div>
  );
}

createRoot(document.getElementById('root')).render(<React.StrictMode><App /></React.StrictMode>);
