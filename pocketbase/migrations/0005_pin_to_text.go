package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func init() { core.AppMigrations.Register(upPinToText, downPinToText, "0005_pin_to_text.go") }

func upPinToText(app core.App) error {
	cards, err := app.FindCollectionByNameOrId("cards")
	if err != nil {
		return err
	}

	// 1. Read old hashes before modifying schema
	records, err := app.FindRecordsByFilter("cards", "1=1", "", 9999, 0, dbx.Params{})
	if err != nil {
		return err
	}
	type hashEntry struct{ customer, merchant string }
	old := make(map[string]hashEntry, len(records))
	for _, r := range records {
		old[r.Id] = hashEntry{
			customer: r.GetString("customerPin:hash"),
			merchant: r.GetString("merchantPin:hash"),
		}
	}

	// 2. Remove old PasswordFields
	if f := cards.Fields.GetByName("customerPin"); f != nil {
		cards.Fields.RemoveByName(f.GetName())
	}
	if f := cards.Fields.GetByName("merchantPin"); f != nil {
		cards.Fields.RemoveByName(f.GetName())
	}

	// 3. Add new TextFields (same names, now stores bcrypt hash directly)
	cards.Fields.Add(&core.TextField{Name: "customerPin", Required: false, Max: 120})
	cards.Fields.Add(&core.TextField{Name: "merchantPin", Required: false, Max: 120})

	if err := app.Save(cards); err != nil {
		return err
	}

	// 4. Write back the original bcrypt hashes into the new TextFields
	for _, r := range records {
		pair := old[r.Id]
		if pair.customer != "" {
			r.Set("customerPin", pair.customer)
		}
		if pair.merchant != "" {
			r.Set("merchantPin", pair.merchant)
		}
		if err := app.Save(r); err != nil {
			return err
		}
	}

	return nil
}

func downPinToText(app core.App) error {
	cards, err := app.FindCollectionByNameOrId("cards")
	if err != nil {
		return err
	}
	if f := cards.Fields.GetByName("customerPin"); f != nil {
		cards.Fields.RemoveByName(f.GetName())
	}
	if f := cards.Fields.GetByName("merchantPin"); f != nil {
		cards.Fields.RemoveByName(f.GetName())
	}
	cards.Fields.Add(&core.PasswordField{Name: "customerPin", Hidden: false, Min: 4, Max: 4})
	cards.Fields.Add(&core.PasswordField{Name: "merchantPin", Hidden: false, Min: 4, Max: 4})
	return app.Save(cards)
}
