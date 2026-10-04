
# syntax=docker/dockerfile:1.6
FROM node:20-alpine AS build

WORKDIR /app
ARG VITE_API_URL=/api
ENV VITE_API_URL=${VITE_API_URL}
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Unprivileged nginx: runs as uid 101 and listens on 8080 (not 80).
FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
# Already the base image's default; explicit so scanners (Trivy DS-0002) see it.
USER 101
CMD ["nginx", "-g", "daemon off;"]
