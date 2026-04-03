# Build Stage
FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* ./

RUN npm ci

COPY . .

RUN npm run build

# Production Stage
FROM nginx:alpine

# Remove default nginx user and create custom non-root user
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

RUN rm -rf /usr/share/nginx/html/*

COPY --from=builder /app/dist /usr/share/nginx/html

COPY nginx.conf.template /etc/nginx/conf.d/default.conf.template
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Change ownership of nginx directories to non-root user
RUN chown -R appuser:appgroup /usr/share/nginx/html /etc/nginx/conf.d /var/cache/nginx /var/log/nginx

RUN mkdir -p /etc/nginx/ssl/ && \
    chown -R appuser:appgroup /etc/nginx/ssl/ && \
    chmod -R 755 /etc/nginx/ssl/

RUN touch /var/run/nginx.pid && \
    chown -R appuser:appgroup /var/run/nginx.pid /run/nginx.pid

# Update nginx config to run as non-root user
RUN sed -i 's/user  nginx;/user  appuser;/' /etc/nginx/nginx.conf

ENV BACKEND_HOST="backend"
ENV BACKEND_PORT="3000"

# Switch to non-root user
USER appuser

EXPOSE 8080

CMD ["/docker-entrypoint.sh"]
