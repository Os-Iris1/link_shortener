const form = document.querySelector('form');
const input = document.querySelector('#url');
const result = document.querySelector('#short-url');
const submit = form.querySelector('button');
const copy = document.querySelector('#copy');
const open = document.querySelector('#open');
const status = document.querySelector('#status');
const error = document.querySelector('#form-error');
const copyIcon = document.querySelector('#copy-icon');
const copiedIcon = document.querySelector('#copied-icon');
const maxURLBytes = 8192;
const requestTimeoutMs = 15_000;
let revision = 0;

function showError(message) {
  error.textContent = message;
  error.hidden = !message;
  input.setAttribute('aria-invalid', String(Boolean(message)));
  if (message) input.setAttribute('aria-describedby', error.id);
  else input.removeAttribute('aria-describedby');
}

function setBusy(busy) {
  form.setAttribute('aria-busy', String(busy));
  input.disabled = busy;
  submit.disabled = busy;
  submit.setAttribute('aria-label', busy ? 'Сокращение ссылки' : 'Сократить ссылку');
  submit.firstElementChild.textContent = busy ? '…' : '→';
}

function setResult(shortURL = '') {
  result.value = shortURL;
  copy.disabled = !shortURL;
  open.classList.toggle('hidden', !shortURL);
  if (shortURL) open.href = shortURL;
  else open.removeAttribute('href');
  copyIcon.classList.remove('hidden');
  copiedIcon.classList.add('hidden');
  status.className = 'sr-only';
  status.textContent = shortURL ? 'Короткая ссылка готова' : '';
}

function isHTTP(url) {
  return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password;
}

form.addEventListener('submit', async (event) => {
  event.preventDefault();
  if (submit.disabled) return;
  revision += 1;
  showError('');
  setResult();
  const value = input.value.trim();
  try {
    if (!isHTTP(new URL(value)) || new TextEncoder().encode(value).length > maxURLBytes) throw new Error();
  } catch {
    showError(`Введите полную ссылку с http:// или https:// без логина и пароля, длиной до ${maxURLBytes} байт.`);
    return;
  }

  setBusy(true);
  try {
    const response = await fetch('/api/links', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: value }),
      signal: AbortSignal.timeout(requestTimeoutMs),
    });
    if (!response.ok) {
      throw new Error(response.status === 400
        ? 'Не удалось сократить этот URL. Проверьте адрес ссылки.'
        : 'Сервис временно недоступен. Попробуйте ещё раз.');
    }
    const data = await response.json();
    const shortURL = new URL(data.short_url);
    if (!isHTTP(shortURL)) throw new Error('Сервис вернул некорректную ссылку.');
    setResult(shortURL.href);
  } catch (failure) {
    showError(failure.name === 'TimeoutError'
      ? 'Сервис не ответил вовремя. Попробуйте ещё раз.'
      : failure instanceof TypeError || failure instanceof SyntaxError
        ? 'Не удалось получить ответ сервиса. Проверьте соединение и повторите попытку.'
        : failure.message);
  } finally {
    setBusy(false);
  }
});

result.addEventListener('focus', () => result.select());
copy.addEventListener('click', async () => {
  const currentRevision = revision;
  let copied = false;
  try {
    await navigator.clipboard.writeText(result.value);
    copied = true;
  } catch {
    // Clipboard access may be unavailable or denied.
  }
  if (currentRevision !== revision) return;
  copyIcon.classList.toggle('hidden', copied);
  copiedIcon.classList.toggle('hidden', !copied);
  status.className = 'copy-status';
  status.textContent = copied ? 'Ссылка скопирована' : 'Скопируйте выделенную ссылку вручную.';
  if (!copied) {
    result.focus();
    result.select();
  }
});
