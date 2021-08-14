package plugin

import (
	"time"
)

type backendRole struct {
	UserName          string    `json:"user_name"`
	TTL               int       `json:"ttl"`
	LastVaultRotation time.Time `json:"last_vault_rotation"`
}

func (r *backendRole) Map() map[string]interface{} {
	m := map[string]interface{}{
		"user_name": r.UserName,
		"ttl":       r.TTL,
	}

	var unset time.Time
	if r.LastVaultRotation != unset {
		m["last_vault_rotation"] = r.LastVaultRotation
	}
	return m
}
