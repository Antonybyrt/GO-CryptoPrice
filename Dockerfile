# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copier les fichiers de dépendances
COPY go.mod go.sum ./

# Télécharger les dépendances
RUN go mod download

# Copier le code source
COPY . .

# Compiler l'application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copier l'exécutable depuis le stage de build
COPY --from=builder /app/main .
COPY --from=builder /app/crypto.db .

# Créer un volume pour la base de données
VOLUME ["/app/data"]

# Exposer le port
EXPOSE 8080

# Lancer l'application
CMD ["./main"] 