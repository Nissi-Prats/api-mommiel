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
	db := config.GetDB()

	r := gin.Default()

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

	userCtrl := controllers.NewUserController(db)
	prodCtrl := controllers.NewProductController(db)
	catCtrl := controllers.NewCategoryController(db)
	orderCtrl := controllers.NewOrderController(db)

	// --- Rutas Públicas ---
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "¡API de Mom Miel funcionando con Arquitectura en Capas y POO!"})
	})

	r.POST("/api/usuarios/registrar", userCtrl.RegistrarUsuario)
	r.POST("/api/usuarios/login", userCtrl.Login)
	r.GET("/api/productos", prodCtrl.ListarProductos)
	r.GET("/api/categorias", catCtrl.ListarCategorias)

	// --- Rutas Protegidas por JWT ---
	protected := r.Group("/api")
	protected.Use(middlewares.ValidarJWT())
	{
		protected.GET("/usuarios/validar", userCtrl.ValidarToken)

		protected.POST("/pedidos", orderCtrl.CrearPedido)
		protected.GET("/pedidos/mis-pedidos", orderCtrl.ListarMisPedidos)

		protected.GET("/admin/productos", prodCtrl.ListarProductosAdmin)
		protected.POST("/admin/productos", prodCtrl.CrearProducto)
		protected.PUT("/admin/productos/:id", prodCtrl.ModificarProducto)
		protected.DELETE("/admin/productos/:id", prodCtrl.EliminarProducto)
		protected.PUT("/admin/productos/:id/reactivar", prodCtrl.ReactivarProducto)

		protected.GET("/admin/pedidos", orderCtrl.ListarTodosLosPedidosAdmin)
		protected.PUT("/admin/pedidos/:id/estado", orderCtrl.ActualizarEstadoPedido)
	}

	fmt.Println("Servidor corriendo en el puerto 8080...")
	r.Run(":8080")
}