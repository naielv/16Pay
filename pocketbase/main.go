package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/naiel/16pay/pocketbase/migrations"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/osutils"
	"golang.org/x/crypto/bcrypt"
)

type pinRequest struct {
	Role string `json:"role"`
	PIN  string `json:"pin"`
}
type merchantLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type onboardRequest struct {
	Name        string `json:"name"`
	CustomerPIN string `json:"customerPin"`
	MerchantPIN string `json:"merchantPin"`
}
type chargeRequest struct {
	MerchantPIN    string          `json:"merchantPin"`
	MerchantToken  string          `json:"merchantToken"`
	Amount         int64           `json:"amount"`
	Concept        string          `json:"concept"`
	Ticket         json.RawMessage `json:"ticket"`
	IdempotencyKey string          `json:"idempotencyKey"`
}
type preconfigRequest struct {
	Ticket json.RawMessage `json:"ticket"`
}

const ferreoCurrency = "FE"

func main() {
	app := pocketbase.New()
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{Automigrate: false, Dir: "migrations"})
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			e.Router.GET("/api/16pay/cards/{id}", cardDetails)
			e.Router.POST("/api/16pay/cards/{id}/onboard", onboardCard)
			e.Router.POST("/api/16pay/cards/{id}/verify-pin", verifyPIN)
			e.Router.POST("/api/16pay/merchants/login", merchantLogin)
			e.Router.POST("/api/16pay/cards/{id}/balance", cardBalance)
			e.Router.POST("/api/16pay/cards/{id}/charge", chargeCard)
			e.Router.POST("/api/16pay/cards/{id}/transactions", listTransactions)
			e.Router.POST("/api/16pay/cards/{id}/transactions/{tx}/refund", refundTransaction)
			e.Router.POST("/api/16pay/preconfigs", createPreconfig)
			e.Router.POST("/api/16pay/preconfigs/{token}/consume", consumePreconfig)
			e.Router.POST("/_/preconfig", createPreconfigCompat)
			publicDir := os.Getenv("PAY_PUBLIC_DIR")
			if publicDir == "" {
				publicDir = defaultPublicDir()
			}
			e.Router.GET("/{path...}", apis.Static(os.DirFS(publicDir), true))
			return e.Next()
		},
		Priority: 999,
	})
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func cardDetails(e *core.RequestEvent) error {
	record, err := e.App.FindRecordById("cards", e.Request.PathValue("id"))
	if err != nil {
		return e.NotFoundError("Card not found", nil)
	}
	return e.JSON(http.StatusOK, map[string]any{"id": record.Id, "name": record.GetString("name"), "status": record.GetString("status"), "currency": record.GetString("currency")})
}

func cardBalance(e *core.RequestEvent) error {
	var body pinRequest
	if err := e.BindBody(&body); err != nil || body.Role != "customer" || !validPIN(body.PIN) {
		return e.BadRequestError("Invalid balance authorization", nil)
	}
	record, err := e.App.FindRecordById("cards", e.Request.PathValue("id"))
	if err != nil {
		return e.NotFoundError("Card not found", nil)
	}
	if record.GetString("currency") != ferreoCurrency {
		return e.Error(http.StatusConflict, "Only Ferreo currency is accepted", nil)
	}
	if !validatePINHash(record.GetString("customerPin"), body.PIN) {
		return e.UnauthorizedError("Incorrect PIN", nil)
	}
	return e.JSON(http.StatusOK, map[string]any{"balance": record.GetInt("balance"), "currency": record.GetString("currency")})
}

func merchantLogin(e *core.RequestEvent) error {
	var body merchantLoginRequest
	if err := e.BindBody(&body); err != nil || strings.TrimSpace(body.Email) == "" || body.Password == "" {
		return e.BadRequestError("Email and password are required", nil)
	}
	merchant, err := e.App.FindFirstRecordByData("merchants", "email", strings.TrimSpace(body.Email))
	if err != nil || !merchant.ValidatePassword(body.Password) {
		return e.UnauthorizedError("Invalid merchant credentials", nil)
	}
	token, err := merchant.NewAuthToken()
	if err != nil {
		return e.InternalServerError("Could not create merchant session", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"token": token, "merchantId": merchant.Id, "name": merchant.GetString("name"), "cardId": merchant.GetString("cardId")})
}

