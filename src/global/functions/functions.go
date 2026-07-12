package functions

import (
	"App/src/controllers/get"
	"App/src/global"
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
)

func CountBotsByUser(userID int) (int, error) {
	var count int
	err := global.ConfigDB.QueryRow(`SELECT COUNT(*) FROM bots WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func RunLifecycle(botID int, client *whatsmeow.Client, ctx context.Context, cancel context.CancelFunc) {
	// Fix 5: reducido de 5s a 60s para minimizar carga en DB con múltiples bots activos
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("⏹️ [Bot %d] Finalizando.\n", botID)
			client.Disconnect()
			return
		case <-ticker.C:
			exp, err := get.GetExpiration(botID)
			if err != nil {
				continue
			}
			if !exp.IsZero() && time.Now().After(exp) {
				fmt.Printf("⏰ [Bot %d] Suscripción expirada.\n", botID)
				cancel()
				client.Disconnect()
				return
			}
			bot, err := get.GetBotByID(botID)
			if err != nil || bot == nil || bot.Blocked {
				fmt.Printf("⛔ [Bot %d] Bot bloqueado o eliminado. Finalizando.\n", botID)
				cancel()
				client.Disconnect()
				return
			}
		}
	}
}

func ConnectWithRetry(client *whatsmeow.Client) error {
	var lastErr error
	for attempt := 1; attempt <= global.MAX_CONNECT_RETRIES; attempt++ {
		if err := client.Connect(); err != nil {
			lastErr = err
			fmt.Printf("⚠️ Intento %d conexión falló: %v\n", attempt, err)
			if attempt < global.MAX_CONNECT_RETRIES {
				time.Sleep(time.Duration(attempt*2) * time.Second)
			}
		} else {
			return nil
		}
	}
	return fmt.Errorf("falló después de %d intentos: %w", global.MAX_CONNECT_RETRIES, lastErr)
}
func LogFailedLogin(ip, reason string) {
	fmt.Printf("⚠️ Intento de login fallido desde %s - razón: %s (%s)\n", ip, reason, time.Now().Format(time.RFC3339))
}
