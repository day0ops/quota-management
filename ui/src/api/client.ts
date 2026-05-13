import { ApiError } from './types';

const API_BASE = '/api/v1';

export class ApiClientError extends Error {
  constructor(
    message: string,
    public status: number,
    public isConflict: boolean = false,
    public isDuplicate: boolean = false
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}

async function fetchApi(url: string, options: RequestInit): Promise<Response> {
  try {
    const response = await fetch(url, options);
    return response;
  } catch (error) {
    // Network errors during fetch typically mean:
    // 1. CORS blocked a redirect to auth provider (session expired)
    // 2. Actual network failure
    // For case 1, the OIDC flow redirects to IdP which triggers CORS error
    // A page refresh will re-trigger the oidcAuthorizationCode flow properly
    if (error instanceof TypeError) {
      console.warn(
        'Session expired (CORS error on auth redirect). Refreshing to re-authenticate...'
      );
      window.location.reload();
      // Return a never-resolving promise since we're reloading
      return new Promise(() => {});
    }
    throw error;
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = 'An error occurred';
    let isDuplicate = false;
    try {
      const errorData: ApiError = await response.json();
      message = errorData.error?.message || message;
      // Detect duplicate entity errors
      if (
        message.includes('duplicate key') ||
        message.includes('23505') ||
        message.includes('already exists')
      ) {
        isDuplicate = true;
      }
    } catch {
      message = response.statusText || message;
    }

    // For 401 with no specific message, hint at session expiration
    if (response.status === 401 && message === 'An error occurred') {
      message = 'Session expired. Please refresh the page.';
    }

    // For 409 conflicts, distinguish between duplicate and optimistic lock conflicts
    const isConflict = response.status === 409 && !isDuplicate;
    throw new ApiClientError(message, response.status, isConflict, isDuplicate);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

export const apiClient = {
  async get<T>(path: string): Promise<T> {
    const response = await fetchApi(`${API_BASE}${path}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });
    return handleResponse<T>(response);
  },

  async post<T, D = unknown>(path: string, data?: D): Promise<T> {
    const response = await fetchApi(`${API_BASE}${path}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: data ? JSON.stringify(data) : undefined,
    });
    return handleResponse<T>(response);
  },

  async put<T, D = unknown>(path: string, data: D): Promise<T> {
    const response = await fetchApi(`${API_BASE}${path}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    return handleResponse<T>(response);
  },

  async delete<T>(path: string): Promise<T> {
    const response = await fetchApi(`${API_BASE}${path}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
    });
    return handleResponse<T>(response);
  },
};
