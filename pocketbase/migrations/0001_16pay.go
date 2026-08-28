package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/crypto/bcrypt"
)

func init() { core.AppMigrations.Register(up, down, "0001_16pay.go") }

func up(app core.App) error {
	if err := addCards(app); err != nil {
		return err
	}
	if err := addMerchants(app); err != nil {
		return err
	}
	if err := addTransactions(app); err != nil {
		return err
	}
	return addPreconfigs(app)
}

func down(app core.App) error {
	for _, name := range []string{"preconfigs", "transactions", "cards", "merchants"} {
		if collection, err := app.FindCollectionByNameOrId(name); err == nil {
			if err := app.Delete(collection); err != nil {
				return err
			}
		}
	}
	return nil
}

func addMerchants(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("merchants"); err == nil {
		return nil
	}
	collection := core.NewAuthCollection("merchants")
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 120})
	cardCollection, err := app.FindCollectionByNameOrId("cards")
	if err != nil {
		return err
	}
	collection.Fields.Add(&core.RelationField{Name: "cardId", CollectionId: cardCollection.Id, MaxSelect: 1})
	if err := app.Save(collection); err != nil {
		return err
	}
	merchant := core.NewRecord(collection)
	merchant.Id = "merchant0000000"
	merchant.Set("email", "demo@16pay.local")
	merchant.Set("password", "demo12345")
	merchant.Set("passwordConfirm", "demo12345")
	merchant.Set("name", "Café Norte")
	merchant.Set("cardId", "demo00000000000")
	merchant.SetVerified(true)
	return app.Save(merchant)
}

func addCards(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("cards"); err == nil {
		return nil
	}
	collection := core.NewBaseCollection("cards")
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 120})
	collection.Fields.Add(&core.TextField{Name: "status", Required: true})
	collection.Fields.Add(&core.TextField{Name: "currency", Required: true})
	collection.Fields.Add(&core.NumberField{Name: "balance", Required: true})
	collection.Fields.Add(&core.TextField{Name: "customerPin", Required: false, Max: 120})
	collection.Fields.Add(&core.TextField{Name: "merchantPin", Required: false, Max: 120})
	collection.Fields.Add(&core.NumberField{Name: "failedAttempts"})
	collection.Fields.Add(&core.TextField{Name: "lockedUntil"})
	if err := app.Save(collection); err != nil {
		return err
	}
	if _, err := app.FindRecordById("cards", "demo00000000000"); err != nil {
		record := core.NewRecord(collection)
		record.Id = "demo00000000000"
		record.Set("name", "Cuenta Demo")
		record.Set("status", "active")
		record.Set("currency", "FE")
		record.Set("balance", 124050)
		// store bcrypt hashes directly
		hCustomer := hashPIN("2468")
		hMerchant := hashPIN("1357")
		record.Set("customerPin", hCustomer)
		record.Set("merchantPin", hMerchant)
		return app.Save(record)
	}
	return nil
}

func addTransactions(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("transactions"); err == nil {
		return nil
	}
	collection := core.NewBaseCollection("transactions")
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.Fields.Add(&core.TextField{Name: "card", Required: true})
	collection.Fields.Add(&core.NumberField{Name: "amount", Required: true})
	collection.Fields.Add(&core.TextField{Name: "concept", Required: true, Max: 120})
	collection.Fields.Add(&core.JSONField{Name: "ticket"})
	collection.Fields.Add(&core.TextField{Name: "status", Required: true})
	collection.Fields.Add(&core.TextField{Name: "idempotencyKey", Required: true, Max: 120})
	collection.Fields.Add(&core.TextField{Name: "merchantName", Required: true, Max: 120})
	collection.Fields.Add(&core.TextField{Name: "merchantCard", Required: true, Max: 15})
	collection.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	collection.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	collection.AddIndex("idx_transactions_idempotency", true, "idempotencyKey", "")
	return app.Save(collection)
}

func addPreconfigs(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("preconfigs"); err == nil {
		return nil
	}
	collection := core.NewBaseCollection("preconfigs")
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.Fields.Add(&core.TextField{Name: "token", Required: true, Max: 80})
	collection.Fields.Add(&core.NumberField{Name: "amount", Required: true})
	collection.Fields.Add(&core.TextField{Name: "concept", Required: true, Max: 120})
	collection.Fields.Add(&core.JSONField{Name: "ticket"})
	collection.Fields.Add(&core.TextField{Name: "expiresAt", Required: true})
	collection.Fields.Add(&core.TextField{Name: "status", Required: true})
	collection.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	collection.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	collection.AddIndex("idx_preconfigs_token", true, "token", "")
	return app.Save(collection)
}

func hashPIN(pin string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}
