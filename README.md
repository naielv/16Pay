# 16Pay

Pasarela NFC para una tarjeta por URL. La única moneda aceptada es Ferreo (símbolo `Fe`). SvelteKit se compila como frontend estático y PocketBase sirve tanto la interfaz como la API.

## Flujo

- `/demo00000000000/` abre una tarjeta activa de demostración.
- Cualquier otro identificador en `/<card>/` inicia el onboarding de una tarjeta nueva.
- El escaneo inicial sólo permite elegir `Soy el cliente` o `Soy el comercio`.
- Las acciones posteriores solicitan PIN. La demo usa `2468` para cliente y `1357` para comercio.
- El comercio tiene un datáfono en dos pasos: importe/concepto y PIN del cliente, seguido de la autorización del comercio.
- El estado de demo se guarda en `localStorage`; no debe usarse para dinero real.

## Preconfiguración

Una integración puede preparar un cobro antes del escaneo:

```http
POST /_/preconfig?amount=12.50&concept=Pedido%201048
Content-Type: application/json

{"ticket":{"order":"1048","items":[{"name":"Café","amount":12.5}]}}
```

La respuesta contiene un token opaco y una expiración de 10 minutos:

```json
{ "token": "...", "status": "pending_scan", "expiresAt": "2026-08-28T12:00:00.000Z" }
```

La pantalla `/​_/preconfig?amount=X&concept=Y` ofrece el mismo flujo desde navegador. Al simular el siguiente escaneo en `/demo00000000000/`, el importe, concepto y ticket se precargan en el datáfono. No se cobra automáticamente.

## PocketBase

PocketBase ya contiene migraciones y endpoints Go para las operaciones sensibles:

- `cards`: ID, estado, moneda, saldo y hashes de PIN separados por rol.
- `transactions`: tarjeta, importe en céntimos, concepto, ticket, estado e idempotencia.
- `preconfigs`: token aleatorio, datos del cobro, expiración y consumo único.
- control de intentos y bloqueo temporal de PIN.

Los PIN se guardan como hashes bcrypt y no viajan en URLs ni respuestas. La tarjeta demo se crea automáticamente en una base vacía con PIN cliente `2468` y comercio `1357`.

## Desarrollo

```sh
npm install
npm run dev
```

Para ejecutar la aplicación completa con PocketBase:

```sh
npm run start:server
```

El puerto por defecto es `8090`. Puedes cambiarlo con `PORT=8091 npm run start:server` y la carpeta de datos con `PB_DATA_DIR=/ruta/de/datos npm run start:server`.

El build deja los archivos en `pocketbase/pb_public`. Las migraciones crean las colecciones y la tarjeta demo automáticamente. La primera ejecución de PocketBase muestra la URL para crear el superusuario del dashboard.

Validaciones disponibles:

```sh
npm run check
npm run lint
npm run build
go -C pocketbase test ./...
```
