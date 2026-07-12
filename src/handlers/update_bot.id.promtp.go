package handlers

import (
	"App/src/controllers/get"
	"App/src/controllers/save"
	"App/src/global"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

func UpdateBotIDPrompt(c fiber.Ctx) error {
	botID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": "ID inválido"})
	}
	userID := c.Locals("user_id").(int)
	role := c.Locals("role").(string)
	bot, err := get.GetBotByID(botID)
	if err != nil || bot == nil {
		return c.JSON(fiber.Map{"status": "error", "message": "Bot no encontrado"})
	}
	if role != "admin" && bot.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"status": "error", "message": "No autorizado"})
	}
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": "Invalid request"})
	}
	if req.Prompt == "" {
		return c.JSON(fiber.Map{"status": "error", "message": "El prompt no puede estar vacío"})
	}
	if len(req.Prompt) > global.MAX_PROMPT_LENGTH {
		return c.JSON(fiber.Map{"status": "error", "message": fmt.Sprintf("El prompt no puede superar los %d caracteres", global.MAX_PROMPT_LENGTH)})
	}
	if err := save.SavePrompt(botID, req.Prompt); err != nil {
		return c.JSON(fiber.Map{"status": "error", "message": "Error guardando prompt"})
	}
	// Invalidar cache de prompts para que el bot tome el nuevo valor de inmediato
	global.PromptCache.Invalidate(botID)
	return c.JSON(fiber.Map{"status": "ok"})
}