func onboardCard(e *core.RequestEvent) error {
	var body onboardRequest
	if err := e.BindBody(&body); err != nil || !validPIN(body.CustomerPIN) || !validPIN(body.MerchantPIN) {
		return e.BadRequestError("Both PINs must contain four digits", nil)
	}
	record, err := e.App.FindRecordById("cards", e.Request.PathValue("id"))
	if err != nil {
		record = core.NewRecord(mustCollection(e, "cards"))
		record.Id = e.Request.PathValue("id")
	}
	record.Set("name", strings.TrimSpace(body.Name))
	record.Set("status", "active")
	record.Set("currency", ferreoCurrency)
	hCustomer, errC := bcrypt.GenerateFromPassword([]byte(body.CustomerPIN), bcrypt.DefaultCost)
	hMerchant, errM := bcrypt.GenerateFromPassword([]byte(body.MerchantPIN), bcrypt.DefaultCost)
	if errC != nil || errM != nil {
		return e.InternalServerError("Could not encrypt PINs", nil)
	}
	record.Set("customerPin", string(hCustomer))
	record.Set("merchantPin", string(hMerchant))
	if err := e.App.Save(record); err != nil {
		return e.InternalServerError("Could not activate card", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"id": record.Id, "status": "active"})
}

func verifyPIN(e *core.RequestEvent) error {
	var body pinRequest
	if err := e.BindBody(&body); err != nil || (body.Role != "customer" && body.Role != "merchant") || (body.Role == "customer" && !validPIN(body.PIN)) {
		return e.BadRequestError("Invalid PIN request", nil)
	}
	record, err := e.App.FindRecordById("cards", e.Request.PathValue("id"))
	if err != nil {
		return e.NotFoundError("Card not found", nil)
	}
	if time.Now().Before(parseTime(record.GetString("lockedUntil"))) {
		return e.TooManyRequestsError("PIN temporarily locked", nil)
	}
	field := "customerPin"
	if body.Role == "merchant" {
		field = "merchantPin"
	}
	if !validatePINHash(record.GetString(field), body.PIN) {
		attempts := record.GetInt("failedAttempts") + 1
		record.Set("failedAttempts", attempts)
		if attempts >= 5 {
			record.Set("lockedUntil", time.Now().Add(5*time.Minute).UTC().Format(time.RFC3339))
			record.Set("failedAttempts", 0)
		}
		_ = e.App.Save(record)
		return e.UnauthorizedError("Incorrect PIN", nil)
	}
	record.Set("failedAttempts", 0)
	record.Set("lockedUntil", "")
	_ = e.App.Save(record)
	return e.JSON(http.StatusOK, map[string]any{"verified": true, "expiresIn": 120})
}

func chargeCard(e *core.RequestEvent) error {
	var body chargeRequest
	if err := e.BindBody(&body); err != nil || body.Amount <= 0 || body.Amount > 10000000 || strings.TrimSpace(body.Concept) == "" || len(body.IdempotencyKey) < 8 {
		return e.BadRequestError("Cobro invalido", nil)
	}
	card, err := e.App.FindRecordById("cards", e.Request.PathValue("id"))
	if err != nil {
		return e.NotFoundError("Tarjeta del cliente no encontrada", nil)
	}
	if card.GetString("currency") != ferreoCurrency {
		return e.Error(http.StatusConflict, "La tarjeta del cliente no acepta Ferreo", nil)
	}
	merchant, err := merchantFromToken(e, body.MerchantToken)
	if err != nil {
		return e.UnauthorizedError("Cuenta de comercio necesaria", nil)
	}
	merchantWallet, err := e.App.FindRecordById("cards", merchant.GetString("cardId"))
	if err != nil || merchantWallet.GetString("currency") != ferreoCurrency {
		return e.Error(http.StatusConflict, "La tarjeta del comercio no acepta Ferreo", nil)
	}
	if !validatePINHash(card.GetString("merchantPin"), body.MerchantPIN) {
		return e.UnauthorizedError("PIN incorrecto", nil)
	}
	if _, err := e.App.FindFirstRecordByData("transactions", "idempotencyKey", body.IdempotencyKey); err == nil {
		return e.JSON(http.StatusOK, map[string]any{"status": "approved", "duplicate": true})
	}
	if card.GetInt("balance") < int(body.Amount) {
		return e.Error(http.StatusConflict, "Saldo insuficiente", nil)
	}
	tx := core.NewRecord(mustCollection(e, "transactions"))
	tx.Set("card", card.Id)
	tx.Set("amount", body.Amount)
	tx.Set("concept", strings.TrimSpace(body.Concept))
	tx.Set("ticket", body.Ticket)
	tx.Set("status", "approved")
	tx.Set("merchantName", merchant.GetString("name"))
	tx.Set("merchantCard", merchantWallet.Id)
	tx.Set("idempotencyKey", body.IdempotencyKey)
	card.Set("balance", card.GetInt("balance")-int(body.Amount))
	merchantWallet.Set("balance", merchantWallet.GetInt("balance")+int(body.Amount))
	if err := e.App.Save(tx); err != nil {
		return e.InternalServerError("No se ha podido guardar el movimiento al cliente", err)
	}
	if err := e.App.Save(card); err != nil {
		return e.InternalServerError("No se ha podido actualizar saldo del cliente", err)
	}
	txc := core.NewRecord(mustCollection(e, "transactions"))
	txc.Set("card", merchantWallet.Id)
	txc.Set("amount", -body.Amount)
	txc.Set("concept", strings.TrimSpace(body.Concept))
	txc.Set("ticket", body.Ticket)
	txc.Set("status", "approved")
	txc.Set("merchantName", card.GetString("name"))
	txc.Set("merchantCard", card.Id)
	txc.Set("idempotencyKey", body.IdempotencyKey)
	if err := e.App.Save(txc); err != nil {
		return e.InternalServerError("No se ha podido guardar el movimiento al comercio", err)
	}
	if err := e.App.Save(merchantWallet); err != nil {
		return e.InternalServerError("No se ha podido actualizar saldo del comercio", err)
	}
	return e.JSON(http.StatusCreated, map[string]any{"id": tx.Id, "status": "approved", "amount": body.Amount})
}

