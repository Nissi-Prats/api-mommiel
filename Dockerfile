# Paso 1: Compilar la aplicación con la versión correcta de Go
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copiar go.mod para descargar las dependencias
COPY go.mod ./ 
RUN go mod download

# Copiar el resto del código del proyecto
COPY . .

# Compilar de forma limpia
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