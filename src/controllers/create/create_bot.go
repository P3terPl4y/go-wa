package create

import (
	wago "App/src/controllers/bot"
	"App/src/controllers/get"
	"App/src/controllers/save"
	"App/src/global"
	"App/src/global/functions"
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// truncateHistory recorta el historial de mensajes para no superar MAX_HISTORY_CHARS
// manteniendo los mensajes más recientes (prioridad al final del slice).

func InitBot(botID int, qrResult chan<- string) {
	ctx, cancel := context.WithCancel(context.Background())

	// Fix 6: helper para enviar en el channel de forma segura (evita panic si ya está cerrado)
	sendQR := func(val string) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("⚠️ [Bot %d] Panic recuperado al enviar QR: %v\n", botID, r)
			}
		}()
		if qrResult != nil {
			qrResult <- val
		}
	}

	defer func() {
		cancel()
		global.ActiveMu.Lock()
		delete(global.ActiveBots, botID)
		global.ActiveMu.Unlock()
		if qrResult != nil {
			close(qrResult)
		}
		fmt.Printf("🧹 [Bot %d] Finalizado.\n", botID)
	}()

	bot, err := get.GetBotByID(botID)
	if err != nil || bot == nil {
		fmt.Printf("❌ [Bot %d] Bot no encontrado\n", botID)
		return
	}
	if bot.Blocked {
		fmt.Printf("⛔ [Bot %d] Bot bloqueado, no se inicia\n", botID)
		return
	}
	// Verificar pago (excepto free)
	if bot.PaymentStatus != "free" && bot.PaymentStatus != "paid" {
		fmt.Printf("⛔ [Bot %d] Pago no confirmado (status: %s)\n", botID, bot.PaymentStatus)
		return
	}

	prompt, _ := get.GetPrompt(botID)
	if prompt == "" {
		fmt.Printf("⚠️ [Bot %d] Sin prompt, usando predeterminado\n", botID)
		prompt = "Eres un asistente útil."
	}

	_, err = get.GetExpiration(botID)
	if err != nil || err == sql.ErrNoRows {
		if err := save.SaveExpiration(botID, global.SUBSCRIPTION_DURATION); err != nil {
			fmt.Printf("❌ [Bot %d] Error guardando expiración: %v\n", botID, err)
			return
		}
	}
	exp, _ := get.GetExpiration(botID)
	fmt.Printf("📅 [Bot %d] Expira: %s\n", botID, exp.Format("2006-01-02 15:04:05"))

	global.ActiveMu.Lock()
	global.ActiveBots[botID] = true
	global.ActiveMu.Unlock()

	container := get.GetContainer(botID)
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		fmt.Printf("❌ [Bot %d] Error obteniendo dispositivo: %v\n", botID, err)
		return
	}

	clientLog := waLog.Stdout("Client", "WARN", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// 1. Ignorar mensajes propios
			if v.Info.IsFromMe {
				return
			}

			// 2. Ignorar mensajes de protocolo (sincronización interna)
			if v.Message.GetProtocolMessage() != nil {
				return
			}

			// 3. Ignorar grupos
			if v.Info.IsGroup {
				return
			}

			// 4. Deduplicación: ignorar si ya procesamos este messageID (Mejora 2)
			if global.MsgDedup.IsDuplicate(v.Info.ID) {
				fmt.Printf("🔄 [Bot %d] Mensaje duplicado ignorado: %s\n", botID, v.Info.ID)
				return
			}

			// 5. Extraer texto: soporta mensajes simples y extendidos (Mejora 4)
			text := v.Message.GetConversation()
			if text == "" {
				if ext := v.Message.GetExtendedTextMessage(); ext != nil {
					text = ext.GetText()
				}
			}
			if text == "" {
				fmt.Printf("📩 [Bot %d] Mensaje sin texto de %s, ignorado\n", botID, v.Info.Sender.ToNonAD())
				return
			}

			// 6. Truncar mensajes demasiado largos (protege contra spam de tokens)
			if len(text) > global.MAX_MSG_LENGTH {
				text = text[:global.MAX_MSG_LENGTH] + "..."
			}

			senderJID := v.Info.Sender.ToNonAD()
			userKey := fmt.Sprintf("%d:%s", botID, senderJID.String())

			// 7. Semáforo por usuario: evita respuestas duplicadas si escribe rápido (Mejora 6)
			/*if !global.UserSem.TryLock(userKey) {
				fmt.Printf("⏳ [Bot %d] Ya procesando mensaje de %s, ignorado\n", botID, senderJID)
				return
			}*/

			fmt.Printf("📩 [Bot %d] Mensaje de %s: %s\n", botID, senderJID, text)

			go wago.Switch(client, userKey, botID, senderJID, text)

		case *events.Disconnected:
			// Mejora 5: Reconexión automática al desconectarse
			fmt.Printf("🔁 [Bot %d] Desconectado, reconectando...\n", botID)
			go func() {
				time.Sleep(3 * time.Second)
				global.ActiveMu.Lock()
				_, isActive := global.ActiveBots[botID]
				global.ActiveMu.Unlock()
				if !isActive {
					return // el bot fue apagado intencionalmente
				}
				if err := functions.ConnectWithRetry(client); err != nil {
					fmt.Printf("❌ [Bot %d] No se pudo reconectar: %v\n", botID, err)
					cancel() // forzar cierre del lifecycle
				} else {
					fmt.Printf("✅ [Bot %d] Reconectado exitosamente\n", botID)
				}
			}()

		case *events.StreamReplaced:
			// Mejora 5: StreamReplaced ocurre cuando WhatsApp abre otra sesión
			fmt.Printf("⚠️ [Bot %d] Sesión reemplazada por otro dispositivo\n", botID)
			cancel()
		}
	})

	if client.Store.ID != nil {
		fmt.Printf("✅ [Bot %d] Sesión restaurada.\n", botID)
		if err := functions.ConnectWithRetry(client); err != nil {
			fmt.Printf("❌ [Bot %d] No se pudo conectar: %v\n", botID, err)
			return
		}
		sendQR("SESSION_EXISTS")
		fmt.Printf("🤖 [Bot %d] Activo.\n", botID)
		functions.RunLifecycle(botID, client, ctx, cancel)
		return
	}

	fmt.Printf("📱 [Bot %d] Generando QR...\n", botID)
	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		fmt.Printf("❌ [Bot %d] Error obteniendo QR: %v\n", botID, err)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("⚠️ [Bot %d] Panic recuperado en goroutine QR: %v\n", botID, r)
			}
		}()
		for evt := range qrChan {
			select {
			case <-ctx.Done():
				return
			default:
				if evt.Event == "code" {
					sendQR(evt.Code)
					fmt.Printf("⏳ [Bot %d] QR generado, expira en ~20s\n", botID)
				} else if evt.Event == "timeout" {
					fmt.Printf("⏰ [Bot %d] QR expirado.\n", botID)
					sendQR("TIMEOUT")
					cancel()
					return
				}
			}
		}
	}()

	if err := functions.ConnectWithRetry(client); err != nil {
		fmt.Printf("❌ [Bot %d] No se pudo conectar: %v\n", botID, err)
		return
	}

	fmt.Printf("⏳ [Bot %d] Esperando autenticación (60s)...\n", botID)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("⏰ [Bot %d] Autenticación falló.\n", botID)
			client.Disconnect()
			return
		case <-ticker.C:
			if client.Store.ID != nil {
				fmt.Printf("✅ [Bot %d] Vinculación exitosa.\n", botID)
				fmt.Printf("🤖 [Bot %d] Activo.\n", botID)
				functions.RunLifecycle(botID, client, ctx, cancel)
				return
			}
		}
	}
}
