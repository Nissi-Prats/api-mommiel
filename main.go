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
	Activo       	int     `json:"activo"`
}

type DetallePedidoInput struct {
	IDProducto     int     `json:"id_producto" binding:"required"`
	Cantidad       int     `json:"cantidad" binding:"required"`
	PrecioUnitario float64 `json:"precio_unitario"` //binding:"required"
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
	ID                 int             `json:"id"`
	IDUsuario          int             `json:"id_usuario"`
	NombreUsuario       string         `json:"nombre_usuario"`
	Fecha              time.Time       `json:"fecha"`
	Direccion          string          `json:"direccion"`
	Ciudad             string          `json:"ciudad"`
	EstadoRepublica    string          `json:"estado_republica"`
	CodigoPostal       string          `json:"codigo_postal"`
	Telefono           string          `json:"telefono"`
	Total              float64         `json:"total"`
	Estado             string          `json:"estado"`
	UltimaActualizacion time.Time      `json:"ultima_actualizacion"` 
	Detalles           []DetallePedidoInput `json:"detalles"`
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
		// Gestión del propio Perfil (Clientes / Todos)
    	apiProtegida.PUT("/usuarios/perfil", actualizarMiPerfil)
		// Transacciones de Pedidos (Clientes) y (Admin)
		apiProtegida.POST("/pedidos", crearPedido)
		apiProtegida.GET("/pedidos", listarMisPedidos)
		apiProtegida.PUT("/pedidos/:id/cancelar", cancelarPedidoLogico)

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

		//  NUEVO: Control Logístico de Pedidos Globales (Solo Admin)
        apiProtegida.GET("/admin/pedidos/global", listarTodosPedidos)// Ver historial de todas las mamás
        apiProtegida.PUT("/admin/pedidos/estado/:id", cambiarEstadoPedidoAdmin)
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
	err := db.QueryRow("SELECT id, nombre, correo, contrasena, rol FROM usuarios WHERE correo = ? AND activo = 1", input.Correo).
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

    // Añadimos u.activo a la consulta SQL
    rows, err := db.Query("SELECT id, nombre, correo, rol, activo, fecha_registro FROM usuarios ORDER BY fecha_registro DESC")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar usuarios"})
        return
    }
    defer rows.Close()

    var usuarios []Usuario = []Usuario{}
    for rows.Next() {
        var u Usuario
        // Agregamos &u.Activo en el Scan para recibir el valor (0 o 1)
        if err := rows.Scan(&u.ID, &u.Nombre, &u.Correo, &u.Rol, &u.Activo, &u.FechaRegistro); err != nil {
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

    // Añadimos 'Activo' a la estructura temporal que recibe los datos del Front
    var input struct {
        Nombre string `json:"nombre" binding:"required"`
        Correo string `json:"correo" binding:"required"`
        Rol    string `json:"rol" binding:"required"`
        Activo *int   `json:"activo" binding:"required"` // Usamos puntero *int para que acepte el valor 0 de forma obligatoria
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos incorrectos o incompletos"})
        return
    }

    if input.Rol != "cliente" && input.Rol != "administrador" {
        c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Rol inválido"})
        return
    }

    // UPDATE modificado para incluir el estado 'activo'
    query := "UPDATE usuarios SET nombre=?, correo=?, rol=?, activo=? WHERE id=?"
    _, err := db.Exec(query, input.Nombre, input.Correo, input.Rol, *input.Activo, id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al actualizar el usuario en la base de datos"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Usuario modificado y estado actualizado exitosamente por el administrador"})
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

    // BORRADO LÓGICO: En lugar de DELETE, desactivamos al usuario cambiando 'activo' a 0
    query := "UPDATE usuarios SET activo = 0 WHERE id = ?"
    _, err := db.Exec(query, id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo suspender al usuario en el sistema"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Usuario dado de baja del sistema correctamente (Borrado Lógico)"})
}

// =========================================================================
// CRUD: PRODUCTOS
// =========================================================================

func listarProductos(c *gin.Context) {
	// FILTRADO ACTIVADO: Agregamos p.activo al SELECT y filtramos con WHERE p.activo = 1
	query := `SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría'), p.activo 
	          FROM productos p 
	          LEFT JOIN categorias c ON p.id_categoria = c.id
	          WHERE p.activo = 1`

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al consultar productos"})
		return
	}
	defer rows.Close()

	var productos []Producto = []Producto{} 
	for rows.Next() {
		var p Producto
		// Agregamos &p.Activo al final del Scan para que coincida exactamente con el SELECT
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria, &p.Activo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer filas"})
			return
		}
		productos = append(productos, p)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "count": len(productos), "data": productos})
}