func listTransactions(e *core.RequestEvent) error {
	var body pinRequest
	if err := e.BindBody(&body); err != nil || (body.Role != "customer" && body.Role != "merchant") || (body.Role == "customer" && !validPIN(body.PIN)) {
		return e.BadRequestError("Invalid history authorization", nil)
	}
	if _, err := e.App.FindRecordById("cards", e.Request.PathValue("id")); err != nil {
		return e.NotFoundError("Tarjeta no encontrada", nil)
	}
	card, err := e.App.FindRecordById("cards", e.Request.PathValue("id"))
	var filter string
	var params dbx.Params
	if body.Role == "merchant" {
		merchant, err := merchantFromToken(e, e.Request.Header.Get("Authorization"))
		if err != nil {
			return e.UnauthorizedError("Cuenta de comercio necesaria", nil)
		}
		filter = "merchantCard = {:merchantCard}"
		params = dbx.Params{"merchantCard": merchant.GetString("cardId")}
	} else {
		if !validatePINHash(card.GetString("customerPin"), body.PIN) {
			return e.UnauthorizedError("PIN incorrecto", nil)
		}
		filter = "card = {:card}"
		params = dbx.Params{"card": e.Request.PathValue("id")}
	}
	records, err := e.App.FindRecordsByFilter("transactions", filter, "-created", 100, 0, params)
	if err != nil {
		return e.InternalServerError("Could not load transactions", err)
	}
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		result = append(result, map[string]any{"id": record.Id, "tid": record.Id, "amount": -record.GetInt("amount") / 100.0, "concept": record.GetString("concept"), "status": record.GetString("status"), "merchant": record.GetString("merchantName"), "ticket": record.Get("ticket"), "date": record.GetString("created")})
	}
	return e.JSON(http.StatusOK, result)
}

func refundTransaction(e *core.RequestEvent) error {
	var body pinRequest
	if err := e.BindBody(&body); err != nil || body.Role != "merchant" {
		return e.BadRequestError("Invalid refund authorization", nil)
	}
	tx, err := e.App.FindRecordById("transactions", e.Request.PathValue("tx"))
	if err != nil {
		return e.NotFoundError("Transaction not found", nil)
	}
	card, err := e.App.FindRecordById("cards", tx.GetString("card"))
	if err != nil {
		return e.NotFoundError("Card not found", nil)
	}
	merchant, err := merchantFromToken(e, e.Request.Header.Get("Authorization"))
	if err != nil || tx.GetString("merchantCard") != merchant.GetString("cardId") {
		return e.UnauthorizedError("Merchant authentication required", nil)
	}
	merchantWallet, err := e.App.FindRecordById("cards", merchant.GetString("cardId"))
	if err != nil {
		return e.NotFoundError("Merchant wallet not found", nil)
	}
	if tx.GetString("status") != "approved" {
		return e.Error(http.StatusConflict, "Transaction cannot be refunded", nil)
	}
	if merchantWallet.GetInt("balance") < tx.GetInt("amount") {
		return e.Error(http.StatusConflict, "Insufficient merchant balance to refund", nil)
	}
	tx.Set("status", "refunded")
	card.Set("balance", card.GetInt("balance")+tx.GetInt("amount"))
	merchantWallet.Set("balance", merchantWallet.GetInt("balance")-tx.GetInt("amount"))
	if err := e.App.Save(tx); err != nil {
		return e.InternalServerError("Could not refund transaction", err)
	}
	if err := e.App.Save(card); err != nil {
		return e.InternalServerError("Could not update balance", err)
	}
	if err := e.App.Save(merchantWallet); err != nil {
		return e.InternalServerError("Could not update merchant balance", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"id": tx.Id, "status": "refunded", "merchantName": merchant.GetString("name")})
}

