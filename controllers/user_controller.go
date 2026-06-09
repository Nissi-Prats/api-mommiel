package controllers

import (
	"database/sql"
	"fmt"
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

func (uc *UserController) LoginUsuario(c *gin.Context) {
	var input models.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Credenciales incompletas"})
		return
	}

	var u models.Usuario
	err := uc.DB.QueryRow("SELECT id, nombre, correo, contrasena, rol FROM usuarios WHERE correo = ? AND activo = 1", input.Correo).
		Scan(&u.ID, &u.Nombre, &u.Correo, &u.Contrasena, &u.Rol)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Usuario o contraseña incorrectos"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Contrasena), []byte(input.Contrasena))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Usuario o contraseña incorrectos"})
		return
	}

	tiempoExpiracion := time.Now().Add(24 * time.Hour)
	claims := &models.Claims{
		UsuarioID: u.ID,
		Rol:       u.Rol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(tiempoExpiracion),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(middlewares.JwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo generar el Token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"token":  tokenString,
		"user": gin.H{
			"id":     u.ID,
			"nombre": u.Nombre,
			"rol":    u.Rol,
		},
	})
}

func (uc *UserController) RegistrarUsuario(c *gin.Context) {
	var input models.Usuario
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos inválidos"})
		return
	}

	var existeID int
	err := uc.DB.QueryRow("SELECT id FROM usuarios WHERE correo = ?", input.Correo).Scan(&existeID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El correo ya está registrado"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar credenciales"})
		return
	}

	_, err = uc.DB.Exec("INSERT INTO usuarios (nombre, correo, contrasena, rol) VALUES (?, ?, ?, 'cliente')",
		input.Nombre, input.Correo, string(hashedPassword))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al registrar en BD"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Usuario creado con éxito"})
}

func (uc *UserController) AdminCrearUsuario(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado: Se requieren permisos de administrador"})
		return
	}

	var input struct {
		Nombre     string `json:"nombre" binding:"required"`
		Correo     string `json:"correo" binding:"required"`
		Contrasena string `json:"contrasena" binding:"required"`
		Rol        string `json:"rol" binding:"required"` 
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incompletos o inválidos"})
		return
	}

	if input.Rol != "cliente" && input.Rol != "administrador" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Rol no válido. Debe ser 'cliente' o 'administrador'"})
		return
	}

	var existeID int
	err := uc.DB.QueryRow("SELECT id FROM usuarios WHERE correo = ?", input.Correo).Scan(&existeID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El correo ya está registrado"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar credenciales"})
		return
	}

	_, err = uc.DB.Exec("INSERT INTO usuarios (nombre, correo, contrasena, rol) VALUES (?, ?, ?, ?)",
		input.Nombre, input.Correo, string(hashedPassword), input.Rol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al guardar el usuario en la BD"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Usuario '%s' con rol '%s' creado exitosamente por el administrador.", input.Nombre, input.Rol),
	})
}

func (uc *UserController) AdminListarUsuarios(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado: Se requieren permisos de administrador"})
		return
	}

	rows, err := uc.DB.Query("SELECT id, nombre, correo, rol, activo, fecha_registro FROM usuarios ORDER BY fecha_registro DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar usuarios"})
		return
	}
	defer rows.Close()

	var usuarios []models.Usuario = []models.Usuario{}
	for rows.Next() {
		var u models.Usuario
		if err := rows.Scan(&u.ID, &u.Nombre, &u.Correo, &u.Rol, &u.Activo, &u.FechaRegistro); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer datos de usuarios"})
			return
		}
		usuarios = append(usuarios, u)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "count": len(usuarios), "data": usuarios})
}

func (uc *UserController) AdminActualizarUsuario(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}
	id := c.Param("id")

	var input struct {
		Nombre string `json:"nombre" binding:"required"`
		Correo string `json:"correo" binding:"required"`
		Rol    string `json:"rol" binding:"required"`
		Activo *int   `json:"activo" binding:"required"` 
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incorrectos o incompletos"})
		return
	}

	if input.Rol != "cliente" && input.Rol != "administrador" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Rol inválido"})
		return
	}

	query := "UPDATE usuarios SET nombre=?, correo=?, rol=?, activo=? WHERE id=?"
	_, err := uc.DB.Exec(query, input.Nombre, input.Correo, input.Rol, *input.Activo, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al actualizar el usuario en la base de datos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Usuario modificado y estado actualizado exitosamente por el administrador"})
}

func (uc *UserController) AdminEliminarUsuario(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}
	id := c.Param("id")

	adminID, _ := c.Get("usuario_id")
	if fmt.Sprintf("%v", adminID) == id {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "No puedes eliminar tu propia cuenta de administrador"})
		return
	}

	query := "UPDATE usuarios SET activo = 0 WHERE id = ?"
	_, err := uc.DB.Exec(query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo suspender al usuario en el sistema"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Usuario dado de baja del sistema correctamente (Borrado Lógico)"})
}

func (uc *UserController) ActualizarMiPerfil(c *gin.Context) {
	usuarioID, exists := c.Get("usuario_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Usuario no autenticado"})
		return
	}

	var input struct {
		Nombre     string `json:"nombre" binding:"required"`
		Contrasena string `json:"contrasena"` 
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El campo nombre es obligatorio"})
		return
	}

	if input.Contrasena != "" {
		if len(input.Contrasena) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "La nueva contraseña debe tener al menos 6 caracteres"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Contrasena), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar la nueva contraseña"})
			return
		}

		query := "UPDATE usuarios SET nombre = ?, contrasena = ? WHERE id = ?"
		_, err = uc.DB.Exec(query, input.Nombre, string(hashedPassword), usuarioID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo actualizar el perfil"})
			return
		}
	} else {
		query := "UPDATE usuarios SET nombre = ? WHERE id = ?"
		_, err := uc.DB.Exec(query, input.Nombre, usuarioID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo actualizar el nombre"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "¡Tu perfil en MomMiel ha sido actualizado correctamente!"})
}