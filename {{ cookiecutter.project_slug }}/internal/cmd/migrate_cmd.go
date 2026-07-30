package cmd

// MigrateCmd applies pending migrations and exits.
//
// `serve` already migrates on boot, so this is for the times you want the
// schema moved without starting the server — checking a migration applies
// cleanly, or running it as a separate step in a deployment.
type MigrateCmd struct {
}

// Run relies on NewApp: bringing the app up is what applies the migrations, so
// there is nothing to duplicate here.
func (m *MigrateCmd) Run() error {
	app, err := NewApp()
	if err != nil {
		return err
	}
	defer app.Close()

	app.Logger.Info("migrations applied")
	return nil
}