func obtenerProducto(c *gin.Context) {
	id := c.Param("id")

	var p Producto
	//  Agregamos p.activo al SELECT por consistencia
	query := `
		SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría'), p.activo 
		FROM productos p 
		LEFT JOIN categorias c ON p.id_categoria = c.id 
		WHERE p.id = ?`

	row := db.QueryRow(query, id)
	
	// Agregamos &p.Activo al Scan
	err := row.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria, &p.Activo)
	
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

	// Al insertar, dejamos que la BD use el DEFAULT 1 para 'activo'
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

	//  EL TRUCO MÁGICO: Cambiamos DELETE por un UPDATE lógico
	_, err := db.Exec("UPDATE productos SET activo = 0 WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo retirar el producto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Producto retirado del catálogo con éxito"})
}

// =========================================================================
// CRUD: CATEGORÍAS
// =========================================================================

func listarCategorias(c *gin.Context) {
	//  FILTRADO ACTIVADO: Solo seleccionamos las categorías donde activo = 1
	query := "SELECT id, nombre_categoria, etapa_embarazo, activo FROM categorias WHERE activo = 1"
	
	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error en servidor"})
		return
	}
	defer rows.Close()

	var categorias []Categoria = []Categoria{}
	for rows.Next() {
		var cat Categoria
		//  FILTRADO ACTIVADO: Solo seleccionamos las categorías donde activo = 1
		if err := rows.Scan(&cat.ID, &cat.NombreCategoria, &cat.EtapaEmbarazo, &cat.Activo); err != nil {
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

	// Al insertar, dejamos que use el DEFAULT 1 de la BD
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

	//  EL CAMBIO CLAVE: Cambiamos el DELETE físico por un UPDATE lógico
	_, err := db.Exec("UPDATE categorias SET activo = 0 WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo deshabilitar la categoría"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Categoría retirada del catálogo con éxito"})
}

// =========================================================================
// TRANSACCIONES COMPLEJAS: PEDIDOS Y DETALLES
// =========================================================================

func crearPedido(c *gin.Context) {
    usuarioID, _ := c.Get("usuario_id") 

    var input PedidoInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Datos de compra o envío mal estructurados"})
        return
    }

    if len(input.Detalles) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El carrito está vacío"})
        return
    }

    tx, err := db.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error interno del sistema"})
        return
    }

    //  VALIDACIÓN Y CÁLCULO DE PRECIOS REALES DESDE EL BACKEND
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

        // Buscamos el precio actual y el estado activo del producto
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

        // Sumamos al total e introducimos el precio congelado en nuestra lista temporal
        totalCalculado += precioReal * float64(det.Cantidad)
        detallesValidados = append(detallesValidados, DetalleValidado{
            IDProducto:     det.IDProducto,
            Cantidad:       det.Cantidad,
            PrecioUnitario: precioReal,
        })
    }

    //  INSERT DEL PEDIDO: Reemplazamos 'input.Total' por el 'totalCalculado' por Go
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

    //  INSERT DE DETALLES: Usamos los datos validados con los precios reales
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
    //  Añadimos WHERE p.activo = 1 y p.activo al SELECT para el mapeo del Struct
    query := `
        SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría'), p.activo
        FROM detalles_pedidos dp
        JOIN productos p ON dp.id_producto = p.id
        LEFT JOIN categorias c ON p.id_categoria = c.id
        WHERE p.activo = 1
        GROUP BY p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, c.nombre_categoria, p.activo
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
        //  Scan completo de 8 variables incluyendo &p.Activo
        if err := rows.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria, &p.Activo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer datos"})
            return
        }
        productos = append(productos, p)
    }

    // RESPALDO: Si no hay ventas aún, devuelve 4 productos vigentes aleatorios
    if len(productos) == 0 {
        //  Añadimos WHERE p.activo = 1 y p.activo al SELECT de respaldo
        queryRespaldo := `SELECT p.id, p.nombre, p.precio, p.descripcion, p.imagen, p.id_categoria, COALESCE(c.nombre_categoria, 'Sin categoría'), p.activo 
                          FROM productos p 
                          LEFT JOIN categorias c ON p.id_categoria = c.id 
                          WHERE p.activo = 1 
                          LIMIT 4`
        rowsR, err := db.Query(queryRespaldo)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error en consulta de respaldo"})
            return
        }
        defer rowsR.Close()
        
        for rowsR.Next() {
            var p Producto
            //  Scan completo de 8 variables para el respaldo
            if err := rowsR.Scan(&p.ID, &p.Nombre, &p.Precio, &p.Descripcion, &p.Imagen, &p.IDCategoria, &p.NombreCategoria, &p.Activo); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error al leer respaldo"})
                return
            }
            productos = append(productos, p)
        }
    }

    c.JSON(http.StatusOK, gin.H{"status": "success", "data": productos})
}
func cancelarPedidoLogico(c *gin.Context) {
    pedidoID := c.Param("id")
    usuarioID, _ := c.Get("usuario_id")
    rol, _ := c.Get("rol")

    // 1. Buscar el pedido en la BD para verificar a quién pertenece y su estado actual
    var currentEstado string
    var idDueno int
    
    queryCheck := "SELECT id_usuario, estado FROM pedidos WHERE id = ?"
    err := db.QueryRow(queryCheck, pedidoID).Scan(&idDueno, &currentEstado)
    
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "El pedido solicitado no existe en MomMiel"})
        return
    }

    // 2. Control de accesos y reglas para Clientes normales (no admin)
    if rol != "administrador" {
        // No puedes cancelar un pedido que no sea tuyo
        if idDueno != usuarioID {
            c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado: Este pedido no pertenece a tu cuenta"})
            return
        }
        
        //  Si ya lo enviaste o entregaste, el cliente ya no puede cancelarlo desde la interfaz
        if currentEstado != "procesado" {
            c.JSON(http.StatusBadRequest, gin.H{
                "status": "error", 
                "message": fmt.Sprintf("No puedes cancelar este pedido porque su estado actual es '%s'", currentEstado),
            })
            return
        }
    }

	// 3. BORRADO LÓGICO: Ejecutamos un UPDATE en lugar de un DELETE físico
	queryUpdate := "UPDATE pedidos SET estado = 'cancelado' WHERE id = ?"
	_, err = db.Exec(queryUpdate, pedidoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error interno al intentar actualizar el estado en la base de datos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("¡Pedido #%s cancelado correctamente de forma lógica!", pedidoID),
	})
}

func actualizarMiPerfil(c *gin.Context) {
    // 1. Obtener el ID del usuario autenticado desde el Token JWT
    usuarioID, exists := c.Get("usuario_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Usuario no autenticado"})
        return
    }

    // 2. Estructura para recibir los datos del Frontend
    // La contraseña es opcional (por si solo quiere cambiar su nombre)
    var input struct {
        Nombre     string `json:"nombre" binding:"required"`
        Contrasena string `json:"contrasena"` 
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El campo nombre es obligatorio"})
        return
    }

    // 3. Evaluar si el usuario decidió cambiar su contraseña o no
    if input.Contrasena != "" {
        // SI MANDÓ NUEVA CONTRASEÑA: La encriptamos y actualizamos ambos campos
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
        _, err = db.Exec(query, input.Nombre, string(hashedPassword), usuarioID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo actualizar el perfil"})
            return
        }
    } else {
        // NO MANDÓ CONTRASEÑA: Solo actualizamos el nombre para no borrar la clave actual
        query := "UPDATE usuarios SET nombre = ? WHERE id = ?"
        _, err := db.Exec(query, input.Nombre, usuarioID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "No se pudo actualizar el nombre"})
            return
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "success",
        "message": "¡Tu perfil en MomMiel ha sido actualizado correctamente!",
    })
}

// [READ - ADMIN] Obtener todas las órdenes con los nombres reales de los clientes y productos
func listarTodosPedidos(c *gin.Context) {
    // 1. Modificamos el SELECT para incluir un LEFT JOIN con la tabla de usuarios.
    // Usamos COALESCE para que si el usuario no existe, devuelva un texto vacío en lugar de null.
    queryPedidos := `
        SELECT 
            p.id, 
            p.id_usuario, 
            COALESCE(u.nombre, '') AS nombre_usuario, 
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
    
    rows, err := db.Query(queryPedidos)
    if err != nil {
        c.JSON(500, gin.H{"status": "error", "message": "Error al consultar pedidos globales"})
        return
    }
    defer rows.Close()

    var historialGlobal []PedidoCompleto = []PedidoCompleto{}

    for rows.Next() {
        var p PedidoCompleto
        
        // 2. Escaneamos respetando estrictamente el orden del SELECT. 
        // &p.NombreUsuario ocupa la tercera posición.
        err := rows.Scan(
            &p.ID, 
            &p.IDUsuario, 
            &p.NombreUsuario, 
            &p.Fecha, 
            &p.Direccion, 
            &p.Ciudad, 
            &p.EstadoRepublica, 
            &p.CodigoPostal, 
            &p.Telefono, 
            &p.Total, 
            &p.Estado, 
            &p.UltimaActualizacion,
        )
        if err != nil {
            c.JSON(500, gin.H{"status": "error", "message": "Error al escanear pedidos"})
            return
        }

        // 3. Respaldo de Seguridad: Si el usuario fue eliminado o no existe en Render, 
        // evitamos que quede vacío usando su ID de forma nativa sin romper el flujo.
        if p.NombreUsuario == "" {
            p.NombreUsuario = "Usuario #" + strconv.Itoa(p.IDUsuario)
        }

        // 4. Hacemos el JOIN con la tabla productos para traernos el detalle de los artículos compuestos
        queryDetalles := `
            SELECT dp.id_producto, p.nombre, dp.cantidad, dp.precio_unitario 
            FROM detalles_pedidos dp
            JOIN productos p ON dp.id_producto = p.id
            WHERE dp.id_pedido = ?`

        rowsD, err := db.Query(queryDetalles, p.ID)
        if err == nil {
            var detalles []DetallePedidoInput = []DetallePedidoInput{}
            for rowsD.Next() {
                var d DetallePedidoInput
                // Escaneamos p.nombre directo en d.NombreProducto
                rowsD.Scan(&d.IDProducto, &d.NombreProducto, &d.Cantidad, &d.PrecioUnitario)
                detalles = append(detalles, d)
            }
            rowsD.Close()
            p.Detalles = detalles
        }

        historialGlobal = append(historialGlobal, p)
    }

    // 5. Retornamos la respuesta limpia y estructurada para el Frontend
    c.JSON(200, gin.H{"status": "success", "data": historialGlobal})
}

