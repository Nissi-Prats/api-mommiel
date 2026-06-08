# Paso 1: Compilar la aplicación con la versión correcta de Go
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copiar go.mod y descargar las dependencias primero (aprovecha la caché de Docker)
COPY go.mod ./
# Si tienes un archivo go.sum, descomenta la siguiente línea quitando el '#'
# COPY go.sum ./

RUN go mod download

# Copiar el resto del código del proyecto
COPY . .

# Compilar de forma limpia sin depender de la carpeta vendor
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Paso 2: Ejecutar la aplicación en un contenedor limpio y ligero
FROM alpine:latest

WORKDIR /app

# Copiar el binario compilado desde el paso anterior
COPY --from=builder /app/main .

# Exponer el puerto de la API
EXPOSE 8080

# Comando para arrancar el servidor
CMD ["./main"]