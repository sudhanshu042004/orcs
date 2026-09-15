import { apiUrl } from './contants';

type RequestOptions = Omit<RequestInit, 'body'> & {
  body?: BodyInit | Record<string, unknown> | null;
};

async function api<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const { body, ...rest } = options;

  const finalBody =
    body instanceof FormData || body === null || body === undefined
      ? body
      : body instanceof ArrayBuffer || body instanceof Blob
        ? body as BodyInit
        : JSON.stringify(body);

  const headers: HeadersInit = {
    ...(!(body instanceof FormData) && body != null
      ? { 'Content-Type': 'application/json' }
      : {}),
    ...(options.headers ?? {}),
  };

  const response = await fetch(`${apiUrl}${endpoint}`, {
    ...rest,
    body: finalBody as BodyInit | undefined,
    headers,
    credentials: 'include',
  });

  if (!response.ok) {
    // Surface the server's own error message when it sends one
    const message = await response
      .json()
      .then((data) => data?.error as string | undefined)
      .catch(() => undefined);

    throw new Error(message || `API Error: ${response.status} ${response.statusText}`);
  }

  return response.json() as Promise<T>;
}

export const get = <T>(endpoint: string, options?: RequestOptions) =>
  api<T>(endpoint, { ...options, method: 'GET' });

export const post = <T>(
  endpoint: string,
  body?: RequestOptions['body'],
  options?: RequestOptions
) =>
  api<T>(endpoint, {
    ...options,
    method: 'POST',
    body,
  });

export const put = <T>(
  endpoint: string,
  body?: RequestOptions['body'],
  options?: RequestOptions
) =>
  api<T>(endpoint, {
    ...options,
    method: 'PUT',
    body,
  });

export const del = <T>(endpoint: string, options?: RequestOptions) =>
  api<T>(endpoint, { ...options, method: 'DELETE' });