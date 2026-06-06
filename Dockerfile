# Paso 1: Compilar la aplicación con la versión correcta de Go
FROM golang:1.26 AS builder

WORKDIR /app

# Copiar archivos del proyecto
COPY . .

# Compilar usando la carpeta vendor
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o main .

# Paso 2: Ejecutar la aplicación en un contenedor limpio y ligero
FROM alpine:latest

WORKDIR /app

# Copiar el binario compilado desde el paso anterior
COPY --from=builder /app/main .

# Exponer el puerto de la API
EXPOSE 8080

# Comando para arrancar el servidor
CMD ["./main"]