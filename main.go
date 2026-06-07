package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("ClaveSecretaUltraSeguraDeMomMiel2026")

// =========================================================================
// ESTRUCTURAS DE DATOS (MODELOS)
// =========================================================================

type Usuario struct {
	ID            int       `json:"id"`
	Nombre        string    `json:"nombre" binding:"required"`
	Correo        string    `json:"correo" binding:"required"`
	Contrasena    string    `json:"contrasena,omitempty" binding:"required"`
	Rol           string    `json:"rol"`
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
}

type Producto struct {
	ID              int     `json:"id"`
	Nombre          string  `json:"nombre" binding:"required"`
	Precio          float64 `json:"precio" binding:"required"`
	Descripcion     string  `json:"descripcion"`
	Imagen          string  `json:"imagen" binding:"required"`
	IDCategoria     int     `json:"id_categoria" binding:"required"`
	NombreCategoria string  `json:"nombre_categoria"`
}

type DetallePedidoInput struct {
	IDProducto     int     `json:"id_producto" binding:"required"`
	Cantidad       int     `json:"cantidad" binding:"required"`
	PrecioUnitario float64 `json:"precio_unitario" binding:"required"`
}

type PedidoInput struct {
	Total    float64              `json:"total" binding:"required"`
	Detalles []DetallePedidoInput `json:"detalles" binding:"required"`
}

type Claims struct {
	UsuarioID int    `json:"id"`
	Rol       string `json:"rol"`
	jwt.RegisteredClaims
}

var db *sql.DB

// =========================================================================
// ENRUTADOR PRINCIPAL
// =========================================================================

func main() {
	// 1. Intentar cargar archivo .env (solo funciona en local)
	_ = godotenv.Load()

	// 2. Leer credenciales desde variables de entorno
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")

	// 3. Unir variables en formato de conexión de MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error configurando la BD: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("No se pudo conectar a Aiven: %v", err)
	}
	fmt.Println("¡Conexión Exitosa a Aiven con soporte JWT y CRUD activo de forma segura!")

	r := gin.Default()

	// Middleware de CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// RUTAS PÚBLICAS (Visitantes y Autorregistro de Clientes)
	r.POST("/api/usuarios/registrar", registrarUsuario)
	r.POST("/api/usuarios/login", loginUsuario)
	
	r.GET("/api/productos", listarProductos)
	r.GET("/api/productos/:id", obtenerProducto)
	r.GET("/api/categorias", listarCategorias)
	r.GET("/api/productos/populares", listarProductosPopulares)

	// RUTAS PROTEGIDAS (Requieren Token JWT)
	apiProtegida := r.Group("/api")
	apiProtegida.Use(JWTMiddleware())
	{
		// Transacciones de Pedidos (Clientes)
		apiProtegida.POST("/pedidos", crearPedido)
		apiProtegida.GET("/pedidos", listarMisPedidos)

		// Panel Administrativo - CRUD completo de Usuarios (Exclusivo Admin)
		apiProtegida.POST("/admin/usuarios/crear", adminCrearUsuario)
		apiProtegida.GET("/admin/usuarios", adminListarUsuarios)
		apiProtegida.PUT("/admin/usuarios/:id", adminActualizarUsuario)
		apiProtegida.DELETE("/admin/usuarios/:id", adminEliminarUsuario)

		// Panel Administrativo - CRUD de Productos (Solo Admin)
		apiProtegida.POST("/productos", crearProducto)
		apiProtegida.PUT("/productos/:id", actualizarProducto)
		apiProtegida.DELETE("/productos/:id", eliminarProducto)

		// Panel Administrativo - CRUD de Categorías (Solo Admin)
		apiProtegida.POST("/categorias", crearCategoria)
		apiProtegida.PUT("/categorias/:id", actualizarCategoria)
		apiProtegida.DELETE("/categorias/:id", eliminarCategoria)
	}

	r.Run(":8080")
}

// =========================================================================
// MIDDLEWARE JWT
// =========================================================================

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Falta el token de autorización o es inválido"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Token inválido o expirado"})
			c.Abort()
			return
		}

		c.Set("usuario_id", claims.UsuarioID)
		c.Set("rol", claims.Rol)
		c.Next()
	}
}

// =========================================================================
// CONTROLADORES: AUTENTICACIÓN Y GESTIÓN DE USUARIOS
// =========================================================================

