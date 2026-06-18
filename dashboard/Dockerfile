FROM node:20-alpine AS builder

WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Nginx serves static files
FROM nginx:alpine
COPY --from=builder /build/dist /usr/share/nginx/html
# Custom nginx config can be mounted at /etc/nginx/conf.d/default.conf
