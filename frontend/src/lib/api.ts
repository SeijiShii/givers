const API_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080';

export async function fetchApi<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }

  return res.json();
}

export async function healthCheck(): Promise<{ status: string; message: string }> {
  return fetchApi('/api/health');
}
