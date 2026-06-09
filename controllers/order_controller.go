package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"mommiel-api/models"
)

type OrderController struct {
	DB *sql.DB
}

func NewOrderController(db *sql.DB) *OrderController {
	return &OrderController{DB: db}
}

func (oc *OrderController) CrearPedido(c *gin.Context) {
	usuarioID, _ := c.Get("usuario_id") 

	var input models.PedidoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos de compra o envío mal estructurados"})
		return
	}

	if len(input.Detalles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El carrito está vacío"})
		return
	}

	tx, err := oc.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error interno del sistema"})
		return
	}

	var totalCalculado float64 = 0.0

	type DetalleValidado struct {
		IDProducto     int
		Cantidad       int
		PrecioUnitario float64
	}
	var detallesValidados []DetalleValidado

	for _, det := range input.Detalles {
		var precioReal float64
		var activo int

		err := tx.QueryRow("SELECT precio, activo FROM productos WHERE id = ?", det.IDProducto).Scan(&precioReal, &activo)
        
		if err == sql.ErrNoRows || activo == 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Uno de los productos en tu carrito ya no está disponible"})
			return
		} else if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al verificar el inventario"})
			return
		}

		totalCalculado += precioReal * float64(det.Cantidad)
		detallesValidados = append(detallesValidados, DetalleValidado{
			IDProducto:     det.IDProducto,
			Cantidad:       det.Cantidad,
			PrecioUnitario: precioReal,
		})
	}

	queryInsertPedido := `
        INSERT INTO pedidos (id_usuario, direccion, ciudad, estado_republica, codigo_postal, telefono, total, estado) 
        VALUES (?, ?, ?, ?, ?, ?, ?, 'procesado')`
        
	res, err := tx.Exec(queryInsertPedido, 
		usuarioID, input.Direccion, input.Ciudad, input.EstadoRepublica, input.CodigoPostal, input.Telefono, totalCalculado)
        
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar los datos de envío y orden general"})
		return
	}

	pedidoID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error de identificador"})
		return
	}

	for _, det := range detallesValidados {
		_, err = tx.Exec("INSERT INTO detalles_pedidos (id_pedido, id_producto, cantidad, precio_unitario) VALUES (?, ?, ?, ?)",
			pedidoID, det.IDProducto, det.Cantidad, det.PrecioUnitario)
		if err != nil {
			tx.Rollback() 
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al registrar artículos del carrito"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo consolidar la compra"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "¡Compra procesada con éxito en MomMiel! Tu pedido ya está registrado para envío."})
}

func (oc *OrderController) ListarMisPedidos(c *gin.Context) {
	usuarioID, _ := c.Get("usuario_id")

	query := `SELECT id, fecha, total, estado FROM pedidos WHERE id_usuario = ? ORDER BY fecha DESC`
	rows, err := oc.DB.Query(query, usuarioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar historial"})
		return
	}
	defer rows.Close()

	type PedidoSimplificado struct {
		ID     int       `json:"id"`
		Fecha  time.Time `json:"fecha"`
		Total  float64   `json:"total"`
		Estado string    `json:"estado"`
	}

	var historial []PedidoSimplificado = []PedidoSimplificado{}
	for rows.Next() {
		var p PedidoSimplificado
		if err := rows.Scan(&p.ID, &p.Fecha, &p.Total, &p.Estado); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error de parseo"})
			return
		}
		historial = append(historial, p)
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": historial})
}

