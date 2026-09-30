// Package web embarque les pages et scripts servis par le module et l'assistant.
package web

import _ "embed"

// BannerJS : bandeau de décompte injecté dans les pages de l'application.
//
//go:embed banner.js
var BannerJS []byte

// AdminHTML : page d'administration.
//
//go:embed admin.html
var AdminHTML []byte

// LoginTpl : gabarit de la page de connexion.
//
//go:embed login.html
var LoginTpl string

// ExpiredTpl : gabarit de la page « accès suspendu ».
//
//go:embed expired.html
var ExpiredTpl string

// SetupHTML : assistant d'installation.
//
//go:embed setup.html
var SetupHTML []byte
