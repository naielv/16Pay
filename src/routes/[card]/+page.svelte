<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import {
		cardForId,
		clearPreconfig,
		hydrateCard,
		money,
		parseAmount,
		readPreconfig,
		saveCard,
		saveTransactions,
		transactionsForCard,
		type Card,
		type Role,
		type Transaction
	} from '$lib/payments';

	let card = $state<Card>(cardForId(page.params.card ?? 'new'));
	let transactions = $state<Transaction[]>([]);
	let role = $state<Role | null>(null);
	let onboardingStep = $state(1);
	let customerPin = $state('');
	let merchantPin = $state('');
	let pin = $state('');
	let pinError = $state('');
	let pinAction = $state('');
	let pendingAction = $state<(() => void) | null>(null);
	let amount = $state('');
	let concept = $state('');
	let ticket = $state('');
	let terminalStep = $state<'amount' | 'pin' | 'done'>('amount');
	let terminalMessage = $state('');
	let preconfigured = $state(false);
	let apiAvailable = $state(false);
	let customerChargePin = $state('');
	let authorizedPin = $state('');
	let actionMessage = $state('');
	let merchantLoginOpen = $state(false);
	let merchantEmail = $state('demo@16pay.local');
	let merchantPassword = $state('demo12345');
	let merchantToken = $state('');
	let merchantName = $state('');
	let merchantLoginError = $state('');
	let loading = $state(true);

	onMount(async () => {
		card = hydrateCard(card);
		try {
			const cardResponse = await fetch(`/api/16pay/cards/${card.id}`);
			if (cardResponse.ok) {
				const remoteCard = await cardResponse.json();
				card = { ...card, ...remoteCard };
				apiAvailable = true;
			}
		} catch {
			apiAvailable = false;
		}
		const preconfigToken = page.url.searchParams.get('preconfig');
		if (preconfigToken && apiAvailable) {
			const response = await fetch(`/api/16pay/preconfigs/${preconfigToken}/consume`, { method: 'POST' });
			if (response.ok) {
				const remotePreconfig = await response.json();
				amount = (remotePreconfig.amount / 100).toFixed(2);
				concept = remotePreconfig.concept;
				ticket = typeof remotePreconfig.ticket === 'string' ? remotePreconfig.ticket : JSON.stringify(remotePreconfig.ticket, null, 2);
				preconfigured = true;
			}
		}
		const preconfig = readPreconfig();
		if (preconfig) {
			amount = preconfig.amount.toFixed(2);
			concept = preconfig.concept;
			ticket =
				typeof preconfig.ticket === 'string'
					? preconfig.ticket
					: JSON.stringify(preconfig.ticket, null, 2);
			preconfigured = true;
			clearPreconfig();
		}
		loading = false;
	});

	function startRole(nextRole: Role) {
		role = nextRole;
		if (nextRole === 'merchant') {
			merchantLoginOpen = true;
			merchantLoginError = '';
		} else {
			openPin('Acceder al espacio cliente', loadCustomerData);
		}
	}

	async function loginMerchant() {
		merchantLoginError = '';
		try {
			const response = await fetch('/api/16pay/merchants/login', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ email: merchantEmail, password: merchantPassword }) });
			if (!response.ok) throw new Error('Correo o contraseña incorrectos.');
			const result = await response.json();
			merchantToken = result.token;
			merchantName = result.name;
			merchantLoginOpen = false;
		} catch (error) {
			merchantLoginError = error instanceof Error ? error.message : 'No se pudo iniciar sesión.';
		}
	}

	async function loadCustomerData() {
		if (!apiAvailable) {
			transactions = transactionsForCard(card.id);
			return;
		}
		const balanceResponse = await fetch(`/api/16pay/cards/${card.id}/balance`, { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ role: 'customer', pin: authorizedPin }) });
		if (balanceResponse.ok) {
			const balance = await balanceResponse.json();
			card = { ...card, balance: balance.balance / 100 };
		}
		const transactionResponse = await fetch(`/api/16pay/cards/${card.id}/transactions`, { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ role: 'customer', pin: authorizedPin }) });
		if (transactionResponse.ok) transactions = await transactionResponse.json();
	}

	function openPin(action: string, callback: () => void) {
		pinAction = action;
		pin = '';
		pinError = '';
		actionMessage = '';
		pendingAction = callback;
	}

	async function verifyPin() {
		const expected = role === 'merchant' ? card.merchantPin : card.customerPin;
		let verified = pin === expected;
		if (apiAvailable) {
			try {
				const response = await fetch(`/api/16pay/cards/${card.id}/verify-pin`, { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ role, pin }) });
				verified = response.ok;
			} catch { verified = false; }
		}
		if (!verified) {
			pinError = 'PIN incorrecto. Comprueba los cuatro dígitos.';
			pin = '';
			return;
		}
		const callback = pendingAction;
		authorizedPin = pin;
		pendingAction = null;
		pin = '';
		pinError = '';
		if (callback) callback();
	}

	async function completeOnboarding() {
		if (!/^\d{4}$/.test(customerPin) || !/^\d{4}$/.test(merchantPin)) return;
		if (apiAvailable) {
			const response = await fetch(`/api/16pay/cards/${card.id}/onboard`, { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ name: 'Mi tarjeta', customerPin, merchantPin }) });
			if (!response.ok) return;
		}
		card = { ...card, name: 'Mi tarjeta', customerPin, merchantPin };
		saveCard(card);
		transactions = [];
		onboardingStep = 3;
	}

	async function refundLatest() {
		const transaction = transactions.find((item) => item.status === 'approved');
		if (!transaction) {
			actionMessage = 'No hay operaciones reembolsables.';
			return;
		}
		if (apiAvailable) {
			const response = await fetch(`/api/16pay/cards/${card.id}/transactions/${transaction.tid}/refund`, {
				method: 'POST',
				headers: { Authorization: `Bearer ${merchantToken}`, 'content-type': 'application/json' },
				body: JSON.stringify({ role: 'merchant' })
			});
			if (!response.ok) {
				actionMessage = 'No se pudo tramitar el reembolso.';
				return;
			}
		}
		transactions = transactions.map((item) => item.tid === transaction.tid ? { ...item, status: 'refunded' } : item);
		if (!apiAvailable) saveTransactions(card.id, transactions);
		actionMessage = 'Reembolso preparado correctamente.';
	}

	function finishOnboarding() {
		card = { ...card, status: 'active' };
		saveCard(card);
		onboardingStep = 4;
	}

	function beginCharge() {
		const parsed = parseAmount(amount);
		if (!parsed || !concept.trim()) {
			terminalMessage = 'Introduce un importe válido y un concepto.';
			return;
		}
		terminalMessage = '';
		terminalStep = 'pin';
	}

	async function confirmCharge() {
		const parsed = parseAmount(amount);
		if (!parsed) return;
		if (apiAvailable) {
			const response = await fetch(`/api/16pay/cards/${card.id}/charge`, { method: 'POST', headers: { Authorization: `Bearer ${merchantToken}`, 'content-type': 'application/json' }, body: JSON.stringify({ merchantPin: customerChargePin, merchantToken, amount: Math.round(parsed * 100), concept: concept.trim(), ticket: ticket.trim() || null, idempotencyKey: `web-${Date.now()}-${Math.random().toString(36).slice(2)}` }) });
			if (!response.ok) { terminalMessage = (await response.json()).message ?? 'No se pudo aprobar el pago.'; terminalStep = 'amount'; return; }
			const transactionResponse = await fetch(`/api/16pay/cards/${card.id}/transactions`, { method: 'POST', headers: { Authorization: `Bearer ${merchantToken}`, 'content-type': 'application/json' }, body: JSON.stringify({ role: 'merchant' }) });
			if (transactionResponse.ok) transactions = await transactionResponse.json();
			terminalStep = 'done'; terminalMessage = 'Pago aprobado'; return;
		}
		if (customerChargePin !== card.merchantPin) { terminalMessage = 'PIN del comercio incorrecto.'; return; }
		const transaction: Transaction = {
			tid: `tx-${Date.now()}`,
			amount: -parsed,
			concept: concept.trim(),
			date: 'Ahora',
			status: 'approved',
			merchant: merchantName || 'Café Norte'
		};
		transactions = [transaction, ...transactions];
		saveTransactions(card.id, transactions);
		terminalStep = 'done';
		terminalMessage = 'Pago aprobado';
	}

	function resetTerminal() {
		amount = '';
		concept = '';
		ticket = '';
		terminalStep = 'amount';
		terminalMessage = '';
		preconfigured = false;
	}

	function formatTicket() {
		if (!ticket.trim()) return;
		try {
			ticket = JSON.stringify(JSON.parse(ticket), null, 2);
		} catch {
			/* texto plano válido */
		}
	}
