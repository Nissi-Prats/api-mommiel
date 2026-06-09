package controllers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"mommiel-api/models"
)

type CategoryController struct {
	DB *sql.DB
}

func NewCategoryController(db *sql.DB) *CategoryController {
	return &CategoryController{DB: db}
}

func (cc *CategoryController) ListarCategorias(c *gin.Context) {
	query := "SELECT id, nombre_categoria, etapa_embarazo, activo FROM categorias WHERE activo = 1"
	rows, err := cc.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error en servidor"})
		return
	}
	defer rows.Close()

	var categorias []models.Categoria = []models.Categoria{}
	for rows.Next() {
		var cat models.Categoria
		if err := rows.Scan(&cat.ID, &cat.NombreCategoria, &cat.EtapaEmbarazo, &cat.Activo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error de lectura"})
			return
		}
		categorias = append(categorias, cat)
	}c.JSON(http.StatusOK, gin.H{"status": "success", "data": categorias})
}

func (cc *CategoryController) CrearCategoria(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "No autorizado"})
		return
	}
	var cat models.Categoria
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incompletos"})
		return
	}

	_, err := cc.DB.Exec("INSERT INTO categorias (nombre_categoria, etapa_embarazo) VALUES (?, ?)", cat.NombreCategoria, cat.EtapaEmbarazo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al guardar"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "Categoría creada"})
}

func (cc *CategoryController) ActualizarCategoria(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "No autorizado"})
		return
	}
	id := c.Param("id")
	var cat models.Categoria
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Entrada inválida"})
		return
	}

	_, err := cc.DB.Exec("UPDATE categorias SET nombre_categoria=?, etapa_embarazo=? WHERE id=?", cat.NombreCategoria, cat.EtapaEmbarazo, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error de actualización"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Categoría actualizada"})
}

func (cc *CategoryController) EliminarCategoria(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "No authorized"})
		return
	}
	id := c.Param("id")

	_, err := cc.DB.Exec("UPDATE categorias SET activo = 0 WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo deshabilitar la categoría"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Categoría retirada del catálogo con éxito"})
}