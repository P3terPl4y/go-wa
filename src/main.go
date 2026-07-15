package main

import (
	"App/src/controllers/get"
	"App/src/database"
	"App/src/global"
	"App/src/middleware"
	"App/src/router"
	"context"
	"fmt"
	"log"
	"os"

	//"os"
	waLog "go.mau.fi/whatsmeow/util/log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/static"
	"go.mau.fi/whatsmeow"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	
    // ... tus imports existentes ...
    "time"

)

func init() {
	global.GoogleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// ============================================================
// SERVIDOR WEB
// ============================================================
func startAdminBot() {
	// Obtener usuario admin
	adminUser, err := get.GetUserByUsername(global.ADMIN_USERNAME)
	if err != nil || adminUser == nil {
		log.Println("⚠️ No se encontró usuario admin")
		return
	}
	bots, err := get.GetBotsByUser(adminUser.ID)
	if err != nil || len(bots) == 0 {
		log.Println("⚠️ El admin no tiene ningún bot. Crea uno manualmente desde el panel.")
		return
	}
	adminBot := bots[0]

	go func() {
		for {
			ctx := context.Background()
			container := get.GetContainer(adminBot.ID)
			deviceStore, err := container.GetFirstDevice(ctx)
			if err != nil {
				log.Printf("❌ Admin bot: error obteniendo dispositivo: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			clientLog := waLog.Stdout("AdminClient", "WARN", true)
			client := whatsmeow.NewClient(deviceStore, clientLog)

			if err := client.Connect(); err != nil {
				log.Printf("❌ Admin bot: error conectando: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			// Guardar en global
			global.AdminBotClient = client
			if client.Store.ID != nil {
				global.AdminJID = *client.Store.ID
				log.Printf("✅ Admin bot activo como %s", global.AdminJID)
			} else {
				log.Println("⚠️ Admin bot: no se pudo obtener el JID")
			}

			// Mantener la goroutine viva y verificar conexión cada 30 segundos
			// Si la conexión se cae, el bucle exterior lo detectará al reintentar
			time.Sleep(30 * time.Second)
		}
	}()
}
func main() {
	database.InitDB()
	// Obtener el usuario admin
	 
			// Iniciar el bot del admin en segundo plano y capturar el cliente
			go startAdminBot()
		
	
	app := fiber.New(fiber.Config{
		TrustProxy: true,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": "Ha ocurrido un error. Inténtelo de nuevo más tarde."})
		},
	})

	app.Use(logger.New(logger.Config{
		Format: "${time} - ${method} ${path} ${status}\n",
	}))
	app.Use(middleware.SecurityHeaders)
	app.Use(global.SessionMW)
	app.Use(static.New("./src/static/"))
	router.Router(app)
	// ============================================================
	// INICIO DEL SERVIDOR
	// ============================================================
	fmt.Printf("🚀 Servidor iniciando en http://localhost:3000\n")
	if err := app.Listen("127.0.0.1:3000"); err != nil {
		log.Fatalf("❌ Error: %v", err)
	}
}
