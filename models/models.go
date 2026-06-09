package models

import "time"

type Usuario struct {
	ID            int       `json:"id"`
	Nombre        string    `json:"nombre" binding:"required"`
	Correo        string    `json:"correo" binding:"required"`
	Contrasena    string    `json:"contrasena,omitempty" binding:"required"`
	Rol           string    `json:"rol"`
	Activo        int       `json:"activo"` 
	FechaRegistro time.Time `json:"fecha_registro"`
}

type LoginInput struct {
	Correo     string `json:"correo" binding:"required"`
	Contrasena string `json:"contrasena" binding:"required"`
}

type Categoria struct {
	ID              int    `json:"id"`
	NombreCategoria string `json:"nombre_categoria" binding:"required"`
	EtapaEmbarazo   string `json:"etapa_embarazo" binding:"required"`
	Activo          int    `json:"activo"`
}

type Producto struct {
	ID              int     `json:"id"`
	Nombre          string  `json:"nombre" binding:"required"`
	Precio          float64 `json:"precio" binding:"required"`
	Descripcion     string  `json:"descripcion"`
	Imagen          string  `json:"imagen" binding:"required"`
	IDCategoria     int     `json:"id_categoria" binding:"required"`
	NombreCategoria string  `json:"nombre_categoria"`
	Activo          int     `json:"activo"`
}

type DetallePedidoInput struct {
	IDProducto     int     `json:"id_producto" binding:"required"`
	Cantidad       int     `json:"cantidad" binding:"required"`
	PrecioUnitario float64 `json:"precio_unitario"`
	NombreProducto string  `json:"nombre_producto"`
}

type PedidoInput struct {
	Total           float64              `json:"total" binding:"required"`
	Direccion       string               `json:"direccion" binding:"required"`
	Ciudad          string               `json:"ciudad" binding:"required"`
	EstadoRepublica string               `json:"estado_republica" binding:"required"`
	CodigoPostal    string               `json:"codigo_postal" binding:"required"`
	Telefono        string               `json:"telefono" binding:"required"`
	Detalles        []DetallePedidoInput `json:"detalles" binding:"required"`
}

type PedidoCompleto struct {
	ID                  int                  `json:"id"`
	IDUsuario           int                  `json:"id_usuario"`
	NombreUsuario       string               `json:"nombre_usuario"`
	Fecha               time.Time            `json:"fecha"`
	Direccion           string               `json:"direccion"`
	Ciudad              string               `json:"ciudad"`
	EstadoRepublica     string               `json:"estado_republica"`
	CodigoPostal        string               `json:"codigo_postal"`
	Telefono            string               `json:"telefono"`
	Total               float64              `json:"total"`
	Estado              string               `json:"estado"`
	UltimaActualizacion time.Time            `json:"ultima_actualizacion"` 
	Detalles            []DetallePedidoInput `json:"detalles"`
}