<svelte:head>
	<title>16Pay | Tu tarjeta, en un toque</title>
	<meta name="description" content="Una pasarela NFC sencilla para clientes y comercios." />
</svelte:head>

<main class="entry-shell">
	<div class="entry-mark"><span>16</span><small>PAY</small></div>
	<section class="entry-copy">
		<p class="eyebrow">NFC PAYMENT GATEWAY</p>
		<h1>Tu tarjeta.<br /><em>Un toque.</em></h1>
		<p class="intro">
			Escanea una tarjeta para empezar. Cobra, consulta tu saldo o prepara una operación desde
			cualquier dispositivo.
		</p>
	</section>
	<div class="entry-footer"><span>Seguro por diseño</span><span>Fe · Ferreo</span></div>
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
	.entry-shell {
		min-height: 100vh;
		padding: 34px clamp(24px, 6vw, 96px);
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		background:
			radial-gradient(circle at 78% 30%, rgba(205, 230, 211, 0.72), transparent 32%), #f3f0e9;
	}
	.entry-mark {
		display: flex;
		align-items: center;
		gap: 8px;
		color: #e75e3f;
		font-weight: 800;
		letter-spacing: -0.08em;
	}
	.entry-mark span {
		font-size: 2rem;
	}
	.entry-mark small {
		align-self: flex-end;
		padding-bottom: 4px;
		font-size: 0.65rem;
		letter-spacing: 0.16em;
	}
	.entry-copy {
		max-width: 740px;
		margin: 8vh 0;
	}
	.eyebrow {
		color: #52715f;
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.18em;
	}
	h1 {
		margin: 18px 0;
		font-size: clamp(4.3rem, 12vw, 9.5rem);
		line-height: 0.82;
		letter-spacing: -0.09em;
		font-weight: 500;
	}
	h1 em {
		color: #e75e3f;
		font-family: Georgia, serif;
		font-weight: 400;
		letter-spacing: -0.08em;
	}
	.intro {
		max-width: 410px;
		margin: 34px 0;
		color: #617068;
		font-size: 1.05rem;
		line-height: 1.55;
	}
	.primary-button {
		display: inline-flex;
		align-items: center;
		gap: 28px;
		padding: 16px 19px 16px 22px;
		border-radius: 2px;
		background: #17211d;
		color: #fff;
		text-decoration: none;
		font-size: 0.9rem;
		font-weight: 700;
	}
	.primary-button span {
		color: #e75e3f;
		font-size: 1.2rem;
	}
	.entry-footer {
		display: flex;
		justify-content: space-between;
		color: #849087;
		font-size: 0.72rem;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	@media (max-width: 600px) {
		.entry-shell {
			padding: 26px 22px;
		}
		.entry-copy {
			margin: 12vh 0;
		}
		.entry-footer {
			font-size: 0.62rem;
		}
	}
</style>
