package migrations

import "github.com/pocketbase/pocketbase/core"

func init() {
	core.AppMigrations.Register(upFerreoCurrency, downFerreoCurrency, "0003_ferreo_currency.go")
}

func upFerreoCurrency(app core.App) error {
	records, err := app.FindAllRecords("cards")
	if err != nil {
		return err
	}
	for _, record := range records {
		record.Set("currency", "FE")
		if err := app.Save(record); err != nil {
			return err
		}
	}
	return nil
}

func downFerreoCurrency(app core.App) error { return nil }
