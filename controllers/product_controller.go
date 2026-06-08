package controllers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"mommiel-api/models"
)

type ProductController struct {
	DB *sql.DB
}

func NewProductController(db *sql.DB) *ProductController {
	return &ProductController{DB: db}
}

func (pc *ProductController) ListarProductos(c *gin.Context) {
	query := `SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría'), p.activo 
	          FROM productos p 
	          LEFT JOIN categorias c ON p.id_categoria = c.id
	          WHERE p.activo = 1`

	rows, err := pc.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar productos"})
		return
	}
	defer rows.Close()

	var productos []models.Producto = []models.Producto{}
	for rows.Next() {
		var p models.Producto
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria, &p.Activo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al mapear filas"})
			return
		}
		productos = append(productos, p)
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "count": len(productos), "data": productos})
}

func (pc *ProductController) ListarProductosAdmin(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}

	query := `SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría'), p.activo 
	          FROM productos p 
	          LEFT JOIN categorias c ON p.id_categoria = c.id`

	rows, err := pc.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar productos globales"})
		return
	}
	defer rows.Close()

	var productos []models.Producto = []models.Producto{}
	for rows.Next() {
		var p models.Producto
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria, &p.Activo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar datos"})
			return
		}
		productos = append(productos, p)
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": productos})
}

func (pc *ProductController) CrearProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}

	var input models.Producto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos de producto no válidos"})
		return
	}

	query := "INSERT INTO productos (nombre, precio, descripcion, imagen, id_categoria, activo) VALUES (?, ?, ?, ?, ?, 1)"
	_, err := pc.DB.Exec(query, input.Nombre, input.Precio, input.Descripcion, input.Imagen, input.IDCategoria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo guardar el producto"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "Producto creado con éxito"})
}

func (pc *ProductController) ModificarProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}

	idParam := c.Param("id")
	var input models.Producto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos de actualización inválidos"})
		return
	}

	query := "UPDATE productos SET nombre=?, precio=?, descripcion=?, imagen=?, id_categoria=? WHERE id=?"
	_, err := pc.DB.Exec(query, input.Nombre, input.Precio, input.Descripcion, input.Imagen, input.IDCategoria, idParam)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo actualizar el producto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Producto modificado con éxito"})
}

func (pc *ProductController) EliminarProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}

	idParam := c.Param("id")
	query := "UPDATE productos SET activo = 0 WHERE id = ?"
	_, err := pc.DB.Exec(query, idParam)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo deshabilitar el producto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Producto dado de baja (borrado lógico)"})
}

func (pc *ProductController) ReactivarProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}

	idParam := c.Param("id")
	query := "UPDATE productos SET activo = 1 WHERE id = ?"
	_, err := pc.DB.Exec(query, idParam)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo reactivar el producto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Producto reactivado con éxito"})
}