func loginUsuario(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Credenciales incompletas"})
		return
	}

	var u Usuario
	err := db.QueryRow("SELECT id, nombre, correo, contrasena, rol FROM usuarios WHERE correo = ?", input.Correo).
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
	claims := &Claims{
		UsuarioID: u.ID,
		Rol:       u.Rol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(tiempoExpiracion),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
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

func registrarUsuario(c *gin.Context) {
	var input Usuario
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos inválidos"})
		return
	}

	var existeID int
	err := db.QueryRow("SELECT id FROM usuarios WHERE correo = ?", input.Correo).Scan(&existeID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El correo ya está registrado"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar credenciales"})
		return
	}

	_, err = db.Exec("INSERT INTO usuarios (nombre, correo, contrasena, rol) VALUES (?, ?, ?, 'cliente')",
		input.Nombre, input.Correo, string(hashedPassword))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al registrar en BD"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Usuario creado con éxito"})
}

func adminCrearUsuario(c *gin.Context) {
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
	err := db.QueryRow("SELECT id FROM usuarios WHERE correo = ?", input.Correo).Scan(&existeID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El correo ya está registrado"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar credenciales"})
		return
	}

	_, err = db.Exec("INSERT INTO usuarios (nombre, correo, contrasena, rol) VALUES (?, ?, ?, ?)",
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

// =========================================================================
// PANEL ADMINISTRATIVO - CRUD COMPLETO DE USUARIOS (Solo Admin)
// =========================================================================

func adminListarUsuarios(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado: Se requieren permisos de administrador"})
		return
	}

	rows, err := db.Query("SELECT id, nombre, correo, rol, fecha_registro FROM usuarios ORDER BY fecha_registro DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar usuarios"})
		return
	}
	defer rows.Close()

	var usuarios []Usuario = []Usuario{}
	for rows.Next() {
		var u Usuario
		if err := rows.Scan(&u.ID, &u.Nombre, &u.Correo, &u.Rol, &u.FechaRegistro); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer datos de usuarios"})
			return
		}
		usuarios = append(usuarios, u)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "count": len(usuarios), "data": usuarios})
}

