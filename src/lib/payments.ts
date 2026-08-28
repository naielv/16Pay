export type CardStatus = 'new' | 'active' | 'blocked';
export type Role = 'customer' | 'merchant';
export type TransactionStatus = 'approved' | 'refunded' | 'declined';

export const CURRENCY_CODE = 'FE';
export const CURRENCY_NAME = 'Ferreo';
export const CURRENCY_PLURAL = 'Ferreos';
export const CURRENCY_SYMBOL = 'Fe';

export interface Card {
	id: string;
	name: string;
	status: CardStatus;
	currency: string;
	balance: number;
	customerPin?: string;
	merchantPin?: string;
}

export interface Transaction {
	tid: string;
	amount: number;
	concept: string;
	date: string;
	status: TransactionStatus;
	merchant: string;
	merchantName?: string;
}

export interface Preconfig {
	token: string;
	amount: number;
	concept: string;
	ticket: unknown;
	expiresAt: number;
}

const demoTransactions: Transaction[] = [
	{
		tid: 'tx-01',
		amount: -4.5,
		concept: 'Café y tostada',
		date: 'Hoy, 09:42',
		status: 'approved',
		merchant: 'Café Norte'
	},
	{
		tid: 'tx-02',
		amount: -28,
		concept: 'Compra semanal',
		date: 'Ayer, 18:16',
		status: 'approved',
		merchant: 'Mercado 16'
	},
	{
		tid: 'tx-03',
		amount: 120,
		concept: 'Ingreso recibido',
		date: '12 ago, 11:03',
		status: 'approved',
		merchant: 'Transferencia'
	}
];

export function cardForId(id: string): Card {
	return { id, name: 'Nueva tarjeta', status: 'new', currency: CURRENCY_CODE, balance: 0 };
}

export function transactionsForCard(cardId: string): Transaction[] {
	if (typeof localStorage !== 'undefined') {
		const saved = localStorage.getItem(`16pay:transactions:${cardId}`);
		if (saved) return JSON.parse(saved) as Transaction[];
	}
	return cardId === 'demo' || cardId === 'ana' || cardId.startsWith('demo') ? demoTransactions : [];
}

export function saveTransactions(cardId: string, transactions: Transaction[]) {
	localStorage.setItem(`16pay:transactions:${cardId}`, JSON.stringify(transactions));
}
export function saveCard(card: Card) {
	localStorage.setItem(`16pay:card:${card.id}`, JSON.stringify(card));
}

export function hydrateCard(card: Card): Card {
	if (typeof localStorage === 'undefined') return card;
	const saved = localStorage.getItem(`16pay:card:${card.id}`);
	return saved ? (JSON.parse(saved) as Card) : card;
}

export function parseAmount(value: string): number | null {
	const normalized = value.replace(',', '.').trim();
	if (!/^\d+(\.\d{1,2})?$/.test(normalized)) return null;
	const amount = Number(normalized);
	return amount > 0 && amount <= 100000 ? Math.round(amount * 100) / 100 : null;
}

export function money(amount: number) {
	return `${new Intl.NumberFormat('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(amount)} ${CURRENCY_SYMBOL}`;
}

export function readPreconfig(): Preconfig | null {
	if (typeof localStorage === 'undefined') return null;
	const raw = localStorage.getItem('16pay:preconfig');
	if (!raw) return null;
	const preconfig = JSON.parse(raw) as Preconfig;
	if (preconfig.expiresAt < Date.now()) {
		localStorage.removeItem('16pay:preconfig');
		return null;
	}
	return preconfig;
}

export function savePreconfig(preconfig: Preconfig) {
	localStorage.setItem('16pay:preconfig', JSON.stringify(preconfig));
}
export function clearPreconfig() {
	localStorage.removeItem('16pay:preconfig');
}
