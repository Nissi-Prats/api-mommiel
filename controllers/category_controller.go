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
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar categorías"})
		return
	}
	defer rows.Close()

	var categorias []models.Categoria = []models.Categoria{}
	for rows.Next() {
		var cat models.Categoria
		if err := rows.Scan(&cat.ID, &cat.NombreCategoria, &cat.EtapaEmbarazo, &cat.Activo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar categorías"})
			return
		}
		categorias = append(categorias, cat)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": categorias})
}