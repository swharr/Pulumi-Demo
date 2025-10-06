# Build an image using tiny image for the Node app to run in
FROM node:20-alpine

# Create non-root user
RUN addgroup -g 1001 -S nodejs && adduser -S nodejs -u 1001

WORKDIR /app

# Copy package files and install dependencies as root
COPY package.json package-lock.json* ./
RUN npm ci --only=production || npm i --only=production

# Copy application code
COPY server.js ./
COPY img ./img

# Change ownership to nodejs user
RUN chown -R nodejs:nodejs /app

# Switch to non-root user
USER nodejs

EXPOSE 3000
CMD ["npm", "start"]
