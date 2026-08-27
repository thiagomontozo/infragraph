FROM node:22.22.3-alpine3.22 AS deps
WORKDIR /app
COPY package*.json ./
RUN --mount=type=secret,id=npm_ca,required=false sh -c 'if [ -f /run/secrets/npm_ca ]; then export NODE_EXTRA_CA_CERTS=/run/secrets/npm_ca; fi; npm ci'

FROM deps AS test
COPY . .
RUN npm run lint && npm run typecheck && npm run test:unit && npm run build

FROM deps AS build
COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.30.4-alpine3.24
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
USER 101:101
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8080/ || exit 1
