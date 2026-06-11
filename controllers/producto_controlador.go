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
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer filas"})
			return
		}
		productos = append(productos, p)
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "count": len(productos), "data": productos})
}

func (pc *ProductController) ObtenerProducto(c *gin.Context) {
	id := c.Param("id")
	var p models.Producto
	query := `SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría'), p.activo 
		FROM productos p 
		LEFT JOIN categorias c ON p.id_categoria = c.id 
		WHERE p.id = ?`

	err := pc.DB.QueryRow(query, id).Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria, &p.Activo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Producto no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": p})
}

func (pc *ProductController) CrearProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado: Se requiere rol administrador"})
		return
	}

	var p models.Producto
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incompletos"})
		return
	}

	_, err := pc.DB.Exec("INSERT INTO productos (nombre, precio, descripcion, imagen, id_categoria) VALUES (?, ?, ?, ?, ?)",
		p.Nombre, p.Precio, p.Descripcion, p.Imagen, p.IDCategoria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al insertar producto"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "Producto añadido al catálogo de MomMiel"})
}

func (pc *ProductController) ActualizarProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}
	id := c.Param("id")

	var p models.Producto
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incorrectos"})
		return
	}

	_, err := pc.DB.Exec("UPDATE productos SET nombre=?, precio=?, descripcion=?, imagen=?, id_categoria=? WHERE id=?",
		p.Nombre, p.Precio, p.Descripcion, p.Imagen, p.IDCategoria, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al actualizar"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Producto modificado con éxito"})
}

func (pc *ProductController) EliminarProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}
	id := c.Param("id")

	_, err := pc.DB.Exec("UPDATE productos SET activo = 0 WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo retirar el producto"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Producto retirado del catálogo con éxito"})
}

func (pc *ProductController) ActivarProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}
	id := c.Param("id")

	_, err := pc.DB.Exec("UPDATE productos SET activo = 1 WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo reactivar el producto"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Producto reactivado en el catálogo con éxito"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer filas"})
			return
		}
		productos = append(productos, p)
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "count": len(productos), "data": productos})
}

func (pc *ProductController) ListarProductosPopulares(c *gin.Context) {
	query := `
        SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría'), p.activo
        FROM detalles_pedidos dp
        JOIN productos p ON dp.id_producto = p.id
        LEFT JOIN categorias c ON p.id_categoria = c.id
        WHERE p.activo = 1
        GROUP BY p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, c.nombre_categoria, p.activo
        ORDER BY SUM(dp.cantidad) DESC
        LIMIT 4`

	rows, err := pc.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar populares"})
		return
	}
	defer rows.Close()

	var productos []models.Producto = []models.Producto{}
	for rows.Next() {
		var p models.Producto
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria, &p.Activo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer datos"})
			return
		}
		productos = append(productos, p)
	}

	if len(productos) == 0 {
		queryRespaldo := `SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría'), p.activo 
                          FROM productos p 
                          LEFT JOIN categorias c ON p.id_categoria = c.id 
                          WHERE p.activo = 1 
                          LIMIT 4`
		rowsR, err := pc.DB.Query(queryRespaldo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error en consulta de respaldo"})
			return
		}
		defer rowsR.Close()
        
		for rowsR.Next() {
			var p models.Producto
			if err := rowsR.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria, &p.Activo); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer respaldo"})
				return
			}
			productos = append(productos, p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": productos})
}