// [PUT] /api/admin/pedidos/estado/:id
// Mueve el pedido a 'en camino', 'entregado' o 'cancelado' (Exclusivo Admin)
func cambiarEstadoPedidoAdmin(c *gin.Context) {
	// 1. Verificación de seguridad: Validamos que quien llame a la ruta sea un administrador
	rol, _ := c.Get("rol")
	if rol != "administrador" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Acceso denegado: Se requieren permisos de administrador"})
		return
	}

	// Capturamos el parámetro ID de la URL (ej. /estado/14)
	pedidoID := c.Param("id")

	// 2. Estructura local para mapear y validar el JSON recibido desde el frontend del Admin
	var input struct {
		Estado string `json:"estado" binding:"required"` // 'en camino', 'entregado', 'cancelado', etc.
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El campo 'estado' es obligatorio"})
		return
	}

	//  VALIDACIÓN DE SEGURIDAD: Comprobamos que el texto coincida de forma estricta con  opciones del ENUM
	if input.Estado != "procesado" && input.Estado != "en camino" && input.Estado != "entregado" && input.Estado != "cancelado" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "El estado proporcionado no es válido para la logística de MomMiel"})
		return
	}

	// 3. Ejecutamos la actualización directa en la Base de Datos
	//  Al usar "ON UPDATE CURRENT_TIMESTAMP" en  BD, MySQL actualizará la fecha sola al guardar el cambio.
	query := "UPDATE pedidos SET estado = ? WHERE id = ?"
	result, err := db.Exec(query, input.Estado, pedidoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Error interno al intentar actualizar el estado logístico"})
		return
	}

	// 4. Verificación extra: Confirmamos si el pedido realmente existía en las tablas
	filasAfectadas, _ := result.RowsAffected()
	if filasAfectadas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "El pedido solicitado no existe en la base de datos"})
		return
	}

	// Respuesta exitosa
	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"message":      fmt.Sprintf("¡Pedido #%s actualizado con éxito a el estado: '%s'!", pedidoID, input.Estado),
		"nuevo_estado": input.Estado,
	})
}