func adminActualizarUsuario(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}
	id := c.Param("id")

	var input struct {
		Nombre string `json:"nombre" binding:"required"`
		Correo string `json:"correo" binding:"required"`
		Rol    string `json:"rol" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incorrectos o incompletos"})
		return
	}

	if input.Rol != "cliente" && input.Rol != "administrador" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Rol inválido"})
		return
	}

	_, err := db.Exec("UPDATE usuarios SET nombre=?, correo=?, rol=? WHERE id=?", input.Nombre, input.Correo, input.Rol, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al actualizar el usuario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Usuario modificado exitosamente por el administrador"})
}

func adminEliminarUsuario(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}
	id := c.Param("id")

	adminIDID, _ := c.Get("usuario_id")
	if fmt.Sprintf("%v", adminIDID) == id {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "No puedes eliminar tu propia cuenta de administrador"})
		return
	}

	_, err := db.Exec("DELETE FROM usuarios WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo eliminar el usuario (verifica si tiene pedidos asociados)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Usuario eliminado del sistema correctamente"})
}

// =========================================================================
// CRUD: PRODUCTOS
// =========================================================================

func listarProductos(c *gin.Context) {
    // Agregamos p.id_categoria a la consulta SQL
    query := `SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría') 
              FROM productos p 
              LEFT JOIN categorias c ON p.id_categoria = c.id`

    rows, err := db.Query(query)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar productos"})
        return
    }
    defer rows.Close()

    var productos []Producto = []Producto{} 
    for rows.Next() {
        var p Producto
        // Agregamos &p.IDCategoria en el Scan en la misma posición de la consulta
        if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer filas"})
            return
        }
        productos = append(productos, p)
    }

    c.JSON(http.StatusOK, gin.H{"status": "success", "count": len(productos), "data": productos})
}

func obtenerProducto(c *gin.Context) {
	id := c.Param("id")
	query := `SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, COALESCE(c.nombre_categoria, 'Sin categoría') 
              FROM productos p LEFT JOIN categorias c ON p.id_categoria = c.id WHERE p.id = ?`

	var p Producto
	err := db.QueryRow(query, id).Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.NombreCategoria)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Producto no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": p})
}

func crearProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado: Se requiere rol administrador"})
		return
	}

	var p Producto
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incompletos"})
		return
	}

	_, err := db.Exec("INSERT INTO productos (nombre, precio, descripcion, imagen, id_categoria) VALUES (?, ?, ?, ?, ?)",
		p.Nombre, p.Precio, p.Descripcion, p.Imagen, p.IDCategoria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al insertar producto"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "Producto añadido al catálogo de MomMiel"})
}

func actualizarProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}
	id := c.Param("id")

	var p Producto
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incorrectos"})
		return
	}

	_, err := db.Exec("UPDATE productos SET nombre=?, precio=?, descripcion=?, imagen=?, id_categoria=? WHERE id=?",
		p.Nombre, p.Precio, p.Descripcion, p.Imagen, p.IDCategoria, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al actualizar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Producto modificado con éxito"})
}

func eliminarProducto(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado"})
		return
	}
	id := c.Param("id")

	_, err := db.Exec("DELETE FROM productos WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo eliminar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Producto retirado del catálogo"})
}

// =========================================================================
// CRUD: CATEGORÍAS
// =========================================================================

func listarCategorias(c *gin.Context) {
	rows, err := db.Query("SELECT id, nombre_categoria, etapa_embarazo FROM categorias")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error en servidor"})
		return
	}
	defer rows.Close()

	var categorias []Categoria = []Categoria{}
	for rows.Next() {
		var cat Categoria
		if err := rows.Scan(&cat.ID, &cat.NombreCategoria, &cat.EtapaEmbarazo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error de lectura"})
			return
		}
		categorias = append(categorias, cat)
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": categorias})
}

func crearCategoria(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "No autorizado"})
		return
	}
	var cat Categoria
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incompletos"})
		return
	}

	_, err := db.Exec("INSERT INTO categorias (nombre_categoria, etapa_embarazo) VALUES (?, ?)", cat.NombreCategoria, cat.EtapaEmbarazo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al guardar"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "Categoría creada"})
}

func actualizarCategoria(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "No autorizado"})
		return
	}
	id := c.Param("id")
	var cat Categoria
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Entrada inválida"})
		return
	}

	_, err := db.Exec("UPDATE categorias SET nombre_categoria=?, etapa_embarazo=? WHERE id=?", cat.NombreCategoria, cat.EtapaEmbarazo, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error de actualización"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Categoría actualizada"})
}

func eliminarCategoria(c *gin.Context) {
	if rol, _ := c.Get("rol"); rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "No autorizado"})
		return
	}
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM categorias WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo eliminar (verifica si tiene productos asociados)"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Categoría borrada"})
}

// =========================================================================
// TRANSACCIONES COMPLEJAS: PEDIDOS Y DETALLES
// =========================================================================

func crearPedido(c *gin.Context) {
	usuarioID, _ := c.Get("usuario_id") 

	var input PedidoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos de compra mal estructurados"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error interno del sistema"})
		return
	}

	res, err := tx.Exec("INSERT INTO pedidos (id_usuario, total, estado) VALUES (?, ?, 'procesado')", usuarioID, input.Total)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al procesar orden general"})
		return
	}

	pedidoID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error de identificador"})
		return
	}

	for _, det := range input.Detalles {
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

	c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "¡Compra procesada con éxito en MomMiel! Tu pedido ya está en camino."})
}

func listarMisPedidos(c *gin.Context) {
	usuarioID, _ := c.Get("usuario_id")

	query := `SELECT id, fecha, total, estado FROM pedidos WHERE id_usuario = ? ORDER BY fecha DESC`
	rows, err := db.Query(query, usuarioID)
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

func listarProductosPopulares(c *gin.Context) {
    // Esta consulta cuenta cuántas veces se ha vendido cada producto,
    // hace un JOIN para traer sus datos y los ordena para darte el TOP 4.
    query := `
        SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, COALESCE(c.nombre_categoria, 'Sin categoría')
        FROM detalles_pedidos dp
        JOIN productos p ON dp.id_producto = p.id
        LEFT JOIN categorias c ON p.id_categoria = c.id
        GROUP BY p.id
        ORDER BY SUM(dp.cantidad) DESC
        LIMIT 4`

    rows, err := db.Query(query)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar populares"})
        return
    }
    defer rows.Close()

    var productos []Producto = []Producto{}
    for rows.Next() {
        var p Producto
        if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.NombreCategoria); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer datos"})
            return
        }
        productos = append(productos, p)
    }

    // RESPALDO: Si tu tienda es nueva y no hay ventas aún, te devuelve 4 productos aleatorios para que no se vea vacío
    if len(productos) == 0 {
        queryRespaldo := `SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, COALESCE(c.nombre_categoria, 'Sin categoría') FROM productos p LEFT JOIN categorias c ON p.id_categoria = c.id LIMIT 4`
        rowsR, _ := db.Query(queryRespaldo)
        defer rowsR.Close()
        for rowsR.Next() {
            var p Producto
            rowsR.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.NombreCategoria)
            productos = append(productos, p)
        }
    }

    c.JSON(http.StatusOK, gin.H{"status": "success", "data": productos})
}