func (oc *OrderController) CancelarPedidoLogico(c *gin.Context) {
	pedidoID := c.Param("id")
	usuarioID, _ := c.Get("usuario_id")
	rol, _ := c.Get("rol")

	var currentEstado string
	var idDueno int
    
	queryCheck := "SELECT id_usuario, estado FROM pedidos WHERE id = ?"
	err := oc.DB.QueryRow(queryCheck, pedidoID).Scan(&idDueno, &currentEstado)
    
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "El pedido solicitado no existe en MomMiel"})
		return
	}

	if rol != "administrador" {
		if idDueno != usuarioID {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado: Este pedido no pertenece a tu cuenta"})
			return
		}
        
		if currentEstado != "procesado" {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error", 
				"message": fmt.Sprintf("No puedes cancelar este pedido porque su estado actual es '%s'", currentEstado),
			})
			return
		}
	}

	queryUpdate := "UPDATE pedidos SET estado = 'cancelado' WHERE id = ?"
	_, err = oc.DB.Exec(queryUpdate, pedidoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error interno al intentar actualizar el estado en la base de datos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("¡Pedido #%s cancelado correctamente de forma lógica!", pedidoID),
	})
}

func (oc *OrderController) ListarTodosPedidos(c *gin.Context) {
	queryPedidos := `
        SELECT 
            p.id, 
            p.id_usuario, 
            COALESCE(u.nombre, CONCAT('Usuario #', p.id_usuario)) AS nombre_usuario, 
            p.fecha, 
            p.direccion, 
            p.ciudad, 
            p.estado_republica, 
            p.codigo_postal, 
            p.telefono, 
            p.total, 
            p.estado, 
            p.ultima_actualizacion 
        FROM pedidos p
        LEFT JOIN usuarios u ON p.id_usuario = u.id
        ORDER BY p.fecha DESC`
    
	rows, err := oc.DB.Query(queryPedidos)
	if err != nil {
		c.JSON(500, gin.H{"status": "error", "message": "Error al consultar pedidos globales"})
		return
	}
	defer rows.Close()

	var historialGlobal []models.PedidoCompleto = []models.PedidoCompleto{}

	for rows.Next() {
		var p models.PedidoCompleto
		err := rows.Scan(&p.ID, &p.IDUsuario, &p.NombreUsuario, &p.Fecha, &p.Direccion, &p.Ciudad, &p.EstadoRepublica, &p.CodigoPostal, &p.Telefono, &p.Total, &p.Estado, &p.UltimaActualizacion)
		if err != nil {
			c.JSON(500, gin.H{"status": "error", "message": "Error al escanear pedidos"})
			return
		}

		queryDetalles := `
            SELECT dp.id_producto, p.nombre, dp.cantidad, dp.precio_unitario 
            FROM detalles_pedidos dp
            JOIN productos p ON dp.id_producto = p.id
            WHERE dp.id_pedido = ?`

		rowsD, err := oc.DB.Query(queryDetalles, p.ID)
		if err == nil {
			var detalles []models.DetallePedidoInput = []models.DetallePedidoInput{}
			for rowsD.Next() {
				var d models.DetallePedidoInput
				rowsD.Scan(&d.IDProducto, &d.NombreProducto, &d.Cantidad, &d.PrecioUnitario)
				detalles = append(detalles, d)
			}
			rowsD.Close()
			p.Detalles = detalles
		}
		historialGlobal = append(historialGlobal, p)
	}
	c.JSON(200, gin.H{"status": "success", "data": historialGlobal})
}

func (oc *OrderController) CambiarEstadoPedidoAdmin(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado: Se requieren permisos de administrador"})
		return
	}

	pedidoID := c.Param("id")
	var input struct {
		Estado string `json:"estado" binding:"required"` 
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El campo 'estado' es obligatorio"})
		return
	}

	if input.Estado != "procesado" && input.Estado != "en camino" && input.Estado != "entregado" && input.Estado != "cancelado" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El estado proporcionado no es válido para la logística de MomMiel"})
		return
	}

	query := "UPDATE pedidos SET estado = ? WHERE id = ?"
	result, err := oc.DB.Exec(query, input.Estado, pedidoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error interno al intentar actualizar el estado logístico"})
		return
	}

	filasAfectadas, _ := result.RowsAffected()
	if filasAfectadas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "El pedido solicitado no existe en la base de datos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"message":      fmt.Sprintf("¡Pedido #%s actualizado con éxito a el estado: '%s'!", pedidoID, input.Estado),
		"nuevo_estado": input.Estado,
	})
}