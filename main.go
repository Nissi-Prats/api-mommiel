package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"mommiel-api/config"
	"mommiel-api/controllers"
	"mommiel-api/middlewares"
)

func main() {
	// Obtener la instancia única de la base de datos (Singleton)
	db := config.GetDB()

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

	// Inyección de Dependencias utilizando Programación Orientada a Objetos (POO)
	userCtrl := controllers.NewUserController(db)
	prodCtrl := controllers.NewProductController(db)
	catCtrl := controllers.NewCategoryController(db)
	orderCtrl := controllers.NewOrderController(db)

	// --- RUTAS PÚBLICAS ---
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "¡API de MomMiel funcionando con Arquitectura MVC en Capas!"})
	})

	r.POST("/api/usuarios/registrar", userCtrl.RegistrarUsuario)
	r.POST("/api/usuarios/login", userCtrl.LoginUsuario)
	
	r.GET("/api/productos", prodCtrl.ListarProductos)
	r.GET("/api/productos/:id", prodCtrl.ObtenerProducto)
	r.GET("/api/categorias", catCtrl.ListarCategorias)
	r.GET("/api/productos/populares", prodCtrl.ListarProductosPopulares)

	// --- RUTAS PROTEGIDAS (Requieren Token JWT) ---
	apiProtegida := r.Group("/api")
	apiProtegida.Use(middlewares.JWTMiddleware())
	{
		// Perfil del Cliente
		apiProtegida.PUT("/usuarios/perfil", userCtrl.ActualizarMiPerfil)
		
		// Pedidos de Clientes
		apiProtegida.POST("/pedidos", orderCtrl.CrearPedido)
		apiProtegida.GET("/pedidos", orderCtrl.ListarMisPedidos)
		apiProtegida.PUT("/pedidos/:id/cancelar", orderCtrl.CancelarPedidoLogico)

		// Panel Administrativo - Usuarios
		apiProtegida.POST("/admin/usuarios/crear", userCtrl.AdminCrearUsuario)
		apiProtegida.GET("/admin/usuarios", userCtrl.AdminListarUsuarios)
		apiProtegida.PUT("/admin/usuarios/:id", userCtrl.AdminActualizarUsuario)
		apiProtegida.DELETE("/admin/usuarios/:id", userCtrl.AdminEliminarUsuario)

		// Panel Administrativo - Productos
		apiProtegida.POST("/productos", prodCtrl.CrearProducto)
		apiProtegida.PUT("/productos/:id", prodCtrl.ActualizarProducto)
		apiProtegida.DELETE("/productos/:id", prodCtrl.EliminarProducto)
		apiProtegida.POST("/productos/:id/activar", prodCtrl.ActivarProducto)
		apiProtegida.GET("/productos-admin", prodCtrl.ListarProductosAdmin)
		
		// Panel Administrativo - Categorías
		apiProtegida.POST("/categorias", catCtrl.CrearCategoria)
		apiProtegida.PUT("/categorias/:id", catCtrl.ActualizarCategoria)
		apiProtegida.DELETE("/categorias/:id", catCtrl.EliminarCategoria)

		// Panel Administrativo - Pedidos Globales
		apiProtegida.GET("/admin/pedidos/global", orderCtrl.ListarTodosPedidos)
		apiProtegida.PUT("/admin/pedidos/estado/:id", orderCtrl.CambiarEstadoPedidoAdmin)
	}

	fmt.Println("Servidor corriendo exitosamente en el puerto 8080...")
	r.Run(":8080")
}