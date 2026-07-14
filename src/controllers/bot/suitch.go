package bot

import (
	"App/src/global"
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// Switch maneja el flujo de mensajes según el estado de bloqueo del usuario.
func Switch(client *whatsmeow.Client, userKey string, botID int, recipient types.JID, txt string) {
	// Bloqueamos todo el crítico con un solo Lock para evitar deadlocks
	global.WrMu.Lock()
	defer global.WrMu.Unlock()

	blocked := global.SenderJIDsBlocked[recipient]

	if blocked {
		// Caso: bot bloqueado
		switch {
		case txt == "-start":
			// Desbloquear
			delete(global.SenderJIDsBlocked, recipient)
			fmt.Printf("✅ Bot iniciado para: %s\n", recipient)
			return

		case strings.Contains(txt, "@Bot"):
			// Responder aunque esté bloqueado (liberamos el mutex antes de la operación pesada)
			global.WrMu.Unlock()
			go respondAndPrint(client, userKey, botID, recipient, txt)
			return

		default:
			// No hacer nada
			return
		}
	}

	// Caso: bot NO bloqueado
	switch {
	case txt == "-stop":
		// Bloquear
		global.SenderJIDsBlocked[recipient] = true
		fmt.Printf("⛔ Bot detenido para: %s\n", recipient)
		return

	case strings.Contains(txt, "Pedido:") || strings.Contains(txt, "Agendar Cita:"):
		// Lógica especial (sin responder)
		// Aquí podrías enviar notificación al dueño, etc.
		global.WrMu.Unlock()
		go notifyAdmin(recipient, txt)
		fmt.Println("📦 Procesando pedido/cita...")
		return

	default:
		// Responder normalmente
		global.WrMu.Unlock()
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
func notifyAdmin(clientJID types.JID, msg string) {
	// Verificar que el bot del admin esté disponible
	if global.AdminBotClient == nil {
		fmt.Println("⚠️ Admin bot no disponible para enviar notificación")
		return
	}
	// El destinatario es el JID del admin (su número)
	if global.AdminJID.IsEmpty() {
		fmt.Println("⚠️ JID del admin no configurado")
		return
	}
	// Construir mensaje con información del cliente
	notif := fmt.Sprintf("📦 Nuevo pedido/cita de %s:\n%s", clientJID, msg)
	_, err := global.AdminBotClient.SendMessage(context.Background(), global.AdminJID, &waE2E.Message{
		Conversation: &notif,
	})
	if err != nil {
		fmt.Printf("❌ Error enviando notificación al admin: %v\n", err)
	} else {
		fmt.Printf("✅ Notificación enviada al admin sobre pedido de %s\n", clientJID)
	}
}
