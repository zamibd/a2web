package handlers

import (
	"html/template"
	"log/slog"
)

type Handler struct {
	Logger    *slog.Logger
	Templates map[string]*template.Template
}

func New(logger *slog.Logger, tmpl map[string]*template.Template) *Handler {
	return &Handler{
		Logger:    logger,
		Templates: tmpl,
	}
}
