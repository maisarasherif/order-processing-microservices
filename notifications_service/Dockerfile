# Use official Python slim image (smaller than full Python image)
FROM python:3.11-slim

# Set working directory inside container
WORKDIR /app

# Install system dependencies for psycopg2 (PostgreSQL adapter)
RUN apt-get update && apt-get install -y \
    gcc \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first (for Docker layer caching)
# If requirements.txt doesn't change, this layer is cached
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Create non-root user for security (similar to Go services)
RUN useradd -m -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

# Expose port
EXPOSE 8083

# Health check (similar to Go services)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8083/health')" || exit 1

# Run the application using uvicorn
# --host 0.0.0.0 allows external connections
# --port 8083 matches our SERVICE_PORT
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8083"]