package bot

import (
	"App/src/controllers/get"
	"App/src/controllers/save"
	"App/src/global"
	"App/src/models"
	"App/src/services/ai"
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func truncateHistory(history []models.ChatMessage) []models.ChatMessage {
	total := 0
	start := len(history)
	for i := len(history) - 1; i >= 0; i-- {
		total += len(history[i].Content)
		if total > global.MAX_HISTORY_CHARS {
			break
		}
		start = i
	}
	return history[start:]
}
func Responder(client *whatsmeow.Client, userKey string, botID int, recipient types.JID, txt string) (string, error) {
	defer global.UserSem.Unlock(userKey)

	// 8. Verificar bot activo y no bloqueado (verificación rápida en memoria primero)
	global.ActiveMu.Lock()
	_, isActive := global.ActiveBots[botID]
	global.ActiveMu.Unlock()
	if !isActive {
		return "Bot no activo", nil
	}

	// Guardar mensaje del usuario
	if err := save.SaveChatMessage(botID, recipient.String(), "user", txt); err != nil {
		fmt.Printf("❌ [Bot %d] Error guardando historial: %v\n", botID, err)
	}

	// Recuperar historial y truncar por caracteres (Mejora 3)
	history, err := get.GetChatHistory(botID, recipient.String(), global.MAX_HISTORY)
	if err != nil {
		history = []models.ChatMessage{}
	}
	history = truncateHistory(history)

	// Obtener prompt del sistema desde cache (Mejora 1)
	contexto, ok := global.PromptCache.Get(botID)
	if !ok {
		contexto, _ = get.GetPrompt(botID)
		global.PromptCache.Set(botID, contexto)
	}
	if contexto == "" {
		contexto = "Eres un asistente útil de WhatsApp. Responde de forma concisa."
	}

	// Construir prompt eficiente
	var promptBuilder strings.Builder
	promptBuilder.WriteString(contexto + "\n\n")
	for _, m := range history {
		switch m.Role {
		case "user":
			promptBuilder.WriteString("U: " + m.Content + "\n")
		case "assistant":
			promptBuilder.WriteString("A: " + m.Content + "\n")
		}
	}
	promptBuilder.WriteString("U: " + txt + "\nA:")

	// Llamar a la IA con timeout controlado
	type aiResult struct {
		resp string
		err  error
	}
	aiCh := make(chan aiResult, 1)
	go func() {
		r, e := ai.CallAI(promptBuilder.String())
		aiCh <- aiResult{r, e}
	}()

	var respuestaIA string
	select {
	case res := <-aiCh:
		if res.err != nil {
			fmt.Printf("❌ [Bot %d] Error IA: %v\n", botID, res.err)
			respuestaIA = "🤖 Lo siento, no pude procesar tu mensaje. Inténtalo de nuevo en un momento."
		} else {
			respuestaIA = res.resp
		}
	case <-time.After(global.AI_TIMEOUT_TOTAL):
		fmt.Printf("⏱️ [Bot %d] Timeout IA para %s\n", botID, recipient)
		respuestaIA = "🤖 Estoy tardando más de lo esperado. Inténtalo de nuevo."
	}

	// Guardar respuesta
	if err := save.SaveChatMessage(botID, recipient.String(), "assistant", respuestaIA); err != nil {
		fmt.Printf("❌ [Bot %d] Error guardando respuesta: %v\n", botID, err)
	}

	// Enviar respuesta a WhatsApp
	_, err = client.SendMessage(context.Background(), recipient, &waE2E.Message{
		Conversation: &respuestaIA,
	})
	if err != nil {
		return fmt.Sprintf("❌ [Bot %d] Error enviando a %s: %v\n", botID, recipient, err), err
	} else {
		return fmt.Sprintf("✅ [Bot %d] Respuesta enviada a %s\n", botID, recipient), nil
	}
}
