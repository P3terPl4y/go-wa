package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3/client"
)

// ============================================================
// CONFIGURACIÓN — Las claves se cargan desde variables de entorno
// ============================================================
func getOpenRouterAPIKey() string {
	if k := os.Getenv("OPENROUTER_API_KEY"); k != "" {
		return k
	}
	// Fallback de desarrollo (eliminar en producción)
	return "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"
}

func getLegacyAPIKey() string {
	if k := os.Getenv("LEGACY_API_KEY"); k != "" {
		return k
	}
	// Fallback de desarrollo (eliminar en producción)
	return "apf_e9tf41wgk7am0l2499yr2eam"
}

const (
	openRouterURL = "https://openrouter.ai/api/v1/chat/completions"
	legacyURL     = "https://apifreellm.com/api/v1/chat"
	localAIURL    = "http://localhost:8080/v1/chat/completions"
)

// Lista de modelos gratuitos de OpenRouter
var freeModels = []string{
	"openrouter/free",
}

var modelIndex uint32  // para round-robin interno de OpenRouter
var sourceIndex uint32 // para round-robin entre fuentes (0: OpenRouter, 1: Legacy, 2: Local)

// ============================================================
// SANITIZACIÓN DE RESPUESTAS (Fix 1: tokens filtrados)
// ============================================================

var (
	// Tokens especiales de plantillas ChatML (Qwen, Llama, etc.)
	reChatmlTokens = regexp.MustCompile(`<\|[^|>]+\|>`)
	// Bloques <think>...</think> de modelos con razonamiento extendido
	reThinkBlocks = regexp.MustCompile(`(?s)<think>.*?</think>`)
	// Líneas sueltas que solo contienen "assistant", "user", "system"
	reRoleLines = regexp.MustCompile(`(?m)^\s*(assistant|user|system)\s*$`)
)

// sanitizeResponse limpia artefactos que los modelos LLM
// no deben enviar al usuario final (tokens ChatML, etiquetas de pensamiento, etc.)
func sanitizeResponse(text string) string {
	text = reThinkBlocks.ReplaceAllString(text, "")
	text = reChatmlTokens.ReplaceAllString(text, "")
	text = reRoleLines.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	// Colapsar múltiples saltos de línea consecutivos
	reMultiNewline := regexp.MustCompile(`\n{3,}`)
	text = reMultiNewline.ReplaceAllString(text, "\n\n")
	return text
}

// ============================================================
// CallAI CON FALLBACK AUTOMÁTICO (Fix 2)
// ============================================================

// CallAI alterna entre OpenRouter, Legacy y Local en cada llamada.
// Si la fuente del turno falla, prueba las demás automáticamente.
func CallAI(prompt string) (string, error) {
	turn := atomic.AddUint32(&sourceIndex, 1) % 3

	// Definir las fuentes en el orden que les corresponde según el turno
	type source struct {
		name string
		fn   func(string) (string, error)
	}
	all := []source{
		{"OpenRouter", callOpenRouter},
		{"Legacy", callLegacy},
		{"Local", Preguntar},
	}

	for i := 0; i < len(all); i++ {
		idx := (int(turn) + i) % len(all)
		s := all[idx]
		resp, err := s.fn(prompt)
		if err == nil && resp != "" {
			if i > 0 {
				fmt.Printf("⚠️ [AI] Fuente principal falló, respondió fallback: %s\n", s.name)
			}
			return resp, nil
		}
		fmt.Printf("⚠️ [AI] Fuente %s falló (intento %d): %v\n", s.name, i+1, err)
	}
	return "", fmt.Errorf("todas las fuentes de IA fallaron")
}

// ============================================================
// IMPLEMENTACIÓN DE OPENROUTER
// ============================================================

func callOpenRouter(prompt string) (string, error) {
	var lastErr error
	for i := 0; i < len(freeModels); i++ {
		idx := atomic.AddUint32(&modelIndex, 1) % uint32(len(freeModels))
		model := freeModels[idx]
		resp, err := callOpenRouterWithModel(prompt, model)
		if err == nil {
			return resp, nil
		}
		lastErr = fmt.Errorf("modelo %s: %w", model, err)
	}
	return "", fmt.Errorf("todos los modelos gratuitos de OpenRouter fallaron: %w", lastErr)
}

func callOpenRouterWithModel(prompt, model string) (string, error) {
	cc := client.New()
	cc.SetTimeout(30 * time.Second)

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + getOpenRouterAPIKey(),
		"HTTP-Referer":  "https://tudominio.com",
		"X-Title":       "WhatsApp Bot",
	}

	resp, err := cc.Post(openRouterURL, client.Config{
		Header: headers,
		Body:   payload,
	})
	if err != nil {
		return "", fmt.Errorf("solicitud fallida: %v", err)
	}
	defer resp.Close()

	if resp.StatusCode() != 200 {
		if resp.StatusCode() == 429 {
			return "", fmt.Errorf("rate limit (429) para modelo %s", model)
		}
		return "", fmt.Errorf("status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", fmt.Errorf("error parseando JSON: %v", err)
	}

	if errObj, ok := result["error"].(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok {
			return "", fmt.Errorf("OpenRouter error: %s", msg)
		}
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return sanitizeResponse(content), nil // Fix 1 aplicado
				}
			}
		}
	}
	return "", fmt.Errorf("no se pudo extraer respuesta")
}

// ============================================================
// IMPLEMENTACIÓN DE LEGACY
// ============================================================

func callLegacy(prompt string) (string, error) {
	cc := client.New()
	cc.SetTimeout(15 * time.Second)
	payload := map[string]string{"message": prompt}
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + getLegacyAPIKey(),
	}

	resp, err := cc.Post(legacyURL, client.Config{
		Header: headers,
		Body:   payload,
	})
	if err != nil {
		return "", fmt.Errorf("error en solicitud Legacy: %v", err)
	}
	defer resp.Close()

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("Legacy error %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", fmt.Errorf("error parseando Legacy: %v", err)
	}

	for _, key := range []string{"response", "message", "text"} {
		if respText, ok := result[key].(string); ok && respText != "" {
			return sanitizeResponse(respText), nil // Fix 1 aplicado
		}
	}
	return "", fmt.Errorf("Legacy no retornó texto válido")
}

// ============================================================
// IMPLEMENTACIÓN LOCAL (go-pherence) — Fix 3: formato ChatML correcto
// ============================================================

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Preguntar llama al servidor local go-pherence usando formato ChatML
// correcto con roles system/user para evitar que el modelo genere tokens de control.
func Preguntar(prompt string) (string, error) {
	reqBody := ChatRequest{
		Model: "qwen3-0.6b",
		Messages: []ChatMessage{
			// El rol "system" establece el contexto sin confundir al modelo
			{Role: "system", Content: "Eres un asistente de WhatsApp. Responde de forma concisa, clara y en el mismo idioma del usuario. No incluyas etiquetas, tokens ni marcadores internos en tu respuesta."},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 300,
		Stream:    false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error al marshal: %v", err)
	}

	httpClient := &http.Client{Timeout: 20 * time.Second}
	resp, err := httpClient.Post(localAIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error al llamar al servidor local: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("servidor local respondió con status %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("error al decodificar respuesta: %v", err)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("no se recibieron respuestas del servidor local")
	}

	return sanitizeResponse(chatResp.Choices[0].Message.Content), nil // Fix 1 aplicado
}