</script>

<svelte:head>
	<title>{loading ? '16Pay' : `${card.name} | 16Pay`}</title>
	<meta name="theme-color" content="#17211d" />
</svelte:head>

<main class="app-shell">
	<header class="topbar">
		<a class="brand" href="/">16<span>PAY</span></a>
		<div class="scan-status">
			<i></i> Tarjeta escaneada <strong>•••• {card.id.slice(-4)}</strong>
		</div>
		<a class="exit" href="/">Salir</a>
	</header>

	{#if loading}
		<section class="loading-state" aria-live="polite"><span></span><p>Cargando tarjeta</p></section>
	{:else}
	{#if card.status === 'new' && onboardingStep < 3}
		<section class="onboarding content-width">
			<div class="step-count">0{onboardingStep} <span>/ 02</span></div>
			<p class="eyebrow">NUEVA TARJETA DETECTADA</p>
			<h1>Hagámosla<br /><em>tuya.</em></h1>
			<p class="lead">
				Configura dos PIN independientes para mantener cada parte de tu dinero en su sitio.
			</p>
			{#if onboardingStep === 1}
				<div class="onboard-panel">
					<div class="panel-number">01</div>
					<div>
						<h2>PIN de cliente</h2>
						<p>Para consultar saldo, ver movimientos y gestionar tus pagos.</p>
						<input
							bind:value={customerPin}
							maxlength="4"
							inputmode="numeric"
							type="password"
							placeholder="••••"
							aria-label="PIN de cliente"
						/>
					</div>
				</div>
				<button
					class="button dark"
					disabled={!/^\d{4}$/.test(customerPin)}
					onclick={() => (onboardingStep = 2)}>Continuar <span>→</span></button
				>
			{:else}
				<div class="onboard-panel">
					<div class="panel-number">02</div>
					<div>
						<h2>PIN de comercio</h2>
						<p>Para cobrar y devolver pagos desde el datáfono.</p>
						<input
							bind:value={merchantPin}
							maxlength="4"
							inputmode="numeric"
							type="password"
							placeholder="••••"
							aria-label="PIN de comercio"
						/>
					</div>
				</div>
				<button
					class="button dark"
					disabled={!/^\d{4}$/.test(merchantPin)}
					onclick={completeOnboarding}>Activar tarjeta <span>→</span></button
				>
			{/if}
		</section>
	{:else if card.status === 'new' && onboardingStep === 3}
		<section class="activated content-width">
			<span class="success-check">✓</span>
			<p class="eyebrow">TARJETA ACTIVADA</p>
			<h1>Todo listo,<br /><em>a disfrutar.</em></h1>
			<button class="button dark" onclick={finishOnboarding}>Entrar a 16Pay <span>→</span></button>
		</section>
	{:else if !role}
		<section class="home content-width">
			<div class="welcome-row">
				<div>
					<p class="eyebrow">BUENAS, {card.name.toUpperCase()}</p>
					<h1>¿Qué necesitas<br /><em>hacer hoy?</em></h1>
				</div>
				<div class="card-orbit"><span>16</span><small>NFC</small></div>
			</div>
			<div class="preconfig-notice" class:visible={preconfigured}>
				<span>✦</span>
				<div>
					<strong>Operación preparada</strong>
					<p>El datáfono tiene los datos listos para este escaneo.</p>
				</div>
				<button
					onclick={() => {
						role = 'merchant';
						merchantLoginOpen = true;
					}}>Abrir <span>→</span></button
				>
			</div>
			<p class="section-label">ELIGE UN ESPACIO</p>
			<div class="role-grid">
				<button class="role-card customer" onclick={() => startRole('customer')}
					><span class="role-icon">◌</span><span class="role-name">Soy el cliente</span><span
						class="role-description">Saldo, pagos y movimientos</span
					><span class="arrow">↗</span></button
				>
				<button class="role-card merchant" onclick={() => startRole('merchant')}
					><span class="role-icon">▣</span><span class="role-name">Soy el comercio</span><span
						class="role-description">Cobrar, devolver y gestionar</span
					><span class="arrow">↗</span></button
				>
			</div>
			<div class="trust-line"><span>●</span> El escaneo inicial no autoriza ninguna operación</div>
		</section>
	{:else if role === 'customer'}
		<section class="dashboard content-width">
			<button class="back" onclick={() => (role = null)}>← Espacios</button>
			<div class="dash-heading">
				<div>
					<p class="eyebrow">ESPACIO CLIENTE</p>
					<h1>Tu dinero,<br /><em>de un vistazo.</em></h1>
				</div>
				<span class="live-dot">● Activo</span>
			</div>
			<div class="balance-card">
				<div><span class="muted">SALDO DISPONIBLE</span><strong>{money(card.balance)}</strong></div>
				<span class="balance-mark">16</span>
			</div>
			<!-- <div class="dash-actions">
						<button onclick={() => openPin('Consultar saldo', () => (actionMessage = 'Saldo actualizado ahora.'))}
					><span>＋</span><b>Consultar saldo</b><small>Actualizado ahora</small></button
						><button onclick={() => openPin('Ver movimientos', () => (actionMessage = 'Movimientos verificados.'))}
					><span>≡</span><b>Ver movimientos</b><small>Últimas operaciones</small></button
				>
			</div> -->
			<div class="activity-head">
				<p class="section-label">ACTIVIDAD RECIENTE</p>
				<button onclick={() => openPin('Ver historial completo', () => (actionMessage = 'Historial completo verificado.'))}
					>Ver todo →</button
				>
			</div>
			<div class="transactions">
				{#each transactions as transaction}<div class="transaction">
						<span class="transaction-icon">{transaction.amount > 0 ? '↑' : '↓'}</span>
						<div>
							<strong>{transaction.concept}</strong><small
								>{transaction.merchant} · {transaction.date}</small
							>
						</div>
						<b class:positive={transaction.amount > 0}
							>{transaction.amount > 0 ? '+' : ''}{money(transaction.amount)}</b
						>
					</div>{/each}
			</div>
		</section>
	{:else}
		<section class="dashboard merchant-space content-width">
			<button class="back" onclick={() => (role = null)}>← Espacios</button>
			<div class="dash-heading">
				<div>
					<p class="eyebrow">ESPACIO COMERCIO</p>
					<h1>Listo para<br /><em>cobrar.</em></h1>
				</div>
				<span class="live-dot">● Terminal activa</span>
			</div>
			<div class="terminal">
				<div class="terminal-top">
					<span>DATÁFONO <b>01</b></span><span class="secure">● SEGURO</span>
				</div>
				{#if terminalStep === 'amount'}
					<div class="terminal-body">
						<p class="terminal-step">PASO 01 <span>/ 02</span></p>
						<h2>¿Cuánto cobramos?</h2>
						<div class="amount-input">
							<span>Fe</span><input
								bind:value={amount}
								inputmode="decimal"
								placeholder="0,00"
								aria-label="Importe"
							/>
						</div>
						<input
							class="line-input"
							bind:value={concept}
							placeholder="Concepto del cobro"
							aria-label="Concepto"
						/><textarea
							class="line-input ticket-input"
							bind:value={ticket}
							onblur={formatTicket}
							placeholder="Ticket (opcional: texto o JSON)"
							aria-label="Ticket"></textarea>{#if preconfigured}<p class="prefill">
								<span>✦</span> Datos precargados desde la integración
							</p>{/if}{#if terminalMessage}<p class="form-error">{terminalMessage}</p>{/if}<button
							class="button coral full"
							onclick={beginCharge}>Continuar <span>→</span></button
						>
					</div>
				{:else if terminalStep === 'pin'}
					<div class="terminal-body centered">
						<p class="terminal-step">PASO 02 <span>/ 02</span></p>
						<div class="pin-ring">⌁</div>
						<h2>Confirma el cobro</h2>
						<p>El cliente introduce el PIN del comercio para autorizar el pago.</p>
						<div class="pin-dots">
							{#each [0, 1, 2, 3] as index}<i class:filled={pin.length > index}></i>{/each}
						</div>
						<input
							bind:value={pin}
							maxlength="4"
							inputmode="numeric"
							type="password"
							class="pin-input"
							placeholder="PIN del comercio"
							aria-label="PIN del comercio"
						/>
						<div class="terminal-buttons">
							<button
								class="button ghost"
								onclick={() => {
									terminalStep = 'amount';
									pin = '';
								}}>Atrás</button
							><button
								class="button dark"
								disabled={!/^\d{4}$/.test(pin)}
								onclick={() => { customerChargePin = pin; confirmCharge(); }}
								>Autorizar <span>→</span></button
							>
						</div>
					</div>
				{:else}
					<div class="terminal-body centered success">
						<div class="success-check">✓</div>
						<p class="terminal-step">OPERACIÓN COMPLETADA</p>
						<h2>Pago aprobado</h2>
						<strong>{money(parseAmount(amount) ?? 0)}</strong>
						<p>{concept}</p>
						<button class="button dark" onclick={resetTerminal}>Nuevo cobro <span>＋</span></button>
					</div>
				{/if}
			</div>
			<div class="merchant-links">
							<button onclick={() => (actionMessage = `Operaciones de ${merchantName || 'tu comercio'} verificadas.`)}>▤ Operaciones</button
				><button onclick={refundLatest}
					>↩ Reembolsar</button
				>
			</div>
			{#if actionMessage}<p class="action-message">{actionMessage}</p>{/if}
		</section>
	{/if}
	{/if}

	{#if merchantLoginOpen}
		<div class="modal-backdrop" role="presentation">
			<div class="pin-modal merchant-login" role="dialog" aria-modal="true" aria-label="Acceso comercio">
				<button class="modal-close" aria-label="Cancelar" onclick={() => { merchantLoginOpen = false; role = null; }}>×</button>
				<span class="modal-lock">▣</span>
				<p class="eyebrow">ACCESO COMERCIO</p>
				<h2>Entra a tu cuenta</h2>
				<p>Usa las credenciales de tu comercio para gestionar el datáfono.</p>
				<input class="login-input" bind:value={merchantEmail} type="email" autocomplete="username" placeholder="Correo electrónico" aria-label="Correo electrónico" />
				<input class="login-input" bind:value={merchantPassword} type="password" autocomplete="current-password" placeholder="Contraseña" aria-label="Contraseña" onkeydown={(event) => event.key === 'Enter' && loginMerchant()} />
				{#if merchantLoginError}<p class="form-error">{merchantLoginError}</p>{/if}
				<button class="button dark full" onclick={loginMerchant}>Entrar <span>→</span></button>
				<small class="demo-hint">Demo: demo@16pay.local · demo12345</small>
			</div>
		</div>
	{/if}

	{#if pendingAction}
		<div class="modal-backdrop" role="presentation">
			<div class="pin-modal" role="dialog" aria-modal="true" aria-label="Autorización PIN">
				<button class="modal-close" aria-label="Cancelar" onclick={() => (pendingAction = null)}
					>×</button
				><span class="modal-lock">⌁</span>
				<p class="eyebrow">AUTORIZACIÓN NECESARIA</p>
				<h2>{pinAction}</h2>
				<p>Introduce tu PIN para continuar.</p>
				<div class="pin-dots">
					{#each [0, 1, 2, 3] as index}<i class:filled={pin.length > index}></i>{/each}
				</div>
				<input
					bind:value={pin}
					maxlength="4"
					inputmode="numeric"
					type="password"
					class="pin-input"
					placeholder="••••"
					aria-label="PIN"
					onkeydown={(event) => event.key === 'Enter' && verifyPin()}
				/>{#if pinError}<p class="form-error">{pinError}</p>{/if}<button
					class="button dark full"
					disabled={!/^\d{4}$/.test(pin)}
					onclick={verifyPin}>Confirmar PIN <span>→</span></button
				><small class="demo-hint">Demo: cliente 2468 · comercio 1357</small>
			</div>
		</div>
	{/if}
</main>

<style>
	:global(*) {
		box-sizing: border-box;
	}
	:global(body) {
		margin: 0;
		background: #f3f0e9;
		color: #17211d;
		font-family: 'Avenir Next', 'Helvetica Neue', sans-serif;
	}
	:global(button),
	:global(input),
	:global(textarea) {
		font: inherit;
	}
	:global(button) {
		cursor: pointer;
	}
	.app-shell {
		min-height: 100vh;
		background:
			radial-gradient(circle at 88% 8%, rgba(207, 230, 212, 0.7), transparent 27%), #f3f0e9;
	}
	.topbar {
		height: 80px;
		padding: 0 clamp(22px, 5vw, 72px);
		display: flex;
		align-items: center;
		justify-content: space-between;
		border-bottom: 1px solid rgba(23, 33, 29, 0.1);
	}
	.brand {
		color: #e75e3f;
		font-size: 1.55rem;
		font-weight: 800;
		letter-spacing: -0.1em;
		text-decoration: none;
	}
	.brand span {
		margin-left: 5px;
		color: #17211d;
		font-size: 0.58rem;
		letter-spacing: 0.14em;
	}
	.scan-status,
	.exit {
		color: #849087;
		font-size: 0.7rem;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	.scan-status i,
	.live-dot {
		color: #4d9566;
		font-style: normal;
	}
	.scan-status strong {
		margin-left: 12px;
		color: #17211d;
		font-weight: 600;
	}
	.exit {
		color: #17211d;
		text-decoration: none;
	}
	.content-width {
		width: min(100% - 44px, 1080px);
		margin: 0 auto;
	}
	.loading-state { min-height: calc(100vh - 80px); display: grid; place-content: center; justify-items: center; gap: 14px; color: #849087; font-size: .75rem; letter-spacing: .12em; text-transform: uppercase; }
	.loading-state span { width: 24px; height: 24px; border: 2px solid #cbd3cd; border-top-color: #e75e3f; border-radius: 50%; animation: loading-spin .8s linear infinite; }
	@keyframes loading-spin { to { transform: rotate(360deg); } }
	.onboarding,
	.home,
	.dashboard {
		padding: 10vh 0 60px;
	}
	.eyebrow,
	.section-label {
		color: #52715f;
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.16em;
	}
	.step-count {
		margin-bottom: 14vh;
		color: #e75e3f;
		font-size: 2.2rem;
		font-weight: 500;
		letter-spacing: -0.08em;
	}
	.step-count span {
		color: #9aa49c;
		font-size: 1rem;
		letter-spacing: 0;
	}
	.onboarding h1,
	.home h1,
	.dashboard h1 {
		margin: 16px 0;
		font-size: clamp(3.6rem, 8vw, 7.4rem);
		line-height: 0.84;
		letter-spacing: -0.095em;
		font-weight: 500;
	}
	.onboarding h1 em,
	.home h1 em,
	.dashboard h1 em {
		color: #e75e3f;
		font-family: Georgia, serif;
		font-weight: 400;
		letter-spacing: -0.08em;
	}
	.lead {
		max-width: 390px;
		color: #69756d;
		line-height: 1.5;
	}
	.onboard-panel {
		max-width: 600px;
		margin: 50px 0 30px;
		padding: 24px 0;
		display: flex;
		gap: 28px;
		border-top: 1px solid #bdc7be;
	}
	.panel-number {
		color: #e75e3f;
		font-weight: 700;
	}
	.onboard-panel h2 {
		margin: 0 0 5px;
		font-size: 1.25rem;
	}
	.onboard-panel p {
		margin: 0 0 20px;
		color: #78837c;
		font-size: 0.9rem;
	}
	.onboard-panel input {
		width: 140px;
		padding: 12px 0;
		border: 0;
		border-bottom: 2px solid #17211d;
		outline: 0;
		background: transparent;
		font-size: 1.4rem;
		letter-spacing: 0.35em;
	}
	.button {
		border: 0;
		padding: 15px 20px;
		display: inline-flex;
		justify-content: space-between;
		align-items: center;
		gap: 30px;
		border-radius: 2px;
		font-size: 0.86rem;
		font-weight: 700;
	}
	.button.dark {
		background: #17211d;
		color: #fff;
	}
	.button.coral {
		background: #e75e3f;
		color: #fff;
	}
	.button.ghost {
		background: transparent;
		border: 1px solid #c7cec8;
		color: #17211d;
	}
	.button:disabled {
		cursor: not-allowed;
		opacity: 0.35;
	}
	.welcome-row,
	.dash-heading {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
	}
	.card-orbit {
		width: 120px;
		height: 120px;
		display: grid;
		place-content: center;
		border: 1px solid #e75e3f;
		border-radius: 50%;
		color: #e75e3f;
		transform: rotate(-12deg);
	}
	.card-orbit span {
		font-size: 2rem;
		font-weight: 700;
		letter-spacing: -0.12em;
	}
	.card-orbit small {
		margin-top: -4px;
		text-align: center;
		font-size: 0.6rem;
		letter-spacing: 0.2em;
	}
	.section-label {
		margin: 9vh 0 16px;
	}
	.role-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 18px;
	}
	.role-card {
		min-height: 235px;
		position: relative;
		padding: 25px;
		border: 0;
		text-align: left;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		justify-content: flex-end;
		color: #fff;
	}
	.role-card.customer {
		background: #52715f;
	}
	.role-card.merchant {
		background: #e75e3f;
	}
	.role-icon {
		position: absolute;
		top: 23px;
		left: 25px;
		font-size: 2.3rem;
		font-weight: 300;
	}
	.role-name {
		font-size: 1.3rem;
		font-weight: 700;
	}
	.role-description {
		margin-top: 8px;
		color: rgba(255, 255, 255, 0.72);
		font-size: 0.8rem;
	}
	.arrow {
		position: absolute;
		right: 25px;
		bottom: 22px;
		font-size: 1.5rem;
	}
	.trust-line {
		margin-top: 20px;
		color: #8a958d;
		font-size: 0.72rem;
	}
	.trust-line span {
		color: #4d9566;
	}
	.preconfig-notice {
		display: none;
		align-items: center;
		gap: 14px;
		margin-top: 36px;
		padding: 14px 18px;
		border-left: 3px solid #e75e3f;
		background: rgba(231, 94, 63, 0.08);
	}
	.preconfig-notice.visible {
		display: flex;
	}
	.preconfig-notice > span {
		color: #e75e3f;
		font-size: 1.3rem;
	}
	.preconfig-notice p {
		margin: 4px 0 0;
		color: #78837c;
		font-size: 0.78rem;
	}
	.preconfig-notice button {
		margin-left: auto;
		border: 0;
		background: transparent;
		font-weight: 700;
	}
	.balance-card {
		margin: 45px 0 18px;
		padding: 28px;
		display: flex;
		justify-content: space-between;
		background: #17211d;
		color: #fff;
	}
	.muted {
		display: block;
		color: #94a49a;
		font-size: 0.68rem;
		letter-spacing: 0.14em;
	}
	.balance-card strong {
		display: block;
		margin-top: 12px;
		font-size: 3rem;
		font-weight: 500;
		letter-spacing: -0.08em;
	}
	.balance-mark {
		color: #e75e3f;
		font-size: 2rem;
		font-weight: 800;
	}
	.dash-actions {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 18px;
	}
	.dash-actions button {
		padding: 18px;
		border: 1px solid #cbd3cd;
		background: transparent;
		text-align: left;
	}
	.dash-actions span {
		display: block;
		color: #e75e3f;
		font-size: 1.5rem;
	}
	.dash-actions b,
	.dash-actions small {
		display: block;
		margin-top: 12px;
	}
	.dash-actions small {
		color: #869188;
		font-size: 0.72rem;
	}
	.activity-head {
		margin-top: 55px;
		display: flex;
		justify-content: space-between;
		align-items: baseline;
	}
	.activity-head button,
	.merchant-links button {
		border: 0;
		background: transparent;
		color: #52715f;
		font-size: 0.75rem;
		font-weight: 700;
	}
	.transactions {
		border-top: 1px solid #cbd3cd;
	}
	.transaction {
		padding: 16px 0;
		display: flex;
		align-items: center;
		gap: 14px;
		border-bottom: 1px solid #d9ded9;
	}
	.transaction-icon {
		width: 32px;
		height: 32px;
		display: grid;
		place-content: center;
		border-radius: 50%;
		background: #dce9df;
		color: #52715f;
	}
	.transaction strong,
	.transaction small {
		display: block;
	}
	.transaction small {
		margin-top: 5px;
		color: #89938b;
		font-size: 0.72rem;
	}
	.transaction b {
		margin-left: auto;
	}
	.transaction b.positive {
		color: #4d9566;
	}
	.back {
		padding: 0;
		border: 0;
		background: transparent;
		color: #52715f;
		font-size: 0.8rem;
	}
	.dash-heading {
		margin-top: 10vh;
	}
	.live-dot {
		font-size: 0.7rem;
	}
	.terminal {
		max-width: 560px;
		margin: 42px auto 0;
		background: #fff;
		box-shadow: 0 16px 40px rgba(23, 33, 29, 0.09);
	}
	.terminal-top {
		padding: 15px 20px;
		display: flex;
		justify-content: space-between;
		border-bottom: 1px solid #e2e6e2;
		color: #77847b;
		font-size: 0.65rem;
		letter-spacing: 0.14em;
	}
	.terminal-top b {
		color: #e75e3f;
	}
	.secure {
		color: #4d9566;
	}
	.terminal-body {
		padding: 35px;
	}
	.terminal-step {
		color: #e75e3f;
		font-size: 0.66rem;
		font-weight: 700;
		letter-spacing: 0.15em;
	}
	.terminal-step span {
		color: #a1aaa3;
	}
	.terminal-body h2 {
		margin: 28px 0;
		font-size: 1.6rem;
	}
	.amount-input {
		display: flex;
		align-items: baseline;
		gap: 8px;
		border-bottom: 2px solid #17211d;
	}
	.amount-input span {
		color: #e75e3f;
		font-size: 1.4rem;
	}
	.amount-input input {
		width: 100%;
		padding: 8px 0;
		border: 0;
		outline: 0;
		background: transparent;
		font-size: 3.4rem;
		letter-spacing: -0.08em;
	}
	.line-input {
		width: 100%;
		padding: 16px 0 10px;
		border: 0;
		border-bottom: 1px solid #cbd3cd;
		outline: 0;
		background: transparent;
	}
	.ticket-input {
		min-height: 54px;
		resize: vertical;
		font-size: 0.8rem;
	}
	.terminal .button {
		margin-top: 26px;
	}
	.full {
		width: 100%;
	}
	.prefill {
		color: #52715f;
		font-size: 0.75rem;
	}
	.prefill span {
		color: #e75e3f;
	}
	.form-error {
        color: #b44832;
        margin: 5px 0;
        font-size: 1.76rem;
        text-align: center;
	}
	.centered {
		text-align: center;
	}
	.pin-ring,
	.modal-lock {
		color: #e75e3f;
		font-size: 3rem;
	}
	.centered h2 {
		margin: 8px 0;
	}
	.centered > p:not(.terminal-step) {
		color: #78837c;
		font-size: 0.84rem;
	}
	.pin-dots {
		display: flex;
		justify-content: center;
		gap: 10px;
		margin: 25px 0 10px;
	}
	.pin-dots i {
		width: 10px;
		height: 10px;
		border: 1px solid #e75e3f;
		border-radius: 50%;
	}
	.pin-dots i.filled {
		background: #e75e3f;
	}
	.pin-input {
		width: 180px;
		padding: 10px;
		border: 0;
		outline: 0;
		background: transparent;
		text-align: center;
		letter-spacing: 0.45em;
	}
	.terminal-buttons {
		display: flex;
		justify-content: center;
		gap: 12px;
	}
	.success-check {
		width: 58px;
		height: 58px;
		margin: 20px auto;
		display: grid;
		place-content: center;
		border-radius: 50%;
		background: #dce9df;
		color: #4d9566;
		font-size: 2rem;
	}
	.success > strong {
		display: block;
		margin: 18px 0 4px;
		font-size: 2rem;
	}
	.merchant-links {
		display: flex;
		justify-content: center;
		gap: 30px;
		margin: 25px;
	}
	.action-message { margin: 0 auto; color: #52715f; text-align: center; font-size: 0.8rem; }
	.activated {
		padding-top: 18vh;
		text-align: center;
	}
	.activated h1 {
		margin-bottom: 45px;
	}
	.activated .success-check {
		width: 82px;
		height: 82px;
		margin: 0 auto 35px;
	}
	.modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 2;
		display: grid;
		place-items: center;
		padding: 20px;
		background: rgba(23, 33, 29, 0.5);
	}
	.pin-modal {
		width: min(100%, 410px);
		position: relative;
		padding: 40px;
		background: #f3f0e9;
		text-align: center;
	}
	.modal-close {
		position: absolute;
		top: 14px;
		right: 17px;
		border: 0;
		background: transparent;
		font-size: 1.6rem;
	}
	.pin-modal h2 {
		margin: 13px 0 8px;
		font-size: 1.6rem;
	}
	.login-input { width: 100%; margin-top: 14px; padding: 12px 0; border: 0; border-bottom: 1px solid #bdc7be; outline: 0; background: transparent; color: #17211d; }
	.pin-modal > p:not(.eyebrow):not(.form-error) {
		color: #78837c;
		font-size: 0.85rem;
	}
	.demo-hint {
		display: block;
		margin-top: 20px;
		color: #9aa49c;
		font-size: 0.68rem;
	}
	@media (max-width: 620px) {
		.topbar {
			height: 68px;
		}
		.scan-status {
			display: none;
		}
		.onboarding,
		.home,
		.dashboard {
			padding-top: 7vh;
		}
		.step-count {
			margin-bottom: 9vh;
		}
		.welcome-row,
		.dash-heading {
			align-items: flex-start;
		}
		.card-orbit {
			width: 72px;
			height: 72px;
		}
		.card-orbit span {
			font-size: 1.3rem;
		}
		.role-grid,
		.dash-actions {
			grid-template-columns: 1fr;
		}
		.role-card {
			min-height: 170px;
		}
		.balance-card strong {
			font-size: 2.5rem;
		}
		.terminal-body {
			padding: 25px 20px;
		}
		.amount-input input {
			font-size: 2.8rem;
		}
		.pin-modal {
			padding: 32px 22px;
		}
		.merchant-links {
			gap: 12px;
		}
	}
</style>
