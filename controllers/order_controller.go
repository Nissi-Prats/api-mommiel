package controllers

import (
	"database/sql"
	"net/http"

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
	idUsuario, _ := c.Get("id_usuario")

	var input models.PedidoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Estructura del pedido incorrecta"})
		return
	}

	tx, err := oc.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al iniciar transacción"})
		return
	}

	queryPedido := `INSERT INTO pedidos (id_usuario, fecha, direccion, ciudad, estado_republica, codigo_postal, telefono, total, estado, ultima_actualizacion) 
	                VALUES (?, NOW(), ?, ?, ?, ?, ?, ?, 'pendiente', NOW())`

	res, err := tx.Exec(queryPedido, idUsuario, input.Direccion, input.Ciudad, input.EstadoRepublica, input.CodigoPostal, input.Telefono, input.Total)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo registrar la cabecera del pedido"})
		return
	}

	pedidoID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al recuperar ID del pedido"})
		return
	}

	queryDetalle := "INSERT INTO detalles_pedidos (id_pedido, id_producto, cantidad, precio_unitario) VALUES (?, ?, ?, ?)"
	for _, d := range input.Detalles {
		_, err := tx.Exec(queryDetalle, pedidoID, d.IDProducto, d.Cantidad, d.PrecioUnitario)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al insertar detalles de compra"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo consolidar la orden"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "¡Pedido procesado con éxito!", "id_pedido": pedidoID})
}

func (oc *OrderController) ListarMisPedidos(c *gin.Context) {
	idUsuario, _ := c.Get("id_usuario")

	query := `SELECT id, fecha, direccion, ciudad, estado_republica, codigo_postal, telefono, total, estado, ultima_actualizacion 
	          FROM pedidos WHERE id_usuario = ? ORDER BY fecha DESC`

	rows, err := oc.DB.Query(query, idUsuario)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar tus órdenes"})
		return
	}
	defer rows.Close()

	var pedidos []models.PedidoCompleto = []models.PedidoCompleto{}
	for rows.Next() {
		var p models.PedidoCompleto
		err := rows.Scan(&p.ID, &p.Fecha, &p.Direccion, &p.Ciudad, &p.EstadoRepublica, &p.CodigoPostal, &p.Telefono, &p.Total, &p.Estado, &p.UltimaActualizacion)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al estructurar el historial"})
			return
		}

		queryDetalles := `SELECT dp.id_producto, dp.cantidad, dp.precio_unitario, p.nombre 
		                  FROM detalles_pedidos dp
		                  JOIN productos p ON dp.id_producto = p.id
		                  WHERE dp.id_pedido = ?`

		detRows, err := oc.DB.Query(queryDetalles, p.ID)
		if err == nil {
			var detalles []models.DetallePedidoInput = []models.DetallePedidoInput{}
			for detRows.Next() {
				var det models.DetallePedidoInput
				if err := detRows.Scan(&det.IDProducto, &det.Cantidad, &det.PrecioUnitario, &det.NombreProducto); err == nil {
					detalles = append(detalles, det)
				}
			}
			detRows.Close()
			p.Detalles = detalles
		}
		pedidos = append(pedidos, p)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": pedidos})
}

func (oc *OrderController) ListarTodosLosPedidosAdmin(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}

	query := `SELECT p.id, p.id_usuario, u.nombre, p.fecha, p.direccion, p.ciudad, p.estado_republica, p.codigo_postal, p.telefono, p.total, p.estado, p.ultima_actualizacion 
	          FROM pedidos p
	          JOIN usuarios u ON p.id_usuario = u.id
	          ORDER BY p.fecha DESC`

	rows, err := oc.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar administración de órdenes"})
		return
	}
	defer rows.Close()

	var pedidos []models.PedidoCompleto = []models.PedidoCompleto{}
	for rows.Next() {
		var p models.PedidoCompleto
		err := rows.Scan(&p.ID, &p.IDUsuario, &p.NombreUsuario, &p.Fecha, &p.Direccion, &p.Ciudad, &p.EstadoRepublica, &p.CodigoPostal, &p.Telefono, &p.Total, &p.Estado, &p.UltimaActualizacion)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar listado maestro"})
			return
		}

		queryDetalles := `SELECT dp.id_producto, dp.cantidad, dp.precio_unitario, pr.nombre 
		                  FROM detalles_pedidos dp
		                  JOIN productos pr ON dp.id_producto = pr.id
		                  WHERE dp.id_pedido = ?`

		detRows, err := oc.DB.Query(queryDetalles, p.ID)
		if err == nil {
			var detalles []models.DetallePedidoInput = []models.DetallePedidoInput{}
			for detRows.Next() {
				var det models.DetallePedidoInput
				if err := detRows.Scan(&det.IDProducto, &det.Cantidad, &det.PrecioUnitario, &det.NombreProducto); err == nil {
					detalles = append(detalles, det)
				}
			}
			detRows.Close()
			p.Detalles = detalles
		}
		pedidos = append(pedidos, p)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": pedidos})
}

func (oc *OrderController) ActualizarEstadoPedido(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}

	idParam := c.Param("id")
	var input struct {
		Estado string `json:"estado" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Estado requerido"})
		return
	}

	query := "UPDATE pedidos SET estado = ?, ultima_actualizacion = NOW() WHERE id = ?"
	_, err := oc.DB.Exec(query, input.Estado, idParam)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo actualizar el estado del pedido"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Estado del pedido actualizado correctamente"})
}