func merchantFromToken(e *core.RequestEvent, rawToken string) (*core.Record, error) {
	token := strings.TrimSpace(strings.TrimPrefix(rawToken, "Bearer "))
	if token == "" {
		return nil, errors.New("missing merchant token")
	}
	record, err := e.App.FindAuthRecordByToken(token, "auth")
	if err != nil || record.Collection().Name != "merchants" {
		return nil, errors.New("invalid merchant token")
	}
	return record, nil
}

func createPreconfig(e *core.RequestEvent) error {
	var body struct {
		Amount  int64           `json:"amount"`
		Concept string          `json:"concept"`
		Ticket  json.RawMessage `json:"ticket"`
	}
	if err := e.BindBody(&body); err != nil || body.Amount <= 0 || strings.TrimSpace(body.Concept) == "" {
		return e.BadRequestError("Invalid preconfiguration", nil)
	}
	token := randomToken()
	record := core.NewRecord(mustCollection(e, "preconfigs"))
	record.Set("token", token)
	record.Set("amount", body.Amount)
	record.Set("concept", strings.TrimSpace(body.Concept))
	record.Set("ticket", body.Ticket)
	record.Set("expiresAt", time.Now().Add(10*time.Minute).UTC().Format(time.RFC3339))
	record.Set("status", "pending")
	if err := e.App.Save(record); err != nil {
		return e.InternalServerError("Could not save preconfiguration", err)
	}
	return e.JSON(http.StatusCreated, map[string]any{"token": token, "status": "pending_scan", "expiresAt": record.GetString("expiresAt")})
}

func createPreconfigCompat(e *core.RequestEvent) error {
	var body preconfigRequest
	if err := e.BindBody(&body); err != nil && e.Request.ContentLength > 0 {
		return e.BadRequestError("Invalid ticket JSON", nil)
	}
	amountText := strings.ReplaceAll(e.Request.URL.Query().Get("amount"), ",", ".")
	var euros float64
	if _, err := fmt.Sscanf(amountText, "%f", &euros); err != nil || euros <= 0 {
		return e.BadRequestError("Invalid amount", nil)
	}
	concept := strings.TrimSpace(e.Request.URL.Query().Get("concept"))
	if concept == "" {
		return e.BadRequestError("Invalid concept", nil)
	}
	token := randomToken()
	record := core.NewRecord(mustCollection(e, "preconfigs"))
	record.Set("token", token)
	record.Set("amount", int64(euros*100))
	record.Set("concept", concept)
	record.Set("ticket", body.Ticket)
	record.Set("expiresAt", time.Now().Add(10*time.Minute).UTC().Format(time.RFC3339))
	record.Set("status", "pending")
	if err := e.App.Save(record); err != nil {
		return e.InternalServerError("Could not save preconfiguration", err)
	}
	return e.JSON(http.StatusCreated, map[string]any{"token": token, "status": "pending_scan", "expiresAt": record.GetString("expiresAt")})
}

func consumePreconfig(e *core.RequestEvent) error {
	record, err := e.App.FindFirstRecordByData("preconfigs", "token", e.Request.PathValue("token"))
	if err != nil {
		return e.NotFoundError("Preconfiguration not found", nil)
	}
	if record.GetString("status") != "pending" || time.Now().After(parseTime(record.GetString("expiresAt"))) {
		return e.Error(http.StatusGone, "Preconfiguration expired or consumed", nil)
	}
	record.Set("status", "consumed")
	if err := e.App.Save(record); err != nil {
		return e.InternalServerError("Could not consume preconfiguration", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"amount": record.GetInt("amount"), "concept": record.GetString("concept"), "ticket": record.Get("ticket")})
}

func mustCollection(e *core.RequestEvent, name string) *core.Collection {
	collection, err := e.App.FindCollectionByNameOrId(name)
	if err != nil {
		panic(err)
	}
	return collection
}
func validPIN(pin string) bool {
	return len(pin) == 4 && pin[0] >= '0' && pin[0] <= '9' && pin[1] >= '0' && pin[1] <= '9' && pin[2] >= '0' && pin[2] <= '9' && pin[3] >= '0' && pin[3] <= '9'
}

func validatePINHash(hash string, pin string) bool {
	return hash != "" && bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin)) == nil
}
func randomToken() string {
	bytes := make([]byte, 24)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
func parseTime(value string) time.Time { parsed, _ := time.Parse(time.RFC3339, value); return parsed }
func defaultPublicDir() string {
	if osutils.IsProbablyGoRun() {
		return "../build"
	}
	return filepath.Join(filepath.Dir(os.Args[0]), "pb_public")
}

var _ = errors.New
