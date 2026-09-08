package handlers

import (
	"net/http"

	"construct/oracle/internal/database"
	"construct/oracle/internal/models"
)

func DashboardOverview(w http.ResponseWriter, r *http.Request) {
	if requireAuth(w, r) == nil { return }
	var userCount, adminCount int64
	database.Get(database.DBAccounts).Model(&models.AccountUser{}).Count(&userCount)
	database.Get(database.DBOracle).Model(&models.Administrator{}).Where("active = ?", true).Count(&adminCount)
	WriteJSON(w, 200, map[string]any{"users": userCount, "administrators": adminCount})
}
