package bot

import (
	"App/src/controllers/get"
	"App/src/global"
	"context"
	"fmt"
	"strings"
"time"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// Switch maneja el flujo de mensajes según el estado de bloqueo del usuario.
func Switch(client *whatsmeow.Client, userKey string, botID int, recipient types.JID, txt string) {
	global.WrMu.Lock()
	defer global.WrMu.Unlock()

	// Asegurar que el mapa no sea nil (por si el paquete global no lo inicializa)
	if global.SenderJIDsBlocked == nil {
		global.SenderJIDsBlocked = make(map[types.JID]bool)
	}

	blocked := global.SenderJIDsBlocked[recipient]

	if blocked {
		switch {
		case txt == "-start":
			delete(global.SenderJIDsBlocked, recipient)
			fmt.Printf("✅ Bot iniciado para: %s\n", recipient)
			return

		case strings.Contains(txt, "@Bot"):
			// Responder aunque esté bloqueado
			go respondAndPrint(client, userKey, botID, recipient, txt)
			return

		default:
			return
		}
	}

	// Caso: bot NO bloqueado
	switch {
	case txt == "-stop":
		global.SenderJIDsBlocked[recipient] = true
		fmt.Printf("⛔ Bot detenido para: %s\n", recipient)
		return

	case strings.Contains(txt, "Pedido:") || strings.Contains(txt, "Agendar Cita:"):
		// Notificar al administrador (en goroutine para no bloquear)
		go notifyAdmin(botID, recipient, txt)
		return

	default:
		// Responder normalmente
		go respondAndPrint(client, userKey, botID, recipient, txt)
	}
}

// respondAndPrint ejecuta Responder y maneja el error/resultado.
func respondAndPrint(client *whatsmeow.Client, userKey string, botID int, recipient types.JID, txt string) {
	res, err := Responder(client, userKey, botID, recipient, txt)
	if err != nil {
		fmt.Printf("❌ Error en Responder: %v\n", err)
		return
	}
	fmt.Println(res)
}

// notifyAdmin envía una notificación al administrador.
// notifyAdmin envía una notificación al dueño del bot usando el bot del administrador.
func notifyAdmin(botID int, clientJID types.JID, msg string) {
    if global.AdminBotClient == nil {
        fmt.Println("⚠️ Admin bot no disponible para enviar notificación")
        return
    }

    bot, err := get.GetBotByID(botID)
    if err != nil || bot == nil {
        fmt.Printf("❌ Error obteniendo bot %d: %v\n", botID, err)
        return
    }

    user, err := get.GetUserByID(bot.UserID)
    if err != nil || user == nil {
        fmt.Printf("❌ Error obteniendo usuario dueño del bot %d: %v\n", botID, err)
        return
    }

    // Validar y construir JID del usuario
    phone := strings.TrimPrefix(user.Phone, "+")
    if phone == "" {
        fmt.Printf("❌ El usuario %d no tiene número de teléfono válido\n", user.ID)
        return
    }
    userJID, err := types.ParseJID(phone + "@s.whatsapp.net")
    if err != nil {
        fmt.Printf("❌ Error parseando JID del usuario %s: %v\n", user.Phone, err)
        return
    }

    notif := fmt.Sprintf("📦 Nuevo pedido/cita de %s:\n%s", clientJID, msg)

    // Reintentar hasta 3 veces con espera progresiva
    for attempt := 1; attempt <= 3; attempt++ {
        // Verificar si el bot admin sigue conectado
        if global.AdminBotClient == nil {
            fmt.Printf("⚠️ Admin bot desconectado, reintento %d/3...\n", attempt)
            time.Sleep(2 * time.Second)
            continue
        }

        _, err = global.AdminBotClient.SendMessage(context.Background(), userJID, &waE2E.Message{
            Conversation: &notif,
        })
        if err == nil {
            fmt.Printf("✅ Notificación enviada al usuario %s (dueño del bot %d)\n", user.Phone, botID)
            return
        }

        fmt.Printf("❌ Error enviando notificación (intento %d/3): %v\n", attempt, err)
        time.Sleep(time.Duration(attempt*2) * time.Second)
    }

    // Si fallaron todos los reintentos, guardar en una cola local (opcional)
    fmt.Printf("⚠️ No se pudo enviar notificación al usuario %s después de 3 intentos\n", user.Phone)
    // Aquí podrías almacenar el mensaje en una base de datos para enviarlo después
}
