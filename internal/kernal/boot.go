package kernal

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/0xJacky/Nginx-UI/internal/analytic"
	"github.com/0xJacky/Nginx-UI/internal/cache"
	"github.com/0xJacky/Nginx-UI/internal/cert"
	"github.com/0xJacky/Nginx-UI/internal/cluster"
	"github.com/0xJacky/Nginx-UI/internal/cron"
	"github.com/0xJacky/Nginx-UI/internal/logger"
	"github.com/0xJacky/Nginx-UI/internal/validation"
	"github.com/0xJacky/Nginx-UI/model"
	"github.com/0xJacky/Nginx-UI/query"
	"github.com/0xJacky/Nginx-UI/settings"
	"github.com/google/uuid"
	"mime"
	"runtime"
)

func Boot() {
	defer recovery()

	async := []func(){
		InitJsExtensionType,
		InitDatabase,
		InitNodeSecret,
		InitCryptoSecret,
		validation.Init,
		cache.Init,
	}

	syncs := []func(){
		analytic.RecordServerAnalytic,
	}

	for _, v := range async {
		v()
	}

	for _, v := range syncs {
		go v()
	}
}

func InitAfterDatabase() {
	syncs := []func(){
		registerPredefinedUser,
		cert.InitRegister,
		cron.InitCronJobs,
		cluster.RegisterPredefinedNodes,
		analytic.RetrieveNodesStatus,
	}

	for _, v := range syncs {
		go v()
	}
}

func recovery() {
	if err := recover(); err != nil {
		buf := make([]byte, 1024)
		runtime.Stack(buf, false)
		logger.Errorf("%s\n%s", err, buf)
	}
}

func InitDatabase() {
	// Skip install
	if settings.ServerSettings.SkipInstallation {
		skipInstall()
	}

	if "" != settings.ServerSettings.JwtSecret {
		db := model.Init()
		query.Init(db)

		InitAfterDatabase()
	}
}

func InitNodeSecret() {
	if "" == settings.ServerSettings.NodeSecret {
		logger.Warn("NodeSecret is empty, generating...")
		settings.ServerSettings.NodeSecret = uuid.New().String()

		err := settings.Save()
		if err != nil {
			logger.Error("Error save settings", err)
		}
		logger.Warn("Generated NodeSecret: ", settings.ServerSettings.NodeSecret)
	}
}

func InitCryptoSecret() {
	if "" == settings.CryptoSettings.Secret {
		logger.Warn("Secret is empty, generating...")

		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			logger.Error("Generate Secret failed: ", err)
			return
		}

		settings.CryptoSettings.Secret = hex.EncodeToString(key)

		err := settings.Save()
		if err != nil {
			logger.Error("Error save settings", err)
		}
		logger.Warn("Secret Generated")
	}
}

func InitJsExtensionType() {
	// Hack: fix wrong Content Type of .js file on some OS platforms
	// See https://github.com/golang/go/issues/32350
	_ = mime.AddExtensionType(".js", "text/javascript; charset=utf-8")
}
