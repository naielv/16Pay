package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() { core.AppMigrations.Register(upMerchantAuth, downMerchantAuth, "0002_merchant_auth.go") }

func upMerchantAuth(app core.App) error {
	merchantCollection, err := app.FindCollectionByNameOrId("merchants")
	if err != nil {
		merchantCollection = core.NewAuthCollection("merchants")
		merchantCollection.ListRule = types.Pointer("")
		merchantCollection.ViewRule = types.Pointer("")
		merchantCollection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 120})
		if err := app.Save(merchantCollection); err != nil {
			return err
		}
	}
	if _, err := app.FindRecordById("merchants", "merchant0000000"); err != nil {
		merchant := core.NewRecord(merchantCollection)
		merchant.Id = "merchant0000000"
		merchant.Set("email", "demo@16pay.local")
		merchant.Set("password", "demo12345")
		merchant.Set("passwordConfirm", "demo12345")
		merchant.Set("name", "Café Norte")
		merchant.SetVerified(true)
		if err := app.Save(merchant); err != nil {
			return err
		}
	}

	cards, err := app.FindCollectionByNameOrId("cards")
	if err != nil {
		return err
	}
	if merchantCollection.Fields.GetByName("cardId") == nil {
		merchantCollection.Fields.Add(&core.RelationField{Name: "cardId", CollectionId: cards.Id, MaxSelect: 1})
		if err := app.Save(merchantCollection); err != nil {
			return err
		}
	}
	if merchant, err := app.FindRecordById("merchants", "merchant0000000"); err == nil && merchant.GetString("cardId") == "" {
		merchant.Set("cardId", "demo00000000000")
		if err := app.Save(merchant); err != nil {
			return err
		}
	}
	if cards.Fields.GetByName("merchantId") != nil {
		cards.Fields.RemoveByName("merchantId")
		if err := app.Save(cards); err != nil {
			return err
		}
	}

	transactions, err := app.FindCollectionByNameOrId("transactions")
	if err != nil {
		return err
	}
	if transactions.Fields.GetByName("merchantName") == nil {
		transactions.Fields.Add(&core.TextField{Name: "merchantName", Required: true, Max: 120})
		if err := app.Save(transactions); err != nil {
			return err
		}
	}
	if transactions.Fields.GetByName("merchantCard") == nil {
		transactions.Fields.Add(&core.TextField{Name: "merchantCard", Required: true, Max: 15})
		if err := app.Save(transactions); err != nil {
			return err
		}
	}
	if transactions.Fields.GetByName("created") == nil {
		transactions.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		transactions.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		if err := app.Save(transactions); err != nil {
			return err
		}
	}
	return nil
}

func downMerchantAuth(app core.App) error { return nil }
