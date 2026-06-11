# OpsOne React dashboard — GreenNode AgentBase (nginx :8080, GET /health)
FROM node:20.18.1-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_BASE_URL
ARG VITE_DEV_AUTH_BYPASS=true
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_DEV_AUTH_BYPASS=$VITE_DEV_AUTH_BYPASS
RUN npm run build

FROM nginx:1.27.3-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
