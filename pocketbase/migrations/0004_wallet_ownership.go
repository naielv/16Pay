package migrations

import (
	"github.com/pocketbase/pocketbase/core"
)

func init() {
	core.AppMigrations.Register(upWalletOwnership, downWalletOwnership, "0004_wallet_ownership.go")
}

func upWalletOwnership(app core.App) error {
	cards, err := app.FindCollectionByNameOrId("cards")
	if err != nil {
		return err
	}
	merchants, err := app.FindCollectionByNameOrId("merchants")
	if err != nil {
		return err
	}
	if merchants.Fields.GetByName("cardId") == nil {
		merchants.Fields.Add(&core.RelationField{Name: "cardId", CollectionId: cards.Id, MaxSelect: 1})
		if err := app.Save(merchants); err != nil {
			return err
		}
	}
	if old := cards.Fields.GetByName("merchantId"); old != nil {
		cards.Fields.RemoveByName(old.GetName())
		if err := app.Save(cards); err != nil {
			return err
		}
	}
	merchant, err := app.FindRecordById("merchants", "merchant0000000")
	if err == nil && merchant.GetString("cardId") == "" {
		merchant.Set("cardId", "demo00000000000")
		if err := app.Save(merchant); err != nil {
			return err
		}
	}
	transactions, err := app.FindCollectionByNameOrId("transactions")
	if err != nil {
		return err
	}
	if transactions.Fields.GetByName("merchantCard") == nil {
		transactions.Fields.Add(&core.TextField{Name: "merchantCard", Required: true, Max: 15})
		if err := app.Save(transactions); err != nil {
			return err
		}
	}
	return nil
}

func downWalletOwnership(app core.App) error { return nil }
