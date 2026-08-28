<script lang="ts">
	import { page } from '$app/state';
	import { parseAmount, savePreconfig } from '$lib/payments';

	let amount = $state(page.url.searchParams.get('amount') ?? '');
	let concept = $state(page.url.searchParams.get('concept') ?? '');
	let ticket = $state('');
	let error = $state('');
	let token = $state('');
	let expiresAt = $state('');
	let saving = $state(false);

	async function prepare() {
		const parsed = parseAmount(amount);
		if (!parsed || !concept.trim()) {
			error = 'Revisa el importe y añade un concepto.';
			return;
		}
		saving = true;
		error = '';
		try {
			const response = await fetch('/_/preconfig', {
				method: 'POST',
				headers: { 'content-type': 'application/json' },
				body: JSON.stringify({ ticket: ticket.trim() || null })
			});
			const result = await response.json();
			if (!response.ok) throw new Error(result.message ?? 'No se pudo preparar el cobro.');
			token = result.token;
			expiresAt = result.expiresAt;
			savePreconfig({
				token,
				amount: parsed,
				concept: concept.trim(),
				ticket: ticket.trim() || null,
				expiresAt: Date.parse(expiresAt)
			});
		} catch (requestError) {
			error =
				requestError instanceof Error ? requestError.message : 'No se pudo preparar el cobro.';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head><title>Preparar cobro | 16Pay</title></svelte:head>

<main class="preconfig-shell">
	<a class="brand" href="/">16<span>PAY</span></a>
	{#if token}
		<section class="ready">
			<span class="ready-icon">✓</span>
			<p class="eyebrow">PRECONFIGURACIÓN LISTA</p>
			<h1>Ahora,<br /><em>acerca la tarjeta.</em></h1>
			<p>El siguiente escaneo abrirá el datáfono con los datos ya preparados.</p>
			<div class="summary"><span>{concept}</span><strong>{amount.replace('.', ',')} Fe</strong></div>
			<small
				>Válida hasta {new Date(expiresAt).toLocaleTimeString('es-ES', {
					hour: '2-digit',
					minute: '2-digit'
				})}</small
			><a class="button dark" href={`/demo00000000000/?preconfig=${token}`}>Simular escaneo <span>→</span></a><code>{token}</code>
		</section>
	{:else}
		<section class="form-wrap">
			<p class="eyebrow">INTEGRACIÓN · PRECONFIG</p>
			<h1>Prepara el<br /><em>próximo cobro.</em></h1>
			<p class="intro">
				Los datos quedarán listos durante 10 minutos. El cobro sólo ocurrirá después de escanear una
				tarjeta y confirmar con PIN.
			</p>
			<form
				onsubmit={(event) => {
					event.preventDefault();
					prepare();
				}}
			>
				<label
					>Importe
					<div class="money-input">
						<span>Fe</span><input
							bind:value={amount}
							inputmode="decimal"
							placeholder="0,00"
							aria-label="Importe"
						/>
					</div></label
				><label>Concepto<input bind:value={concept} placeholder="Ej. Pedido #1048" /></label><label
					>Ticket <span class="optional">opcional</span><textarea
						bind:value={ticket}
						placeholder="Texto plano o JSON"></textarea></label
				>{#if error}<p class="form-error">{error}</p>{/if}<button
					class="button dark"
					disabled={saving}>{saving ? 'Preparando...' : 'Esperar escaneo'} <span>→</span></button
				>
			</form>
		</section>
	{/if}
	<footer>16Pay · Preparación segura · Sin cobro automático</footer>
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
	.preconfig-shell {
		min-height: 100vh;
		padding: 34px clamp(22px, 6vw, 90px);
		display: flex;
		flex-direction: column;
		background:
			radial-gradient(circle at 84% 30%, rgba(205, 230, 211, 0.7), transparent 30%), #f3f0e9;
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
	.form-wrap,
	.ready {
		width: min(100%, 610px);
		margin: auto 0;
		padding: 8vh 0;
	}
	.eyebrow {
		color: #52715f;
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.16em;
	}
	.form-wrap h1,
	.ready h1 {
		margin: 22px 0;
		font-size: clamp(3.8rem, 8vw, 7rem);
		line-height: 0.84;
		letter-spacing: -0.095em;
		font-weight: 500;
	}
	.form-wrap h1 em,
	.ready h1 em {
		color: #e75e3f;
		font-family: Georgia, serif;
		font-weight: 400;
	}
	.intro,
	.ready > p:not(.eyebrow) {
		max-width: 420px;
		color: #69756d;
		line-height: 1.5;
	}
	form {
		margin-top: 42px;
	}
	label {
		display: block;
		margin-top: 21px;
		color: #52715f;
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}
	input,
	textarea {
		width: 100%;
		padding: 13px 0;
		border: 0;
		border-bottom: 1px solid #aebbb1;
		outline: 0;
		background: transparent;
		color: #17211d;
		font-size: 1rem;
		letter-spacing: 0;
	}
	textarea {
		min-height: 70px;
		resize: vertical;
		font-size: 0.85rem;
	}
	.money-input {
		display: flex;
		align-items: baseline;
		border-bottom: 2px solid #17211d;
	}
	.money-input span {
		color: #e75e3f;
		font-size: 1.4rem;
	}
	.money-input input {
		border: 0;
		font-size: 2.8rem;
		letter-spacing: -0.07em;
	}
	.optional {
		color: #96a199;
		font-size: 0.62rem;
		letter-spacing: 0;
		text-transform: none;
	}
	.button {
		display: inline-flex;
		align-items: center;
		justify-content: space-between;
		gap: 28px;
		margin-top: 30px;
		padding: 15px 20px;
		border: 0;
		border-radius: 2px;
		font-weight: 700;
		text-decoration: none;
	}
	.button.dark {
		background: #17211d;
		color: #fff;
	}
	.button:disabled {
		opacity: 0.45;
	}
	.form-error {
		color: #b44832;
		font-size: 0.8rem;
	}
	.ready-icon {
		width: 58px;
		height: 58px;
		display: grid;
		place-content: center;
		border-radius: 50%;
		background: #dce9df;
		color: #4d9566;
		font-size: 2rem;
	}
	.summary {
		max-width: 510px;
		margin-top: 42px;
		padding: 18px 0;
		display: flex;
		justify-content: space-between;
		border-top: 1px solid #bdc7be;
		border-bottom: 1px solid #bdc7be;
	}
	.summary strong {
		font-size: 1.5rem;
	}
	.ready small {
		display: block;
		margin-top: 12px;
		color: #869188;
	}
	.ready code {
		display: block;
		margin-top: 20px;
		color: #9aa49c;
		font-size: 0.65rem;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	footer {
		margin-top: auto;
		color: #89938b;
		font-size: 0.68rem;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	@media (max-width: 600px) {
		.preconfig-shell {
			padding: 26px 22px;
		}
		.form-wrap,
		.ready {
			padding: 8vh 0;
		}
		.summary {
			gap: 12px;
		}
		.summary strong {
			font-size: 1.2rem;
		}
	}
</style>
