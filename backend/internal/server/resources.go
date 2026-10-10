package server

import (
	"context"
	"errors"
	"log/slog"
	"peptide-site/internal/conf"
	"peptide-site/internal/data"
	adminData "peptide-site/internal/data/admin"
	brandingData "peptide-site/internal/data/branding"
	catalogData "peptide-site/internal/data/catalog"
	mediaData "peptide-site/internal/data/media"
	newsData "peptide-site/internal/data/news"
	whatsappData "peptide-site/internal/data/whatsapp"
	"peptide-site/internal/pkg/configreload"
	"peptide-site/internal/pkg/geoip"
	"peptide-site/internal/service"
	adminService "peptide-site/internal/service/admin"
	brandingService "peptide-site/internal/service/branding"
	catalogService "peptide-site/internal/service/catalog"
	cms "peptide-site/internal/service/cms"
	mediaService "peptide-site/internal/service/media"
	newsService "peptide-site/internal/service/news"
	whatsappService "peptide-site/internal/service/whatsapp"
	"sync"
	"time"
)

// NewResources constructs one immutable dependency generation. Wire owns its
// cleanup; preparation never reuses a pool still serving old requests.
func NewResources(c conf.Config, d *data.Data, s *service.System, content *cms.Handler) (*configreload.Resource, error) {
	ctx, cancelPreparation := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelPreparation()
	identity, schema, root, commit, err := d.RuntimeMetadata(ctx, c.Data.Storage.UploadDir)
	if err != nil {
		return nil, err
	}
	for _, code := range c.Product.AllowedCurrencies {
		var count int64
		if err := d.DB.WithContext(ctx).Table("currency").Where("code=? AND enabled", code).Count(&count).Error; err != nil {
			return nil, err
		}
		if count != 1 {
			return nil, errors.New("currency dictionary is not initialized; run seed")
		}
	}
	authRepo, err := adminData.NewRepo(d, c)
	if err != nil {
		return nil, err
	}
	authHandler, err := adminService.NewHandler(authRepo, content.AdministrativeRoutes())
	if err != nil {
		return nil, err
	}
	mediaRepo := mediaData.NewRepo(d)
	mediaHandler, err := mediaService.NewHandler(mediaRepo, c, root)
	if err != nil {
		return nil, err
	}
	content.SetMedia(mediaHandler, mediaHandler.PublicFiles())
	content.SetBranding(brandingService.NewHandler(brandingData.NewRepo(d)))
	content.SetNews(newsService.NewHandler(newsData.NewRepo(d, c)))
	content.SetCatalog(catalogService.NewHandler(catalogData.NewRepo(d, c)))
	geo, err := geoip.Open(c.Contact.GeoIPDatabasePath)
	if err != nil {
		return nil, errors.New("country database unavailable")
	}
	content.SetWhatsApp(whatsappService.NewHandler(whatsappData.NewRepo(d), c, geo))
	content.SetAdministration(authHandler)
	handler := NewHTTPServer(c, s, content)
	var cancel context.CancelFunc
	var worker sync.WaitGroup
	commitResources := func() error {
		if err := commit(); err != nil {
			return err
		}
		if !c.Auth.TOTP.Enabled {
			slog.Warn("Admin TOTP disabled")
		}
		var workerCtx context.Context
		workerCtx, cancel = context.WithCancel(context.Background())
		worker.Add(1)
		go func() {
			defer worker.Done()
			ticker := time.NewTicker(c.Data.Storage.CleanupInterval)
			defer ticker.Stop()
			for {
				select {
				case <-workerCtx.Done():
					return
				case <-ticker.C:
					ctx, stop := context.WithTimeout(workerCtx, 10*time.Second)
					_ = mediaRepo.Cleanup(ctx)
					_ = authRepo.Cleanup(ctx)
					stop()
				}
			}
		}()
		return nil
	}
	return &configreload.Resource{Handler: handler, Identity: identity, Schema: schema, Commit: commitResources, Close: func() {
		geo.Close()
		if cancel != nil {
			cancel()
			worker.Wait()
		}
	}}, nil
}
