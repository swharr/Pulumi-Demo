# Build
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm install --omit=dev
COPY . ./

# Run
FROM node:20-alpine
ENV NODE_ENV=production
WORKDIR /app
COPY --from=build /app /app
USER 1001
EXPOSE 3000
CMD ["node", "server.js"]
