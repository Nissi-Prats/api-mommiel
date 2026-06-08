package controllers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"mommiel-api/middlewares"
	"mommiel-api/models"
)

type UserController struct {
	DB *sql.DB
}

func NewUserController(db *sql.DB) *UserController {
	return &UserController{DB: db}
}

func (uc *UserController) RegistrarUsuario(c *gin.Context) {
	var input models.Usuario
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos de entrada inválidos"})
		return
	}

	input.Rol = "cliente"

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar la contraseña"})
		return
	}

	query := "INSERT INTO usuarios (nombre, correo, contrasena, rol, activo) VALUES (?, ?, ?, ?, 1)"
	_, err = uc.DB.Exec(query, input.Nombre, input.Correo, string(hashedPassword), input.Rol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "El correo ya está registrado o hubo un error en la BD"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "Usuario registrado exitosamente"})
}

func (uc *UserController) Login(c *gin.Context) {
	var input models.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Correo y contraseña requeridos"})
		return
	}

	var u models.Usuario
	query := "SELECT id, nombre, correo, contrasena, rol, activo FROM usuarios WHERE correo = ?"
	err := uc.DB.QueryRow(query, input.Correo).Scan(&u.ID, &u.Nombre, &u.Correo, &u.Contrasena, &u.Rol, &u.Activo)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Credenciales incorrectas (Usuario no encontrado)"})
		return
	}

	if u.Activo == 0 {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Esta cuenta está desactivada"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Contrasena), []byte(input.Contrasena))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Credenciales incorrectas (Contraseña errónea)"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHMAC, jwt.MapClaims{
		"id_usuario": u.ID,
		"correo":     u.Correo,
		"rol":        u.Rol,
		"exp":        time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(middlewares.JwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo generar el token de seguridad"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"token":   tokenString,
		"usuario": gin.H{"id": u.ID, "nombre": u.Nombre, "correo": u.Correo, "rol": u.Rol},
	})
}

func (uc *UserController) ValidarToken(c *gin.Context) {
	id, _ := c.Get("id_usuario")
	correo, _ := c.Get("correo")
	rol, _ := c.Get("rol")

	c.JSON(http.StatusOK, gin.H{
		"status": "valid",
		"user":   gin.H{"id_usuario": id, "correo": correo, "rol": rol},
	})
}