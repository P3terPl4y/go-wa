package bot

import (
	"App/src/global"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
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
		go notifyAdmin(recipient, txt)
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
func notifyAdmin(clientJID types.JID, msg string) {
	// Asegurar que el bot del admin esté disponible
	if global.AdminBotClient == nil {
		fmt.Println("⚠️ Admin bot no disponible para enviar notificación")
		return
	}
	// Construir mensaje
	notif := fmt.Sprintf("📦 Pedido/Cita de %s:\n%s", clientJID, msg)
	// Aquí deberías enviar el mensaje usando global.AdminBotClient y global.AdminJID
	// ...
	go notifyAdmin(recipient, notif)
	fmt.Printf("📦 Notificación al admin de %s: %s\n", clientJID, msg)
}
