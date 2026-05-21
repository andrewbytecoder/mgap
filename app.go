package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/moderato-app/live-pprof/internal/app"
)

type App struct {
	ctx     context.Context
	service *app.Service
}

func NewApp() *App {
	return &App{
		service: app.NewService(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) DetectURL(input string) ([]app.EndpointResult, error) {
	return a.service.DetectURL(a.ctx, input)
}

func (a *App) FetchMetrics(input string, metric string, profileSeconds uint64, useMock bool) (*app.MetricsSnapshot, error) {
	return a.service.FetchMetrics(a.ctx, input, metric, profileSeconds, useMock)
}

func (a *App) ImportProfile(metric string) (*app.MetricsSnapshot, error) {
	return a.service.ImportProfile(a.ctx, metric)
}

func (a *App) ExportProfile(metric string) (string, error) {
	return a.service.ExportProfile(a.ctx, metric)
}

func (a *App) ProfileMeta(metric string) (app.ProfileMeta, error) {
	return a.service.ProfileMeta(metric)
}

func (a *App) NormalizeURL(input string) (string, error) {
	return a.service.NormalizeURL(input)
}

func (a *App) InitialURL() string {
	return a.service.InitialURL()
}

func (a *App) AvailableMetrics() []app.MetricInfo {
	return a.service.AvailableMetrics()
}

func (a *App) AppInfo() map[string]string {
	return map[string]string{
		"name":        "live-pprof",
		"description": "Desktop pprof monitor built with Wails, Vue 3, TypeScript and Vuetify.",
		"stack":       strings.Join([]string{"Wails", "Vue 3", "TypeScript", "Vuetify"}, " + "),
		"hint":        fmt.Sprintf("Enter a pprof endpoint manually, such as %s", "http://localhost:6060/debug/pprof"),
	}
}
