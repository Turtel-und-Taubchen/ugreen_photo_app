export interface Health {
  status: string;
  version: string;
}

export async function fetchHealth(fetchFn: typeof fetch = fetch): Promise<Health> {
  const response = await fetchFn('/api/health');
  if (response.status !== 200) {
    throw new Error(`health request failed with status ${response.status}`);
  }
  const body = (await response.json()) as Health;
  return { status: body.status, version: body.version };
}
