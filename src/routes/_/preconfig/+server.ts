import { json } from '@sveltejs/kit';
import { parseAmount } from '$lib/payments';
import type { RequestHandler } from './$types';

const preconfigs = new Map<
	string,
	{ amount: number; concept: string; ticket: unknown; expiresAt: string }
>();

export const POST: RequestHandler = async ({ request, url }) => {
	const amount = parseAmount(url.searchParams.get('amount') ?? '');
	const concept = url.searchParams.get('concept')?.trim() ?? '';
	if (!amount || !concept || concept.length > 120) {
		return json({ message: 'Importe o concepto no válido.' }, { status: 400 });
	}

	let body: { ticket?: unknown } = {};
	try {
		body = (await request.json()) as { ticket?: unknown };
	} catch {
		/* ticket opcional */
	}
	const token = crypto.randomUUID();
	const expiresAt = new Date(Date.now() + 10 * 60 * 1000).toISOString();
	preconfigs.set(token, { amount, concept, ticket: body.ticket ?? null, expiresAt });
	return json({ token, status: 'pending_scan', expiresAt });
};
