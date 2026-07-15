 package handlers
 
 func UpdateUserPhone(c fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Datos inválidos"})
	}
	// Validar que no esté vacío
	if req.Phone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "El número de teléfono es obligatorio"})
	}
	// Validar longitud mínima (por ejemplo, 8 dígitos)
	if len(req.Phone) < 8 {
		return c.Status(400).JSON(fiber.Map{"error": "El número debe tener al menos 8 caracteres"})
	}
	if err := controllers.UpdateUserPhone(userID, req.Phone); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok"}
}
